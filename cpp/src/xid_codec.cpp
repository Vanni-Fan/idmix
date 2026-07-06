#include "idmix/idmix.hpp"

#include <algorithm>
#include <stdexcept>

namespace idmix {
namespace xid_codec {

namespace {

constexpr int SW_BYTES[] = {1, 2, 4, 8};
constexpr int EMBEDDED_OTYPE[2][4] = {
    {OTYPE_UINT8, OTYPE_UINT16, OTYPE_UINT32, OTYPE_UINT64},
    {OTYPE_INT8, OTYPE_INT16, OTYPE_INT32, OTYPE_INT64},
};

bool isUnsigned(int otype) { return otype <= OTYPE_UINT64; }
bool isSigned(int otype) { return otype >= OTYPE_INT8; }

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

int targetBits(int otype) {
    switch (otype) {
        case OTYPE_UINT8:
        case OTYPE_INT8:
            return 8;
        case OTYPE_UINT16:
        case OTYPE_INT16:
            return 16;
        case OTYPE_UINT32:
        case OTYPE_INT32:
            return 32;
        default:
            return 64;
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

int64_t valueFromMagnitude(uint64_t mag, bool neg) {
    if (!neg) return static_cast<int64_t>(mag);
    if (mag == 1ULL << 63) return INT64_MIN;
    return -static_cast<int64_t>(mag);
}

uint8_t swFromMagnitude(uint64_t mag) {
    if (mag < 256) return 0;
    if (mag < 65536) return 1;
    if (mag < 4294967296ULL) return 2;
    return 3;
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

std::vector<uint8_t> encodeObject(const TypedValue& tv) {
    validateRange(tv.otype, tv.val);
    uint8_t embedded;
    if (tryEmbeddedHead(tv.otype, tv.val, &embedded)) {
        return {embedded};
    }
    auto mn = magnitudeFromTyped(tv.otype, tv.val);
    int sw = swFromMagnitude(mn.mag);
    auto payload = uintToLeBytes(mn.mag, SW_BYTES[sw]);
    uint8_t head = static_cast<uint8_t>(0x80 | (sw << 4) | tv.otype);
    if (mn.neg) head |= 1 << 6;
    std::vector<uint8_t> out(1 + payload.size());
    out[0] = head;
    std::copy(payload.begin(), payload.end(), out.begin() + 1);
    return out;
}

struct DecodeResult {
    TypedValue tv;
    int consumed;
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
        return {{otype, val}, 1};
    }
    int sw = (head >> 4) & 0x03;
    int otype = head & 0x0F;
    if (otype > OTYPE_INT64) throw std::invalid_argument("invalid otype");
    int numBytes = SW_BYTES[sw];
    if (data.size() < offset + 1 + static_cast<size_t>(numBytes))
        throw std::invalid_argument("truncated object payload");
    uint64_t mag = 0;
    for (int i = 0; i < numBytes; ++i) mag |= static_cast<uint64_t>(data[offset + 1 + i]) << (8 * i);
    bool neg = ((head >> 6) & 1) != 0;
    int64_t val = valueFromMagnitude(mag, neg);
    validateRange(otype, val);
    return {{otype, val}, 1 + numBytes};
}

}  // namespace

std::vector<uint8_t> encodeBinary(const IdMix& m, const std::vector<TypedValue>& typed, int variantId) {
    std::vector<uint8_t> objects;
    for (const auto& tv : typed) {
        auto obj = encodeObject(tv);
        objects.insert(objects.end(), obj.begin(), obj.end());
    }
    uint8_t mask = static_cast<uint8_t>((variantId * 0x9D + 0x37) & 0xFF);
    for (auto& b : objects) b ^= mask;

    int count = static_cast<int>(typed.size());
    int header = (variantId << m.variantShift()) | (count << m.countShift());
    std::vector<uint8_t> data(2 + objects.size());
    data[0] = static_cast<uint8_t>(header & 0xFF);
    data[1] = static_cast<uint8_t>((header >> 8) & 0xFF);
    std::copy(objects.begin(), objects.end(), data.begin() + 2);

    int xorSum = 0;
    for (uint8_t b : data) xorSum ^= b;
    header |= xorSum & m.checkMask();
    data[0] = static_cast<uint8_t>(header & 0xFF);
    data[1] = static_cast<uint8_t>((header >> 8) & 0xFF);
    return data;
}

std::vector<TypedValue> decodeBinary(const IdMix& m, const std::vector<uint8_t>& data) {
    if (data.size() < 2) throw std::invalid_argument("invalid data: too short");
    int header = data[0] | (data[1] << 8);
    int check = header & m.checkMask();
    int count = (header & m.countMask()) >> m.countShift();
    int variantId = (header & m.variantMask()) >> m.variantShift();

    if (variantId >= m.maxVariants()) throw std::invalid_argument("invalid variant_id");
    if (count > m.maxObjects()) throw std::invalid_argument("invalid count");

    std::vector<uint8_t> verify = data;
    verify[0] &= static_cast<uint8_t>(~m.checkMask());
    int xorSum = 0;
    for (uint8_t b : verify) xorSum ^= b;
    if ((xorSum & m.checkMask()) != check) throw std::invalid_argument("checksum mismatch");

    std::vector<uint8_t> objects(data.begin() + 2, data.end());
    uint8_t mask = static_cast<uint8_t>((variantId * 0x9D + 0x37) & 0xFF);
    for (auto& b : objects) b ^= mask;

    std::vector<TypedValue> result;
    size_t pos = 0;
    for (int i = 0; i < count; ++i) {
        if (pos >= objects.size()) throw std::invalid_argument("premature end of data");
        auto dr = decodeObject(objects, pos);
        result.push_back(dr.tv);
        pos += static_cast<size_t>(dr.consumed);
    }
    if (pos != objects.size()) throw std::invalid_argument("extra bytes after data objects");
    return result;
}

}  // namespace xid_codec
}  // namespace idmix
