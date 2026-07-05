#include "idmix/idmix.hpp"

#include <cstdlib>
#include <iostream>
#include <set>
#include <sstream>

using namespace idmix;

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

int main() {
    IdMix m = IdMix::newDefault();
    std::vector<TypedValue> typed = {TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)};
    auto data = xid_codec::encodeBinary(m, typed, 0);
    std::vector<uint8_t> want = {0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F};
    check(data == want, "spec example binary");

    auto s = m.encode(typed);
    auto out = m.decode(s);
    check(out.size() == 3 && out[0] == typed[0] && out[1] == typed[1] && out[2] == typed[2],
          "round trip basic");

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
