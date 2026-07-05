#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace idmix {

constexpr const char* DEFAULT_ALPHABET =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

enum OType : int {
    OTYPE_UINT8 = 0,
    OTYPE_UINT16 = 1,
    OTYPE_UINT32 = 2,
    OTYPE_UINT64 = 3,
    OTYPE_INT8 = 4,
    OTYPE_INT16 = 5,
    OTYPE_INT32 = 6,
    OTYPE_INT64 = 7,
};

struct TypedValue {
    int otype;
    int64_t val;

    static TypedValue u8(int v) { return {OTYPE_UINT8, v}; }
    static TypedValue u16(int v) { return {OTYPE_UINT16, v}; }
    static TypedValue u32(int64_t v) { return {OTYPE_UINT32, v}; }
    static TypedValue u64(int64_t v) { return {OTYPE_UINT64, v}; }
    static TypedValue i8(int v) { return {OTYPE_INT8, v}; }
    static TypedValue i16(int v) { return {OTYPE_INT16, v}; }
    static TypedValue i32(int v) { return {OTYPE_INT32, v}; }
    static TypedValue i64(int64_t v) { return {OTYPE_INT64, v}; }

    bool operator==(const TypedValue& o) const { return otype == o.otype && val == o.val; }
};

class RadixCodec {
public:
    explicit RadixCodec(const std::string& alphabet);
    std::string encodeBytes(const std::vector<uint8_t>& data) const;
    std::vector<uint8_t> decodeBytes(const std::string& s) const;
    int base() const { return base_; }
    const std::string& chars() const { return chars_; }

private:
    int base_;
    std::string chars_;
    std::vector<int> fromCustom_;
    std::string intToString(const std::vector<uint8_t>& n) const;
    std::vector<uint8_t> stringToInt(const std::string& s) const;
};

class IdMix {
public:
    IdMix();
    explicit IdMix(const std::string& alphabet);
    IdMix(const std::string& alphabet, int maxObjects, int maxVariants, int checkBits);

    static IdMix newDefault() { return IdMix(); }

    std::string encode(const std::vector<TypedValue>& values);
    std::vector<TypedValue> decode(const std::string& s);

    const RadixCodec& radix() const { return radix_; }
    int maxObjects() const { return maxObjects_; }
    int maxVariants() const { return maxVariants_; }
    int checkBits() const { return checkBits_; }
    int countBits() const { return countBits_; }
    int variantBits() const { return variantBits_; }
    int checkMask() const { return checkMask_; }
    int countMask() const { return countMask_; }
    int variantMask() const { return variantMask_; }
    int countShift() const { return countShift_; }
    int variantShift() const { return variantShift_; }

private:
    RadixCodec radix_;
    int maxObjects_;
    int maxVariants_;
    int checkBits_;
    int countBits_;
    int variantBits_;
    int checkMask_;
    int countMask_;
    int variantMask_;
    int countShift_;
    int variantShift_;
    void finalizeLayout();
};

namespace xid_codec {
std::vector<uint8_t> encodeBinary(const IdMix& m, const std::vector<TypedValue>& typed, int variantId);
std::vector<TypedValue> decodeBinary(const IdMix& m, const std::vector<uint8_t>& data);
}  // namespace xid_codec

}  // namespace idmix
