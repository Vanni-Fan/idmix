#include "idmix/idmix.hpp"

#include <random>
#include <stdexcept>
#include <unordered_map>

namespace idmix {

namespace {

int bitLen(int n) {
    if (n <= 0) return 1;
    int bits = 0;
    while (n > 0) {
        n >>= 1;
        ++bits;
    }
    return bits;
}

}  // namespace

RadixCodec::RadixCodec(const std::string& alphabet) : base_(static_cast<int>(alphabet.size())), chars_(alphabet) {
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

std::string RadixCodec::encodeBytes(const std::vector<uint8_t>& data) const {
    if (data.empty()) return std::string(1, chars_[0]);
    std::vector<uint8_t> wrapped;
    wrapped.push_back(static_cast<uint8_t>((data.size() >> 8) & 0xFF));
    wrapped.push_back(static_cast<uint8_t>(data.size() & 0xFF));
    wrapped.insert(wrapped.end(), data.begin(), data.end());
    return intToString(wrapped);
}

std::vector<uint8_t> RadixCodec::decodeBytes(const std::string& s) const {
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

IdMix::IdMix() : IdMix(DEFAULT_ALPHABET, 511, 32, 2) {}

IdMix::IdMix(const std::string& alphabet) : IdMix(alphabet, 511, 32, 2) {}

IdMix::IdMix(const std::string& alphabet, int maxObjects, int maxVariants, int checkBits)
    : radix_(alphabet),
      maxObjects_(maxObjects),
      maxVariants_(maxVariants),
      checkBits_(checkBits) {
    finalizeLayout();
}

void IdMix::finalizeLayout() {
    int variantBits = maxVariants_ <= 1 ? 1 : bitLen(maxVariants_ - 1);
    int countBits = maxObjects_ <= 1 ? 1 : bitLen(maxObjects_);
    int total = checkBits_ + countBits + variantBits;
    if (total > 16) throw std::invalid_argument("header layout exceeds 16 bits");
    countBits_ = countBits;
    variantBits_ = variantBits;
    checkMask_ = (1 << checkBits_) - 1;
    countMask_ = ((1 << countBits_) - 1) << checkBits_;
    variantMask_ = ((1 << variantBits_) - 1) << (checkBits_ + countBits_);
    countShift_ = checkBits_;
    variantShift_ = checkBits_ + countBits_;
}

std::string IdMix::encode(const std::vector<TypedValue>& values) {
    if (values.empty()) throw std::invalid_argument("at least one value is required");
    if (static_cast<int>(values.size()) > maxObjects_) throw std::invalid_argument("too many objects");
    static thread_local std::mt19937 rng{std::random_device{}()};
    std::uniform_int_distribution<int> dist(0, maxVariants_ - 1);
    int variantId = dist(rng);
    auto data = xid_codec::encodeBinary(*this, values, variantId);
    return radix_.encodeBytes(data);
}

std::vector<TypedValue> IdMix::decode(const std::string& s) {
    auto data = radix_.decodeBytes(s);
    return xid_codec::decodeBinary(*this, data);
}

}  // namespace idmix
