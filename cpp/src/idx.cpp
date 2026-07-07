#include "idmix/idx.hpp"

#include <algorithm>
#include <cstdint>
#include <cstring>
#include <stdexcept>

namespace idmix {
namespace {

constexpr int SW_BYTES[] = {1, 2, 4, 8};
constexpr int EMBEDDED_OTYPE[2][4] = {
    {OTYPE_UINT8, OTYPE_UINT16, OTYPE_UINT32, OTYPE_UINT64},
    {OTYPE_INT8, OTYPE_INT16, OTYPE_INT32, OTYPE_INT64},
};

struct DataObject {
    bool isString = false;
    int otype = 0;
    int64_t val = 0;
    std::vector<uint8_t> str;
};

bool isUnsigned(int otype) { return otype <= OTYPE_UINT64; }

int widthBits(int otype) {
    switch (otype) {
        case OTYPE_UINT8:
        case OTYPE_INT8:
            return 0;
        case OTYPE_UINT16:
        case OTYPE_INT16:
            return 1;
        case OTYPE_UINT32:
        case OTYPE_INT32:
            return 2;
        default:
            return 3;
    }
}

void validateRange(int otype, int64_t val) {
    bool ok = false;
    switch (otype) {
        case OTYPE_UINT8:
            ok = val >= 0 && val <= 0xFF;
            break;
        case OTYPE_UINT16:
            ok = val >= 0 && val <= 0xFFFF;
            break;
        case OTYPE_UINT32:
            ok = val >= 0 && val <= 0xFFFFFFFFLL;
            break;
        case OTYPE_UINT64:
            ok = true;
            break;
        case OTYPE_INT8:
            ok = val >= -128 && val <= 127;
            break;
        case OTYPE_INT16:
            ok = val >= -32768 && val <= 32767;
            break;
        case OTYPE_INT32:
            ok = val >= INT32_MIN && val <= INT32_MAX;
            break;
        case OTYPE_INT64:
            ok = true;
            break;
    }
    if (!ok) throw std::invalid_argument("value out of range for otype");
}

std::vector<uint8_t> uintToLeBytes(uint64_t v, int size) {
    std::vector<uint8_t> buf(size);
    for (int i = 0; i < size; ++i) buf[i] = static_cast<uint8_t>((v >> (8 * i)) & 0xFF);
    return buf;
}

uint64_t leBytesToUint(const std::vector<uint8_t>& payload) {
    uint64_t u = 0;
    for (size_t i = 0; i < payload.size(); ++i) u |= static_cast<uint64_t>(payload[i]) << (8 * i);
    return u;
}

int64_t leBytesToSigned(const std::vector<uint8_t>& payload) {
    uint64_t u = leBytesToUint(payload);
    int shift = 64 - static_cast<int>(payload.size()) * 8;
    return static_cast<int64_t>(u << shift) >> shift;
}

uint8_t swFromMagnitude(uint64_t mag) {
    if (mag < 256) return 0;
    if (mag < 65536) return 1;
    if (mag < 4294967296ULL) return 2;
    return 3;
}

uint8_t swFromSignedValue(int64_t val) {
    if (val >= INT8_MIN && val <= INT8_MAX) return 0;
    if (val >= INT16_MIN && val <= INT16_MAX) return 1;
    if (val >= INT32_MIN && val <= INT32_MAX) return 2;
    return 3;
}

std::vector<uint8_t> signedToLeBytes(int64_t val, int size) {
    return uintToLeBytes(static_cast<uint64_t>(val), size);
}

struct MagNeg {
    uint64_t mag;
    bool neg;
};

MagNeg magnitudeFromTyped(int otype, int64_t val) {
    if (isUnsigned(otype)) return {static_cast<uint64_t>(val), false};
    if (val < 0) return {static_cast<uint64_t>(0 - static_cast<uint64_t>(val)), true};
    return {static_cast<uint64_t>(val), false};
}

bool tryEmbeddedHead(int otype, int64_t val, uint8_t* head) {
    auto mn = magnitudeFromTyped(otype, val);
    if (mn.mag >= 17) return false;
    int wb = widthBits(otype);
    if (mn.mag == 16) {
        if (mn.neg) {
            *head = static_cast<uint8_t>((1 << 6) | (wb << 4) | 15);
            return true;
        }
        return false;
    }
    if (mn.neg) {
        *head = static_cast<uint8_t>((1 << 6) | (wb << 4) | static_cast<int>(mn.mag - 1));
    } else {
        *head = static_cast<uint8_t>((wb << 4) | mn.mag);
    }
    return true;
}

int64_t valueFromPayload(int otype, const std::vector<uint8_t>& payload) {
    if (isUnsigned(otype)) {
        uint64_t mag = leBytesToUint(payload);
        if (otype != OTYPE_UINT64 && mag > static_cast<uint64_t>(INT64_MAX))
            throw std::invalid_argument("value out of range for otype");
        return static_cast<int64_t>(mag);
    }
    return leBytesToSigned(payload);
}

DataObject objectFromValue(const Value& v) {
    if (std::holds_alternative<std::string>(v)) {
        const auto& s = std::get<std::string>(v);
        if (s.empty()) throw std::invalid_argument("empty string is not allowed");
        if (s.size() > MAX_STRING_LEN)
            throw std::invalid_argument("string length exceeds max");
        DataObject obj;
        obj.isString = true;
        obj.str.assign(s.begin(), s.end());
        return obj;
    }
    const auto& tv = std::get<TypedValue>(v);
    validateRange(tv.otype, tv.val);
    DataObject obj;
    obj.otype = tv.otype;
    obj.val = tv.val;
    return obj;
}

Value materializeValue(const DataObject& obj) {
    if (obj.isString) return std::string(obj.str.begin(), obj.str.end());
    return TypedValue{obj.otype, obj.val};
}

std::vector<uint8_t> encodeObject(const DataObject& obj) {
    if (obj.isString) {
        if (obj.str.empty() || obj.str.size() > MAX_STRING_LEN)
            throw std::invalid_argument("invalid string length");
        std::vector<uint8_t> out(1 + obj.str.size());
        out[0] = static_cast<uint8_t>(0xC0 | obj.str.size());
        std::copy(obj.str.begin(), obj.str.end(), out.begin() + 1);
        return out;
    }
    validateRange(obj.otype, obj.val);
    uint8_t embedded;
    if (tryEmbeddedHead(obj.otype, obj.val, &embedded)) return {embedded};

    std::vector<uint8_t> payload;
    uint8_t sw;
    if (obj.otype == OTYPE_UINT64) {
        uint64_t mag = static_cast<uint64_t>(obj.val);
        sw = swFromMagnitude(mag);
        payload = uintToLeBytes(mag, SW_BYTES[sw]);
    } else if (isUnsigned(obj.otype)) {
        if (obj.val < 0) throw std::invalid_argument("negative value for unsigned otype");
        uint64_t mag = static_cast<uint64_t>(obj.val);
        sw = swFromMagnitude(mag);
        payload = uintToLeBytes(mag, SW_BYTES[sw]);
    } else {
        sw = swFromSignedValue(obj.val);
        payload = signedToLeBytes(obj.val, SW_BYTES[sw]);
    }
    std::vector<uint8_t> out(1 + payload.size());
    out[0] = static_cast<uint8_t>(0x80 | (sw << 4) | obj.otype);
    std::copy(payload.begin(), payload.end(), out.begin() + 1);
    return out;
}

struct DecodeResult {
    DataObject obj;
    size_t consumed;
};

DecodeResult decodeObject(const std::vector<uint8_t>& data, size_t offset) {
    if (offset >= data.size()) throw std::invalid_argument("truncated object header");
    uint8_t head = data[offset];
    if ((head & 0x80) == 0) {
        int sign = (head >> 6) & 1;
        int wb = (head >> 4) & 0x03;
        int v = head & 0x0F;
        int otype = EMBEDDED_OTYPE[sign][wb];
        int64_t val = sign == 0 ? v : -v - 1LL;
        return {{false, otype, val, {}}, 1};
    }
    if ((head & 0x40) != 0) {
        size_t n = head & 0x3F;
        if (n < 1 || n > MAX_STRING_LEN) throw std::invalid_argument("invalid string length");
        if (data.size() < offset + 1 + n) throw std::invalid_argument("truncated string payload");
        DataObject obj;
        obj.isString = true;
        obj.str.assign(data.begin() + offset + 1, data.begin() + offset + 1 + n);
        return {obj, 1 + n};
    }
    int sw = (head >> 4) & 0x03;
    int otype = head & 0x0F;
    if (otype > OTYPE_INT64) throw std::invalid_argument("invalid otype");
    int numBytes = SW_BYTES[sw];
    if (data.size() < offset + 1 + static_cast<size_t>(numBytes))
        throw std::invalid_argument("truncated object payload");
    std::vector<uint8_t> payload(data.begin() + offset + 1, data.begin() + offset + 1 + numBytes);
    int64_t val = valueFromPayload(otype, payload);
    validateRange(otype, val);
    return {{false, otype, val, {}}, static_cast<size_t>(1 + numBytes)};
}

}  // namespace

Idx::Idx() : Idx(255, 32, 2) {}

Idx::Idx(int maxObjects, int maxVariants, int checkBits)
    : maxObjects_(maxObjects), maxVariants_(maxVariants), checkBits_(checkBits) {
    if (maxObjects < 1 || maxObjects > 255) throw std::invalid_argument("maxObjects out of range");
    if (maxVariants < 1 || maxVariants > 32) throw std::invalid_argument("maxVariants out of range");
    if (checkBits < 1 || checkBits > 2) throw std::invalid_argument("checkBits must be 1 or 2");
    checkMask_ = (1 << checkBits_) - 1;
}

std::vector<uint8_t> Idx::encodeWithVariant(int variantId, const std::vector<Value>& values) const {
    if (values.empty()) throw std::invalid_argument("at least one value is required");
    if (static_cast<int>(values.size()) > maxObjects_) throw std::invalid_argument("too many objects");
    if (variantId < 0 || variantId >= maxVariants_)
        throw std::invalid_argument("invalid variant_id");

    std::vector<uint8_t> objBytes;
    for (const auto& v : values) {
        auto obj = objectFromValue(v);
        auto ob = encodeObject(obj);
        objBytes.insert(objBytes.end(), ob.begin(), ob.end());
    }

    uint8_t mask = static_cast<uint8_t>((variantId * 0x9D + 0x37) & 0xFF);
    for (auto& b : objBytes) b ^= mask;

    int count = static_cast<int>(values.size());
    int headerLen = count == 1 ? 1 : 2;
    std::vector<uint8_t> data(headerLen + objBytes.size());
    if (count == 1) {
        data[0] = static_cast<uint8_t>(variantId << checkBits_);
    } else {
        data[0] = static_cast<uint8_t>(0x80 | (variantId << checkBits_));
        data[1] = static_cast<uint8_t>(count);
    }
    std::copy(objBytes.begin(), objBytes.end(), data.begin() + headerLen);

    int xorSum = 0;
    for (uint8_t b : data) xorSum ^= b;
    data[0] |= static_cast<uint8_t>(xorSum & checkMask_);
    return data;
}

std::vector<uint8_t> Idx::encode(const std::vector<Value>& values) const {
    return encodeWithVariant(0, values);
}

std::vector<Value> Idx::decode(const std::vector<uint8_t>& data) const {
    if (data.empty()) throw std::invalid_argument("invalid data: too short");

    uint8_t byte0 = data[0];
    uint8_t check = static_cast<uint8_t>(byte0 & checkMask_);
    bool multi = (byte0 & 0x80) != 0;
    int variantId = (byte0 & 0x7F) >> checkBits_;
    if (variantId >= maxVariants_) throw std::invalid_argument("invalid variant_id");

    int headerLen = 1;
    int count = 1;
    if (multi) {
        if (data.size() < 2) throw std::invalid_argument("invalid data: missing count byte");
        headerLen = 2;
        count = data[1];
        if (count < 2 || count > maxObjects_) throw std::invalid_argument("invalid count");
    }

    std::vector<uint8_t> verify = data;
    verify[0] &= static_cast<uint8_t>(~checkMask_);
    int xorSum = 0;
    for (uint8_t b : verify) xorSum ^= b;
    if ((xorSum & checkMask_) != check) throw std::invalid_argument("checksum mismatch");

    std::vector<uint8_t> objData(data.begin() + headerLen, data.end());
    uint8_t mask = static_cast<uint8_t>((variantId * 0x9D + 0x37) & 0xFF);
    for (auto& b : objData) b ^= mask;

    std::vector<Value> result;
    size_t pos = 0;
    for (int i = 0; i < count; ++i) {
        if (pos >= objData.size()) throw std::invalid_argument("premature end of data");
        auto dr = decodeObject(objData, pos);
        result.push_back(materializeValue(dr.obj));
        pos += dr.consumed;
    }
    if (pos != objData.size()) throw std::invalid_argument("extra bytes after data objects");
    return result;
}

}  // namespace idmix
