#pragma once

#include <cstdint>
#include <stdexcept>
#include <string>
#include <variant>
#include <vector>

namespace idmix {

constexpr int MAX_STRING_LEN = 63;

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

using Value = std::variant<TypedValue, std::string>;

class Idx {
public:
    Idx();
    Idx(int maxObjects, int maxVariants, int checkBits);

    int maxObjects() const { return maxObjects_; }
    int maxVariants() const { return maxVariants_; }
    int checkBits() const { return checkBits_; }

    std::vector<uint8_t> encode(const std::vector<Value>& values) const;
    std::vector<uint8_t> encodeWithVariant(int variantId, const std::vector<Value>& values) const;
    std::vector<Value> decode(const std::vector<uint8_t>& data) const;

private:
    int maxObjects_;
    int maxVariants_;
    int checkBits_;
    int checkMask_;
};

}  // namespace idmix
