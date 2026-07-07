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

static bool valuesEqual(const std::vector<Value>& a, const std::vector<Value>& b) {
    if (a.size() != b.size()) return false;
    for (size_t i = 0; i < a.size(); ++i) {
        if (a[i].index() != b[i].index()) return false;
        if (std::holds_alternative<std::string>(a[i])) {
            if (std::get<std::string>(a[i]) != std::get<std::string>(b[i])) return false;
        } else {
            if (!(std::get<TypedValue>(a[i]) == std::get<TypedValue>(b[i]))) return false;
        }
    }
    return true;
}

static void check(bool ok, const char* msg) {
    if (!ok) {
        std::cerr << "FAIL: " << msg << std::endl;
        std::exit(1);
    }
    std::cout << "OK: " << msg << std::endl;
}

static void roundTrip(IdMix& m, const std::vector<Value>& in, const char* name) {
    auto s = m.encode(in);
    auto out = m.decode(s);
    check(valuesEqual(out, in), name);
}

static void decodeVector(IdMix& m, const std::string& encoded, const std::vector<Value>& want,
                         const char* name) {
    auto out = m.decode(encoded);
    check(valuesEqual(out, want), name);
}

static void reEncode(IdMix& m, int variant, const std::vector<Value>& in, const std::string& want,
                     const char* name) {
    auto s = m.encodeWithVariant(variant, in);
    check(s == want, name);
}

int main() {
    IdMix m = IdMix::newDefault();
    std::vector<Value> typed = {TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)};
    auto data = m.idx().encodeWithVariant(0, typed);
    std::vector<uint8_t> wantBin = {0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F};
    check(data == wantBin, "spec example binary");

    roundTrip(m, typed, "round trip basic");

    roundTrip(m, {TypedValue::u32(EXTREME_UINT32_MAX)}, "uint32_max");
    roundTrip(m, {TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN))}, "int32_min");
    roundTrip(m, {TypedValue::i64(EXTREME_INT64_MIN)}, "int64_min");
    roundTrip(m, {TypedValue::i64(EXTREME_INT64_MAX)}, "int64_max");
    roundTrip(m, {TypedValue::u64(UINT64_MAX)}, "uint64_max");

    std::vector<Value> mixed = {
        TypedValue::u32(EXTREME_UINT32_MAX),
        TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN)),
        TypedValue::i64(EXTREME_INT64_MIN),
        TypedValue::i64(EXTREME_INT64_MAX),
    };
    roundTrip(m, mixed, "mixed_extremes");

    std::vector<Value> stringEx = {
        std::string("hello"),
        TypedValue::u16(5),
        std::string("\xe4\xb8\x96\xe7\x95\x8c"),
    };
    roundTrip(m, stringEx, "string_example");

    decodeVector(m, "ixHjl0FK7", typed, "cross-language spec_example");
    decodeVector(m, "hUdZLNKGa", {TypedValue::u32(EXTREME_UINT32_MAX)}, "cross-language uint32_max");
    decodeVector(m, "hUdElRoHP", {TypedValue::i32(static_cast<int>(EXTREME_INT32_MIN))},
                 "cross-language int32_min");
    decodeVector(m, "8B10qg6x0EAf3b", {TypedValue::i64(EXTREME_INT64_MIN)}, "cross-language int64_min");
    decodeVector(m, "8B2cU8kbWpQ2RM", {TypedValue::i64(EXTREME_INT64_MAX)}, "cross-language int64_max");
    decodeVector(m, "8B3CPRsv0Owa6S", {TypedValue::u64(UINT64_MAX)}, "cross-language uint64_max");
    decodeVector(m, "bULoRnNZJinEZGKD78wIigIaw6QplS8B0HGNCKO2L6", mixed, "cross-language mixed_extremes");
    decodeVector(m, "ceOqw5RPaTfgnfXyp7Sdepb", stringEx, "cross-language string_example");

    reEncode(m, 0, typed, "ixHjl0FK7", "re-encode spec_example");
    reEncode(m, 0, stringEx, "ceOqw5RPaTfgnfXyp7Sdepb", "re-encode string_example");

    IdMix m2("abcd");
    auto s2 = m2.encode({TypedValue::u16(100), TypedValue::i32(-10), TypedValue::u8(3)});
    auto out2 = m2.decode(s2);
    check(out2.size() == 3, "custom alphabet round trip");

    auto tamperedData = m.idx().encodeWithVariant(0, {TypedValue::u32(1)});
    tamperedData[1] ^= 0x01;
    auto tampered = m.codec().encode(tamperedData);
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

    auto raw = encodeBytes({0x08, 0x96, 0x01});
    auto back = decodeString(raw);
    check(back.size() == 3 && back[0] == 0x08 && back[1] == 0x96 && back[2] == 0x01,
          "encodeBytes/decodeString");

    std::cout << "All tests passed." << std::endl;
    return 0;
}
