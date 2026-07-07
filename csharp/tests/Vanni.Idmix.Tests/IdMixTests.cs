using Vanni.Idmix;
using Xunit;

namespace Vanni.Idmix.Tests;

public class IdMixTests
{
    private static string Hex(byte[] b)
    {
        if (b.Length == 0) return "(empty)";
        return string.Join(" ", b.Select(x => $"{x:X2}"));
    }

    private static object[] LogRoundTrip(IdMix m, string title, params object[] values)
    {
        Console.WriteLine($"\n{new string('─', 40)}");
        Console.WriteLine($">> {title}");
        var radix = (RadixCodec)m.Codec;
        Console.WriteLine($"  字符表: \"{radix.Alphabet}\" (进制={radix.Base})");
        for (var i = 0; i < values.Length; i++)
            Console.WriteLine($"  编码输入[{i}] {values[i]}");
        var encoded = m.Encode(values);
        var raw = m.Codec.Decode(encoded);
        Console.WriteLine($"  二进制: {Hex(raw)} ({raw.Length} bytes)");
        Console.WriteLine($"  字符串: \"{encoded}\" (len={encoded.Length})");
        var decoded = m.Decode(encoded);
        for (var i = 0; i < decoded.Length; i++)
            Console.WriteLine($"  解码输出[{i}] {decoded[i]}");
        Assert.Equal(values.Length, decoded.Length);
        for (var i = 0; i < values.Length; i++)
            AssertValueEqual(values[i], decoded[i], i);
        return decoded;
    }

    private static void AssertValueEqual(object want, object got, int index)
    {
        if (want is string ws)
        {
            Assert.Equal(ws, Assert.IsType<string>(got));
            return;
        }
        var wantObj = Number.ObjectFromAny(want);
        var gotObj = Number.ObjectFromAny(got);
        Assert.False(gotObj.IsString);
        Assert.Equal(wantObj.OType, gotObj.OType);
        Assert.Equal(wantObj.Val, gotObj.Val);
    }

    [Fact]
    public void SpecExampleBinary()
    {
        var idx = Idx.Create();
        var data = idx.EncodeWithVariant(0, (ushort)5, -1L, 40u);
        var want = new byte[] { 0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F };
        Console.WriteLine($"\n>> 规范二进制块 (variant=0): {Hex(data)}");
        Assert.Equal(want, data);
    }

    [Fact]
    public void RoundTripBasic() =>
        LogRoundTrip(IdMix.Create(), "规范示例: u16(5), i64(-1), u32(40)",
            (ushort)5, -1L, 40u);

    [Fact]
    public void RoundTripUint32Large()
    {
        var decoded = LogRoundTrip(IdMix.Create(), "单值 u32(2000000000)", 2_000_000_000u);
        Assert.Equal(2_000_000_000u, decoded[0]);
    }

    [Fact]
    public void CustomAlphabet() =>
        LogRoundTrip(IdMix.Create(b => b.WithAlphabet("abcd")), "四进制 abcd",
            (ushort)100, -10, (byte)3);

    [Fact]
    public void ChecksumRejects()
    {
        var m = IdMix.Create();
        var data = m.Idx.EncodeWithVariant(0, 1u);
        data[0] ^= 0x01;
        var tampered = m.Codec.Encode(data);
        Assert.Throws<ArgumentException>(() => m.Decode(tampered));
    }

    [Fact]
    public void MultipleEncodingsDiffer()
    {
        var m = IdMix.Create();
        var seen = new HashSet<string>();
        for (var i = 0; i < 50; i++) seen.Add(m.Encode(42u));
        Assert.True(seen.Count >= 2);
    }

    [Fact]
    public void ExtremeValuesRoundTrip()
    {
        var m = IdMix.Create();
        Assert.Equal(4294967295u, LogRoundTrip(m, "uint32_max", 4294967295u)[0]);
        Assert.Equal(-2147483648, LogRoundTrip(m, "int32_min", -2147483648)[0]);
        Assert.Equal(long.MinValue, LogRoundTrip(m, "int64_min", long.MinValue)[0]);
        Assert.Equal(long.MaxValue, LogRoundTrip(m, "int64_max", long.MaxValue)[0]);
        Assert.Equal(ulong.MaxValue, LogRoundTrip(m, "uint64_max", ulong.MaxValue)[0]);
    }

    [Fact]
    public void CrossLanguageVectors()
    {
        var m = IdMix.Create(b => b.WithAlphabet(CrossLanguageFixtures.Alphabet));
        foreach (var c in CrossLanguageFixtures.Cases)
        {
            var decoded = m.Decode(c.Encoded);
            Assert.Equal(c.Values.Length, decoded.Length);
            for (var i = 0; i < c.Values.Length; i++)
                CrossLanguageFixtures.AssertValueEquals(c.Values[i], decoded[i], i);
        }
    }

    [Fact]
    public void CrossLanguageEncodeDeterministic()
    {
        var m = IdMix.Create(b => b.WithAlphabet(CrossLanguageFixtures.Alphabet));
        foreach (var c in CrossLanguageFixtures.Cases)
        {
            var inputs = c.Values.Select(CrossLanguageFixtures.Materialize).ToArray();
            var enc = m.EncodeWithVariant(c.Variant, inputs);
            Assert.Equal(c.Encoded, enc);
        }
    }

    [Fact]
    public void StringRoundTrip()
    {
        var m = IdMix.Create();
        var decoded = m.Decode(m.EncodeWithVariant(0, "hello", (ushort)5, "世界"));
        Assert.Equal("hello", decoded[0]);
        Assert.Equal((ushort)5, decoded[1]);
        Assert.Equal("世界", decoded[2]);
    }
}
