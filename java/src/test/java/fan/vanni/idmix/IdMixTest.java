package fan.vanni.idmix;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/**
 * idmix 测试套件。运行: mvn test
 * 输出详细日志（类似 go test -v）。
 */
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

    private List<TypedValue> logRoundTrip(IdMix m, String title, TypedValue... values) {
        System.out.println("\n" + "─".repeat(40));
        System.out.println("▶ " + title);
        System.out.printf("  字符表: \"%s\" (进制=%d)%n", m.getRadix().getChars(), m.getRadix().getBase());
        for (int i = 0; i < values.length; i++) {
            System.out.printf("  编码输入[%d] otype=%d val=%d%n", i, values[i].otype, values[i].val);
        }
        String encoded = m.encode(values);
        byte[] raw = m.getRadix().decodeBytes(encoded);
        System.out.printf("  二进制: %s (%d bytes)%n", hex(raw), raw.length);
        System.out.printf("  字符串: \"%s\" (len=%d)%n", encoded, encoded.length());
        List<TypedValue> decoded = m.decode(encoded);
        for (int i = 0; i < decoded.size(); i++) {
            System.out.printf("  解码输出[%d] otype=%d val=%d%n", i, decoded.get(i).otype, decoded.get(i).val);
        }
        for (int i = 0; i < values.length; i++) {
            String mark = decoded.get(i).equals(values[i]) ? "✓" : "✗";
            System.out.printf("  校验[%d]: %s  want(%d,%d) => got(%d,%d)%n",
                    i, mark, values[i].otype, values[i].val,
                    decoded.get(i).otype, decoded.get(i).val);
            assertEquals(values[i], decoded.get(i));
        }
        return decoded;
    }

    @Test
    @DisplayName("规范二进制块与 arithmetic.md 第7节一致 (variant=0)")
    void specExampleBinary() {
        IdMix m = IdMix.newDefault();
        List<TypedValue> typed = List.of(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
        byte[] data = XidCodec.encodeBinary(m, typed, 0);
        byte[] want = new byte[]{0x0F, 0x00, 0x22, 0x47, (byte) 0xB5, 0x1F};
        System.out.println("\n▶ 规范二进制块 (variant=0): " + hex(data));
        assertArrayEquals(want, data);
        System.out.println("  与 arithmetic.md 第7节示例一致 ✓");
    }

    @Test
    @DisplayName("规范示例往返: u16(5), i64(-1), u32(40)")
    void roundTripBasic() {
        logRoundTrip(IdMix.newDefault(), "规范示例: u16(5), i64(-1), u32(40)",
                TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
    }

    @Test
    @DisplayName("单值 u32(2000000000) 往返")
    void roundTripUint32Large() {
        List<TypedValue> out = logRoundTrip(IdMix.newDefault(), "单值 u32(2000000000)",
                TypedValue.u32(2_000_000_000L));
        assertEquals(2_000_000_000L, out.get(0).val);
    }

    @Test
    @DisplayName("自定义四进制字符表往返")
    void customAlphabet() {
        logRoundTrip(new IdMix("abcd"), "四进制 abcd",
                TypedValue.u16(100), TypedValue.i32(-10), TypedValue.u8(3));
    }

    @Test
    @DisplayName("校验和不匹配时拒绝解码")
    void checksumRejects() {
        IdMix m = IdMix.newDefault();
        byte[] data = XidCodec.encodeBinary(m, List.of(TypedValue.u32(1)), 0);
        System.out.println("\n▶ 校验和拒绝测试");
        System.out.println("  原始: " + hex(data));
        data[2] ^= 0x01;
        System.out.println("  篡改: " + hex(data));
        String tampered = m.getRadix().encodeBytes(data);
        assertThrows(IllegalArgumentException.class, () -> m.decode(tampered));
        System.out.println("  解码拒绝 ✓");
    }

    @Test
    @DisplayName("变体多态: 同一输入产生多种字符串")
    void multipleEncodingsDiffer() {
        IdMix m = IdMix.newDefault();
        Set<String> seen = new HashSet<>();
        for (int i = 0; i < 50; i++) seen.add(m.encode(TypedValue.u32(42)));
        System.out.printf("%n▶ 变体多态: u32(42) 编码 50 次 => %d 种不同字符串%n", seen.size());
        assertTrue(seen.size() >= 2);
    }
}
