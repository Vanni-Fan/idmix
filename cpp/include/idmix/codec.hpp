#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace idmix {

constexpr const char* DEFAULT_ALPHABET =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

class ICodec {
public:
    virtual ~ICodec() = default;
    virtual std::string encode(const std::vector<uint8_t>& data) const = 0;
    virtual std::vector<uint8_t> decode(const std::string& s) const = 0;
};

class RadixCodec : public ICodec {
public:
    explicit RadixCodec(const std::string& alphabet);
    std::string encode(const std::vector<uint8_t>& data) const override;
    std::vector<uint8_t> decode(const std::string& s) const override;
    int base() const { return base_; }
    const std::string& chars() const { return chars_; }

private:
    int base_;
    std::string chars_;
    std::vector<int> fromCustom_;
    std::string intToString(const std::vector<uint8_t>& n) const;
    std::vector<uint8_t> stringToInt(const std::string& s) const;
};

std::string encodeBytes(const std::vector<uint8_t>& data, const ICodec* codec = nullptr);
std::vector<uint8_t> decodeString(const std::string& s, const ICodec* codec = nullptr);

}  // namespace idmix
