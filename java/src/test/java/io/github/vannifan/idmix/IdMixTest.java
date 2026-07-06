package io.github.vannifan.idmix;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

class IdMixTest {

    private static final long EXTREME_UINT32_MAX = 4294967295L;
    private static final int EXTREME_INT32_MIN = -2147483648;
    private static final long EXTREME_INT64_MIN = Long.MIN_VALUE;
    private static final long EXTREME_INT64_MAX = Long.MAX_VALUE;
    private static final long EXTREME_UINT64_MAX_BITS = -1L;

    private static String hex(byte[] b) {
        if (b == null || b.length == 0) return "(empty)";
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < b.length; i++) {
            if (i > 0) sb.append(' ');
            sb.append(String.format("%02X", b[i]));
        }
        return sb.toString();
    }

    @Test
    @DisplayName("spec example binary block variant=0")
    void specExampleBinary() {
        IdMix m = IdMix.newDefault();
        List<TypedValue> typed = List.of(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
        byte[] data = XidCodec.encodeBinary(m, typed, 0);
        byte[] want = new byte[]{0x0F, 0x00, 0x22, 0x47, (byte) 0xB5, 0x1F};
        assertArrayEquals(want, data);
    }

    @Test
    @DisplayName("round trip u16(5), i64(-1), u32(40)")
    void roundTripBasic() {
        IdMix m = IdMix.newDefault();
        String encoded = m.encode(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
        List<TypedValue> decoded = m.decode(encoded);
        assertEquals(TypedValue.u16(5), decoded.get(0));
        assertEquals(TypedValue.i64(-1), decoded.get(1));
        assertEquals(TypedValue.u32(40), decoded.get(2));
    }

    @Test
    @DisplayName("round trip u32(2000000000)")
    void roundTripUint32Large() {
        IdMix m = IdMix.newDefault();
        List<TypedValue> out = List.of(m.decode(m.encode(TypedValue.u32(2_000_000_000L))).get(0));
        assertEquals(2_000_000_000L, out.get(0).val);
    }

    @Test
    @DisplayName("custom alphabet abcd")
    void customAlphabet() {
        IdMix m = new IdMix("abcd");
        List<TypedValue> decoded = m.decode(m.encode(
                TypedValue.u16(100), TypedValue.i32(-10), TypedValue.u8(3)));
        assertEquals(3, decoded.size());
    }

    @Test
    @DisplayName("checksum mismatch rejects decode")
    void checksumRejects() {
        IdMix m = IdMix.newDefault();
        byte[] data = XidCodec.encodeBinary(m, List.of(TypedValue.u32(1)), 0);
        data[2] ^= 0x01;
        String tampered = m.getRadix().encodeBytes(data);
        assertThrows(IllegalArgumentException.class, () -> m.decode(tampered));
    }

    @Test
    @DisplayName("variant polymorphism")
    void multipleEncodingsDiffer() {
        IdMix m = IdMix.newDefault();
        Set<String> seen = new HashSet<>();
        for (int i = 0; i < 50; i++) seen.add(m.encode(TypedValue.u32(42)));
        assertTrue(seen.size() >= 2);
    }

    @Test
    @DisplayName("extreme values round trip")
    void extremeValuesRoundTrip() {
        IdMix m = IdMix.newDefault();
        assertEquals(EXTREME_UINT32_MAX, m.decode(m.encode(TypedValue.u32(EXTREME_UINT32_MAX))).get(0).val);
        assertEquals(EXTREME_INT32_MIN, m.decode(m.encode(TypedValue.i32(EXTREME_INT32_MIN))).get(0).val);
        assertEquals(EXTREME_INT64_MIN, m.decode(m.encode(TypedValue.i64(EXTREME_INT64_MIN))).get(0).val);
        assertEquals(EXTREME_INT64_MAX, m.decode(m.encode(TypedValue.i64(EXTREME_INT64_MAX))).get(0).val);
        assertEquals(EXTREME_UINT64_MAX_BITS, m.decode(m.encode(TypedValue.u64("18446744073709551615"))).get(0).val);
    }

    @Test
    @DisplayName("cross-language vectors")
    void crossLanguageVectors() {
        IdMix m = new IdMix(CrossLanguageFixtures.ALPHABET);
        for (CrossLanguageFixtures.Case c : CrossLanguageFixtures.cases()) {
            List<TypedValue> decoded = m.decode(c.encoded());
            assertEquals(c.values().size(), decoded.size(), c.name());
            for (int i = 0; i < c.values().size(); i++) {
                assertEquals(c.values().get(i), decoded.get(i), c.name() + "[" + i + "]");
            }
        }
    }
}
