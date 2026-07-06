package io.github.vannifan.idmix;

import java.util.List;

/** 与 testdata/cross_language_vectors.json 同步的跨语言测试向量（避免测试依赖 JSON 库）。 */
final class CrossLanguageFixtures {
    private CrossLanguageFixtures() {}

    static final String ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    record Case(String name, String encoded, List<TypedValue> values) {}

    static List<Case> cases() {
        return List.of(
                new Case("spec_example", "hYpGvRq6B",
                        List.of(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40))),
                new Case("uint32_max", "LwMDzFPIwK",
                        List.of(TypedValue.u32(4294967295L))),
                new Case("int32_min", "LwMH4is20x",
                        List.of(TypedValue.i32(-2147483648))),
                new Case("int64_min", "eA3BqyCfeJ73bad1",
                        List.of(TypedValue.i64(Long.MIN_VALUE))),
                new Case("int64_max", "eA3A34tsjcVrPPF6",
                        List.of(TypedValue.i64(Long.MAX_VALUE))),
                new Case("uint64_max", "eA3A5uobrwZQuXVc",
                        List.of(TypedValue.u64("18446744073709551615"))),
                new Case("mixed_extremes", "bTcNSaewCwrxPlc5fGCbq11xnBz120cpBTJ1A6ztNY",
                        List.of(
                                TypedValue.u32(4294967295L),
                                TypedValue.i32(-2147483648),
                                TypedValue.i64(Long.MIN_VALUE),
                                TypedValue.i64(Long.MAX_VALUE))),
                new Case("embedded_small", "hYI25mckd",
                        List.of(TypedValue.u8(15), TypedValue.i8(-16), TypedValue.u16(0), TypedValue.i16(-1))),
                new Case("access_key", "eB12pBLCoFhaAgPE",
                        List.of(TypedValue.u32(1001), TypedValue.u64(1690000000L), TypedValue.u8(3)))
        );
    }
}
