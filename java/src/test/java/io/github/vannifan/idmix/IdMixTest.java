package io.github.vannifan.idmix;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

class IdMixTest {

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
        Idx idx = Idx.create();
        byte[] data = idx.encodeWithVariant(0, TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
        byte[] want = new byte[]{(byte) 0x80, 0x03, 0x22, 0x47, (byte) 0xB5, 0x1F};
        assertArrayEquals(want, data);
    }

    @Test
    @DisplayName("round trip u16(5), i64(-1), u32(40)")
    void roundTripBasic() {
        IdMix m = IdMix.create();
        String encoded = m.encode(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
        Object[] decoded = m.decode(encoded);
        assertEquals(3, decoded.length);
        CrossLanguageFixtures.assertValueEquals(
                CrossLanguageFixtures.CrossLangValue.integer(1, 5), decoded[0], "u16");
        CrossLanguageFixtures.assertValueEquals(
                CrossLanguageFixtures.CrossLangValue.integer(7, -1), decoded[1], "i64");
        CrossLanguageFixtures.assertValueEquals(
                CrossLanguageFixtures.CrossLangValue.integer(2, 40), decoded[2], "u32");
    }

    @Test
    @DisplayName("round trip u32(2000000000)")
    void roundTripUint32Large() {
        IdMix m = IdMix.create();
        Object[] out = m.decode(m.encode(TypedValue.u32(2_000_000_000L)));
        assertEquals(2_000_000_000L, ((TypedValue) out[0]).val);
    }

    @Test
    @DisplayName("custom alphabet abcd")
    void customAlphabet() {
        IdMix m = IdMix.create(new IdMix.IdMixBuilder().withAlphabet("abcd"));
        Object[] decoded = m.decode(m.encode(TypedValue.u16(100), TypedValue.i32(-10), TypedValue.u8(3)));
        assertEquals(3, decoded.length);
    }

    @Test
    @DisplayName("checksum mismatch rejects decode")
    void checksumRejects() {
        IdMix m = IdMix.create();
        byte[] data = m.getIdx().encodeWithVariant(0, TypedValue.u32(1));
        data[0] ^= 0x01;
        String tampered = m.getCodec().encode(data);
        assertThrows(IllegalArgumentException.class, () -> m.decode(tampered));
    }

    @Test
    @DisplayName("variant polymorphism")
    void multipleEncodingsDiffer() {
        IdMix m = IdMix.create();
        Set<String> seen = new HashSet<>();
        for (int i = 0; i < 50; i++) seen.add(m.encode(TypedValue.u32(42)));
        assertTrue(seen.size() >= 2);
    }

    @Test
    @DisplayName("extreme values round trip")
    void extremeValuesRoundTrip() {
        IdMix m = IdMix.create();
        assertEquals(4294967295L, ((TypedValue) m.decode(m.encode(TypedValue.u32(4294967295L)))[0]).val);
        assertEquals(-2147483648, ((TypedValue) m.decode(m.encode(TypedValue.i32(-2147483648)))[0]).val);
        assertEquals(Long.MIN_VALUE, ((TypedValue) m.decode(m.encode(TypedValue.i64(Long.MIN_VALUE)))[0]).val);
        assertEquals(Long.MAX_VALUE, ((TypedValue) m.decode(m.encode(TypedValue.i64(Long.MAX_VALUE)))[0]).val);
        assertEquals(-1L, ((TypedValue) m.decode(m.encode(TypedValue.u64("18446744073709551615")))[0]).val);
    }

    @Test
    @DisplayName("cross-language vectors")
    void crossLanguageVectors() {
        IdMix m = IdMix.create(new IdMix.IdMixBuilder().withAlphabet(CrossLanguageFixtures.ALPHABET));
        for (CrossLanguageFixtures.Case c : CrossLanguageFixtures.cases()) {
            Object[] decoded = m.decode(c.encoded());
            assertEquals(c.values().size(), decoded.length, c.name());
            for (int i = 0; i < c.values().size(); i++) {
                CrossLanguageFixtures.assertValueEquals(c.values().get(i), decoded[i], c.name() + "[" + i + "]");
            }
        }
    }

    @Test
    @DisplayName("cross-language encode deterministic")
    void crossLanguageEncodeDeterministic() {
        IdMix m = IdMix.create(new IdMix.IdMixBuilder().withAlphabet(CrossLanguageFixtures.ALPHABET));
        for (CrossLanguageFixtures.Case c : CrossLanguageFixtures.cases()) {
            Object[] inputs = c.values().stream().map(CrossLanguageFixtures::materialize).toArray();
            String enc = m.encodeWithVariant(c.variant(), inputs);
            assertEquals(c.encoded(), enc, c.name());
        }
    }

    @Test
    @DisplayName("string round trip")
    void stringRoundTrip() {
        IdMix m = IdMix.create();
        Object[] decoded = m.decode(m.encodeWithVariant(0, "hello", TypedValue.u16(5), "世界"));
        assertEquals("hello", decoded[0]);
        CrossLanguageFixtures.assertValueEquals(
                CrossLanguageFixtures.CrossLangValue.integer(1, 5), decoded[1], "u16");
        assertEquals("世界", decoded[2]);
    }
}
