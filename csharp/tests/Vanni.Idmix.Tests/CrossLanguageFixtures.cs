namespace Vanni.Idmix.Tests;

/// <summary>与 testdata/cross_language_vectors.json 同步的跨语言测试向量。</summary>
internal static class CrossLanguageFixtures
{
    internal const string Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    internal record Case(string Name, string Encoded, TypedValue[] Values);

    internal static Case[] Cases { get; } =
    [
        new("spec_example", "hYpGvRq6B",
            [TypedValue.U16(5), TypedValue.I64(-1), TypedValue.U32(40)]),
        new("uint32_max", "LwMDzFPIwK", [TypedValue.U32(4294967295L)]),
        new("int32_min", "LwMH4is20x", [TypedValue.I32(-2147483648)]),
        new("int64_min", "eA3BqyCfeJ73bad1", [TypedValue.I64(long.MinValue)]),
        new("int64_max", "eA3A34tsjcVrPPF6", [TypedValue.I64(long.MaxValue)]),
        new("uint64_max", "eA3A5uobrwZQuXVc", [TypedValue.U64(ulong.MaxValue)]),
        new("mixed_extremes", "bTcNSaewCwrxPlc5fGCbq11xnBz120cpBTJ1A6ztNY",
            [TypedValue.U32(4294967295L), TypedValue.I32(-2147483648), TypedValue.I64(long.MinValue), TypedValue.I64(long.MaxValue)]),
        new("embedded_small", "hYI25mckd",
            [TypedValue.U8(15), TypedValue.I8(-16), TypedValue.U16(0), TypedValue.I16(-1)]),
        new("access_key", "eB12pBLCoFhaAgPE",
            [TypedValue.U32(1001), TypedValue.U64(1690000000L), TypedValue.U8(3)]),
    ];
}
