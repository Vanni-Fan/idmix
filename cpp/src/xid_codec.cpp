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
            ok = val >= 0;
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

int64_t reconstructInt(int otype, int sw, uint64_t raw) {
    int tbits = targetBits(otype);
    int storedBits = SW_BYTES[sw] * 8;
    if (isUnsigned(otype)) {
        uint64_t mask = tbits == 64 ? ~0ULL : ((1ULL << tbits) - 1);
        return static_cast<int64_t>(raw & mask);
    }
    int signBit = static_cast<int>((raw >> (storedBits - 1)) & 1);
    if (tbits <= storedBits) {
        uint64_t mask = (1ULL << tbits) - 1;
        int64_t val = static_cast<int64_t>(raw & mask);
        if (signBit == 1 && (val & (1LL << (tbits - 1)))) val -= 1LL << tbits;
        return val;
    }
    int64_t extended;
    if (signBit == 1) {
        uint64_t extendMask = (~((1ULL << storedBits) - 1)) & ((1ULL << tbits) - 1);
        extended = static_cast<int64_t>(raw | extendMask);
    } else {
        extended = static_cast<int64_t>(raw);
    }
    if (extended >= (1LL << (tbits - 1))) extended -= 1LL << tbits;
    return extended;
}

struct SwPayload {
    int sw;
    std::vector<uint8_t> payload;
};

SwPayload minimalComplementBytes(int otype, int64_t val) {
    if (val == 0) return {0, {0}};
    if (isUnsigned(otype)) {
        if (val < 0) throw std::invalid_argument("negative value for unsigned type");
        auto uval = static_cast<uint64_t>(val);
        for (int sw = 0; sw < 4; ++sw) {
            int size = SW_BYTES[sw];
            if (size < 8 && uval >= (1ULL << (size * 8))) continue;
            auto buf = uintToLeBytes(uval, size);
            if ((buf[size - 1] & 0x80) == 0) return {sw, buf};
        }
        throw std::invalid_argument("value too large for unsigned type");
    }
    int tbits = targetBits(otype);
    uint64_t mask = tbits == 64 ? ~0ULL : ((1ULL << tbits) - 1);
    uint64_t uval = static_cast<uint64_t>(val) & mask;

    if (val < 0) {
        for (int sw = 0; sw < 4; ++sw) {
            int size = SW_BYTES[sw];
            int shift = size * 8;
            if (shift >= tbits) return {sw, uintToLeBytes(uval, size)};
            uint64_t lower = uval & ((1ULL << shift) - 1);
            uint64_t upper = uval >> shift;
            uint64_t upperMask = (1ULL << (tbits - shift)) - 1;
            if (upper != upperMask) continue;
            int highByte = static_cast<int>((lower >> (shift - 8)) & 0xFF);
            if ((highByte & 0x80) == 0) continue;
            return {sw, uintToLeBytes(lower, size)};
        }
    } else {
        for (int sw = 0; sw < 4; ++sw) {
            int size = SW_BYTES[sw];
            if (size < 8 && uval >= (1ULL << (size * 8))) continue;
            auto buf = uintToLeBytes(uval, size);
            if ((buf[size - 1] & 0x80) == 0) return {sw, buf};
        }
    }
    int swFinal = 3;
    if (tbits == 8) swFinal = 0;
    else if (tbits == 16) swFinal = 1;
    else if (tbits == 32) swFinal = 2;
    return {swFinal, uintToLeBytes(uval, SW_BYTES[swFinal])};
}

std::vector<uint8_t> encodeObject(const TypedValue& tv) {
    validateRange(tv.otype, tv.val);
    if (isUnsigned(tv.otype) && tv.val >= 0 && tv.val <= 15) {
        int wb = widthBits(tv.otype);
        return {static_cast<uint8_t>((wb << 4) | tv.val)};
    }
    if (isSigned(tv.otype) && tv.val >= -16 && tv.val <= -1) {
        int wb = widthBits(tv.otype);
        int v = static_cast<int>(-tv.val - 1);
        return {static_cast<uint8_t>((1 << 6) | (wb << 4) | v)};
    }
    auto sp = minimalComplementBytes(tv.otype, tv.val);
    std::vector<uint8_t> out(1 + sp.payload.size());
    out[0] = static_cast<uint8_t>(0x80 | (sp.sw << 4) | tv.otype);
    std::copy(sp.payload.begin(), sp.payload.end(), out.begin() + 1);
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
    if (((head >> 6) & 1) != 0) throw std::invalid_argument("reserved bit set in extended mode");
    int sw = (head >> 4) & 0x03;
    int otype = head & 0x0F;
    if (otype > OTYPE_INT64) throw std::invalid_argument("invalid otype");
    int numBytes = SW_BYTES[sw];
    if (data.size() < offset + 1 + static_cast<size_t>(numBytes))
        throw std::invalid_argument("truncated object payload");
    uint64_t raw = 0;
    for (int i = 0; i < numBytes; ++i) raw |= static_cast<uint64_t>(data[offset + 1 + i]) << (8 * i);
    int64_t val = reconstructInt(otype, sw, raw);
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
