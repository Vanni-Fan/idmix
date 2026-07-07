#include "idmix/codec.hpp"

#include <algorithm>
#include <memory>
#include <stdexcept>

namespace idmix {

namespace {

const RadixCodec& defaultCodecInstance() {
    static const RadixCodec codec(DEFAULT_ALPHABET);
    return codec;
}

const ICodec& resolveCodec(const ICodec* codec) {
    if (codec) return *codec;
    return defaultCodecInstance();
}

}  // namespace

RadixCodec::RadixCodec(const std::string& alphabet)
    : base_(static_cast<int>(alphabet.size())), chars_(alphabet) {
    if (alphabet.size() < 2) throw std::invalid_argument("alphabet must have at least 2 unique characters");
    fromCustom_.assign(256, -1);
    for (size_t i = 0; i < alphabet.size(); ++i) {
        unsigned char c = static_cast<unsigned char>(alphabet[i]);
        if (fromCustom_[c] >= 0) throw std::invalid_argument("alphabet contains duplicate character");
        fromCustom_[c] = static_cast<int>(i);
    }
}

std::string RadixCodec::intToString(const std::vector<uint8_t>& bytes) const {
    if (bytes.empty() || (bytes.size() == 1 && bytes[0] == 0)) return std::string(1, chars_[0]);
    std::vector<uint8_t> n = bytes;
    std::string out;
    while (!(n.size() == 1 && n[0] == 0)) {
        std::vector<uint8_t> quot;
        int rem = 0;
        for (uint8_t b : n) {
            int cur = rem * 256 + b;
            quot.push_back(static_cast<uint8_t>(cur / base_));
            rem = cur % base_;
        }
        while (!quot.empty() && quot[0] == 0) quot.erase(quot.begin());
        out.push_back(chars_[rem]);
        n.swap(quot);
        if (n.empty()) n.push_back(0);
    }
    std::reverse(out.begin(), out.end());
    return out;
}

std::vector<uint8_t> RadixCodec::stringToInt(const std::string& s) const {
    std::vector<uint8_t> n = {0};
    for (char ch : s) {
        unsigned char c = static_cast<unsigned char>(ch);
        if (c >= fromCustom_.size() || fromCustom_[c] < 0)
            throw std::invalid_argument("invalid character");
        int idx = fromCustom_[c];
        int carry = 0;
        for (auto it = n.rbegin(); it != n.rend(); ++it) {
            int cur = (*it) * base_ + carry;
            if (it == n.rbegin()) cur += idx;
            *it = static_cast<uint8_t>(cur & 0xFF);
            carry = cur >> 8;
        }
        while (carry > 0) {
            n.insert(n.begin(), static_cast<uint8_t>(carry & 0xFF));
            carry >>= 8;
        }
    }
    while (n.size() > 1 && n[0] == 0) n.erase(n.begin());
    return n;
}

std::string RadixCodec::encode(const std::vector<uint8_t>& data) const {
    if (data.empty()) return std::string(1, chars_[0]);
    std::vector<uint8_t> wrapped;
    wrapped.push_back(static_cast<uint8_t>((data.size() >> 8) & 0xFF));
    wrapped.push_back(static_cast<uint8_t>(data.size() & 0xFF));
    wrapped.insert(wrapped.end(), data.begin(), data.end());
    return intToString(wrapped);
}

std::vector<uint8_t> RadixCodec::decode(const std::string& s) const {
    if (s.empty()) throw std::invalid_argument("empty string");
    auto raw = stringToInt(s);
    for (int pad = 0; pad <= 1; ++pad) {
        std::vector<uint8_t> buf(pad + raw.size());
        std::copy(raw.begin(), raw.end(), buf.begin() + pad);
        if (buf.size() < 2) continue;
        int dataLen = (buf[0] << 8) | buf[1];
        if (static_cast<int>(buf.size()) != 2 + dataLen) continue;
        return std::vector<uint8_t>(buf.begin() + 2, buf.end());
    }
    throw std::invalid_argument("invalid encoded data length");
}

std::string encodeBytes(const std::vector<uint8_t>& data, const ICodec* codec) {
    return resolveCodec(codec).encode(data);
}

std::vector<uint8_t> decodeString(const std::string& s, const ICodec* codec) {
    return resolveCodec(codec).decode(s);
}

}  // namespace idmix
