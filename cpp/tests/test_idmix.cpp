#include "idmix/idmix.hpp"

#include <cstdlib>
#include <iostream>
#include <set>
#include <sstream>

using namespace idmix;

static constexpr int64_t EXTREME_UINT32_MAX = 4294967295LL;
static constexpr int64_t EXTREME_INT32_MIN = -2147483648LL;
static constexpr int64_t EXTREME_INT64_MIN = INT64_MIN;
static constexpr int64_t EXTREME_INT64_MAX = INT64_MAX;

static std::string hex(const std::vector<uint8_t>& b) {
    std::ostringstream os;
    for (size_t i = 0; i < b.size(); ++i) {
        if (i) os << ' ';
        os << std::hex << std::uppercase << (b[i] >> 4) << (b[i] & 0xF);
    }
    return b.empty() ? "(empty)" : os.str();
}

static void check(bool ok, const char* msg) {
    if (!ok) {
        std::cerr << "FAIL: " << msg << std::endl;
        std::exit(1);
    }
    std::cout << "OK: " << msg << std::endl;
}

static void roundTrip(IdMix& m, const std::vector<TypedValue>& in, const char* name) {
    auto s = m.encode(in);
    auto out = m.decode(s);
    check(out == in, name);
}

static void decodeVector(IdMix& m, const std::string& encoded, const std::vector<TypedValue>& want,
                         const char* name) {
    auto out = m.decode(encoded);
    check(out == want, name);
}

int main() {
    IdMix m = IdMix::newDefault();
    std::vector<TypedValue> typed = {TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)};
    auto data = xid_codec::encodeBinary(m, typed, 0);
    std::vector<uint8_t> want = {0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F};
    check(data == want, "spec example binary");

    roundTrip(m, typed, "round trip basic");

    roundTrip(m, {TypedValue::u32(EXTREME_UINT32_MAX)}, "uint32_max");
    roundTrip(m, {TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN))}, "int32_min");
    roundTrip(m, {TypedValue::i64(EXTREME_INT64_MIN)}, "int64_min");
    roundTrip(m, {TypedValue::i64(EXTREME_INT64_MAX)}, "int64_max");
    roundTrip(m, {TypedValue::u64(UINT64_MAX)}, "uint64_max");

    std::vector<TypedValue> mixed = {
        TypedValue::u32(EXTREME_UINT32_MAX),
        TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN)),
        TypedValue::i64(EXTREME_INT64_MIN),
        TypedValue::i64(EXTREME_INT64_MAX),
    };
    roundTrip(m, mixed, "mixed_extremes");

    decodeVector(m, "hYpGvRq6B", typed, "cross-language spec_example");
    decodeVector(m, "LwMDzFPIwK", {TypedValue::u32(EXTREME_UINT32_MAX)}, "cross-language uint32_max");
    decodeVector(m, "LwMH4is20x", {TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN))}, "cross-language int32_min");
    decodeVector(m, "eA3BqyCfeJ73bad1", {TypedValue::i64(EXTREME_INT64_MIN)}, "cross-language int64_min");
    decodeVector(m, "bTcNSaewCwrxPlc5fGCbq11xnBz120cpBTJ1A6ztNY", mixed, "cross-language mixed_extremes");

    IdMix m2("abcd");
    auto s2 = m2.encode({TypedValue::u16(100), TypedValue::i32(-10), TypedValue::u8(3)});
    auto out2 = m2.decode(s2);
    check(out2.size() == 3, "custom alphabet round trip");

    auto tamperedData = xid_codec::encodeBinary(m, {TypedValue::u32(1)}, 0);
    tamperedData[2] ^= 0x01;
    auto tampered = m.radix().encodeBytes(tamperedData);
    bool threw = false;
    try {
        m.decode(tampered);
    } catch (const std::exception&) {
        threw = true;
    }
    check(threw, "checksum rejects");

    std::set<std::string> seen;
    for (int i = 0; i < 50; ++i) seen.insert(m.encode({TypedValue::u32(42)}));
    check(seen.size() >= 2, "multiple encodings differ");

    std::cout << "All tests passed." << std::endl;
    return 0;
}
