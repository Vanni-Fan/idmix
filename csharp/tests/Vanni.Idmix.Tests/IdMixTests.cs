using Vanni.Idmix;

namespace Vanni.Idmix.Tests;

public class IdMixTests
{
    private static string Hex(byte[] b)
    {
        if (b.Length == 0) return "(empty)";
        return string.Join(" ", b.Select(x => $"{x:X2}"));
    }

    private static List<TypedValue> LogRoundTrip(IdMix m, string title, params TypedValue[] values)
    {
        Console.WriteLine($"\n{new string('─', 40)}");
        Console.WriteLine($">> {title}");
        Console.WriteLine($"  字符表: \"{m.Radix.Chars}\" (进制={m.Radix.Base})");
        for (var i = 0; i < values.Length; i++)
            Console.WriteLine($"  编码输入[{i}] otype={values[i].OType} val={values[i].Val}");
        var encoded = m.Encode(values);
        var raw = m.Radix.DecodeBytes(encoded);
        Console.WriteLine($"  二进制: {Hex(raw)} ({raw.Length} bytes)");
        Console.WriteLine($"  字符串: \"{encoded}\" (len={encoded.Length})");
        var decoded = m.Decode(encoded);
        for (var i = 0; i < decoded.Count; i++)
            Console.WriteLine($"  解码输出[{i}] otype={decoded[i].OType} val={decoded[i].Val}");
        for (var i = 0; i < values.Length; i++)
        {
            var mark = decoded[i].Equals(values[i]) ? "OK" : "FAIL";
            Console.WriteLine($"  校验[{i}]: {mark}  want({values[i].OType},{values[i].Val}) => got({decoded[i].OType},{decoded[i].Val})");
            Assert.Equal(values[i], decoded[i]);
        }
        return decoded;
    }

    [Fact]
    public void SpecExampleBinary()
    {
        var m = IdMix.NewDefault();
        var typed = new List<TypedValue> { TypedValue.U16(5), TypedValue.I64(-1), TypedValue.U32(40) };
        var data = XidCodec.EncodeBinary(m, typed, 0);
        var want = new byte[] { 0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F };
        Console.WriteLine($"\n>> 规范二进制块 (variant=0): {Hex(data)}");
        Assert.Equal(want, data);
    }

    [Fact]
    public void RoundTripBasic() =>
        LogRoundTrip(IdMix.NewDefault(), "规范示例: u16(5), i64(-1), u32(40)",
            TypedValue.U16(5), TypedValue.I64(-1), TypedValue.U32(40));

    [Fact]
    public void RoundTripUint32Large()
    {
        var outList = LogRoundTrip(IdMix.NewDefault(), "单值 u32(2000000000)", TypedValue.U32(2_000_000_000L));
        Assert.Equal(2_000_000_000L, outList[0].Val);
    }

    [Fact]
    public void CustomAlphabet() =>
        LogRoundTrip(new IdMix("abcd"), "四进制 abcd",
            TypedValue.U16(100), TypedValue.I32(-10), TypedValue.U8(3));

    [Fact]
    public void ChecksumRejects()
    {
        var m = IdMix.NewDefault();
        var data = XidCodec.EncodeBinary(m, [TypedValue.U32(1)], 0);
        data[2] ^= 0x01;
        var tampered = m.Radix.EncodeBytes(data);
        Assert.Throws<ArgumentException>(() => m.Decode(tampered));
    }

    [Fact]
    public void MultipleEncodingsDiffer()
    {
        var m = IdMix.NewDefault();
        var seen = new HashSet<string>();
        for (var i = 0; i < 50; i++) seen.Add(m.Encode(TypedValue.U32(42)));
        Assert.True(seen.Count >= 2);
    }
}
