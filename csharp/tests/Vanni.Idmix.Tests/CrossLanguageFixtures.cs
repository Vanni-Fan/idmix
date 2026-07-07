using Vanni.Idmix;
using Xunit;

namespace Vanni.Idmix.Tests;

/// <summary>与 testdata/cross_language_vectors.json 同步的跨语言测试向量。</summary>
internal static class CrossLanguageFixtures
{
    internal const string Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    internal record CrossLangValue(int OType, long Val, string? StrValue)
    {
        internal static CrossLangValue Int(int otype, long val) => new(otype, val, null);
        internal static CrossLangValue Str(string str) => new(0, 0, str);
    }

    internal record Case(string Name, int Variant, string Encoded, CrossLangValue[] Values);

    internal static Case[] Cases { get; } =
    [
        new("spec_example", 0, "ixHjl0FK7",
            [CrossLangValue.Int(1, 5), CrossLangValue.Int(7, -1), CrossLangValue.Int(2, 40)]),
        new("uint32_max", 0, "hUdZLNKGa", [CrossLangValue.Int(2, 4294967295L)]),
        new("int32_min", 0, "hUdElRoHP", [CrossLangValue.Int(6, -2147483648)]),
        new("int64_min", 0, "8B10qg6x0EAf3b", [CrossLangValue.Int(7, long.MinValue)]),
        new("int64_max", 0, "8B2cU8kbWpQ2RM", [CrossLangValue.Int(7, long.MaxValue)]),
        new("uint64_max", 0, "8B3CPRsv0Owa6S", [CrossLangValue.Int(3, unchecked((long)ulong.MaxValue))]),
        new("mixed_extremes", 0, "bULoRnNZJinEZGKD78wIigIaw6QplS8B0HGNCKO2L6",
            [
                CrossLangValue.Int(2, 4294967295L),
                CrossLangValue.Int(6, -2147483648),
                CrossLangValue.Int(7, long.MinValue),
                CrossLangValue.Int(7, long.MaxValue),
            ]),
        new("embedded_small", 0, "ixHorRWmh",
            [
                CrossLangValue.Int(0, 15),
                CrossLangValue.Int(4, -16),
                CrossLangValue.Int(1, 0),
                CrossLangValue.Int(5, -1),
            ]),
        new("access_key", 0, "eNe8RmcNtYw60Xjc",
            [CrossLangValue.Int(2, 1001), CrossLangValue.Int(3, 1690000000L), CrossLangValue.Int(0, 3)]),
        new("string_example", 0, "ceOqw5RPaTfgnfXyp7Sdepb",
            [CrossLangValue.Str("hello"), CrossLangValue.Int(1, 5), CrossLangValue.Str("世界")]),
    ];

    internal static object Materialize(CrossLangValue v)
    {
        if (v.StrValue != null) return v.StrValue;
        return v.OType switch
        {
            TypedValue.OTypeUint8 => (byte)v.Val,
            TypedValue.OTypeUint16 => (ushort)v.Val,
            TypedValue.OTypeUint32 => (uint)v.Val,
            TypedValue.OTypeUint64 => TypedValue.U64(unchecked((ulong)v.Val)),
            TypedValue.OTypeInt8 => (sbyte)v.Val,
            TypedValue.OTypeInt16 => (short)v.Val,
            TypedValue.OTypeInt32 => (int)v.Val,
            TypedValue.OTypeInt64 => v.Val,
            _ => throw new ArgumentException($"invalid otype {v.OType}"),
        };
    }

    internal static void AssertValueEquals(CrossLangValue want, object got, int index)
    {
        if (want.StrValue != null)
        {
            Assert.Equal(want.StrValue, Assert.IsType<string>(got));
            return;
        }
        var obj = Number.ObjectFromAny(got);
        Assert.False(obj.IsString);
        Assert.Equal(want.OType, obj.OType);
        Assert.Equal(want.Val, obj.Val);
    }
}
