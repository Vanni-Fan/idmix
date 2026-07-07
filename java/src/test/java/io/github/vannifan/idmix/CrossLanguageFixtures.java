package io.github.vannifan.idmix;

import java.util.List;

/** Cross-language test vectors synced with testdata/cross_language_vectors.json. */
final class CrossLanguageFixtures {
    private CrossLanguageFixtures() {}

    static final String ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    record CrossLangValue(int otype, long val, String str) {
        static CrossLangValue integer(int otype, long val) { return new CrossLangValue(otype, val, null); }
        static CrossLangValue str(String str) { return new CrossLangValue(0, 0, str); }
    }

    record Case(String name, int variant, String encoded, List<CrossLangValue> values) {}

    static List<Case> cases() {
        return List.of(
                new Case("spec_example", 0, "ixHjl0FK7",
                        List.of(
                                CrossLangValue.integer(1, 5),
                                CrossLangValue.integer(7, -1),
                                CrossLangValue.integer(2, 40))),
                new Case("uint32_max", 0, "hUdZLNKGa",
                        List.of(CrossLangValue.integer(2, 4294967295L))),
                new Case("int32_min", 0, "hUdElRoHP",
                        List.of(CrossLangValue.integer(6, -2147483648))),
                new Case("int64_min", 0, "8B10qg6x0EAf3b",
                        List.of(CrossLangValue.integer(7, Long.MIN_VALUE))),
                new Case("int64_max", 0, "8B2cU8kbWpQ2RM",
                        List.of(CrossLangValue.integer(7, Long.MAX_VALUE))),
                new Case("uint64_max", 0, "8B3CPRsv0Owa6S",
                        List.of(CrossLangValue.integer(3, Long.parseUnsignedLong("18446744073709551615")))),
                new Case("mixed_extremes", 0, "bULoRnNZJinEZGKD78wIigIaw6QplS8B0HGNCKO2L6",
                        List.of(
                                CrossLangValue.integer(2, 4294967295L),
                                CrossLangValue.integer(6, -2147483648),
                                CrossLangValue.integer(7, Long.MIN_VALUE),
                                CrossLangValue.integer(7, Long.MAX_VALUE))),
                new Case("embedded_small", 0, "ixHorRWmh",
                        List.of(
                                CrossLangValue.integer(0, 15),
                                CrossLangValue.integer(4, -16),
                                CrossLangValue.integer(1, 0),
                                CrossLangValue.integer(5, -1))),
                new Case("access_key", 0, "eNe8RmcNtYw60Xjc",
                        List.of(
                                CrossLangValue.integer(2, 1001),
                                CrossLangValue.integer(3, 1690000000L),
                                CrossLangValue.integer(0, 3))),
                new Case("string_example", 0, "ceOqw5RPaTfgnfXyp7Sdepb",
                        List.of(
                                CrossLangValue.str("hello"),
                                CrossLangValue.integer(1, 5),
                                CrossLangValue.str("世界")))
        );
    }

    static Object materialize(CrossLangValue v) {
        if (v.str() != null) return v.str();
        return switch (v.otype()) {
            case TypedValue.OTYPE_UINT8 -> TypedValue.u8((int) v.val());
            case TypedValue.OTYPE_UINT16 -> TypedValue.u16((int) v.val());
            case TypedValue.OTYPE_UINT32 -> TypedValue.u32(v.val());
            case TypedValue.OTYPE_UINT64 -> TypedValue.u64(Long.toUnsignedString(v.val()));
            case TypedValue.OTYPE_INT8 -> TypedValue.i8((int) v.val());
            case TypedValue.OTYPE_INT16 -> TypedValue.i16((int) v.val());
            case TypedValue.OTYPE_INT32 -> TypedValue.i32((int) v.val());
            case TypedValue.OTYPE_INT64 -> TypedValue.i64(v.val());
            default -> throw new IllegalArgumentException("invalid otype " + v.otype());
        };
    }

    static void assertValueEquals(CrossLangValue want, Object got, String label) {
        if (want.str() != null) {
            if (!(got instanceof String s) || !want.str().equals(s)) {
                throw new AssertionError(label + ": got str=" + got + ", want " + want.str());
            }
            return;
        }
        if (got instanceof TypedValue tv) {
            if (tv.otype != want.otype() || tv.val != want.val()) {
                throw new AssertionError(label + ": got otype=" + tv.otype + " val=" + tv.val
                        + ", want otype=" + want.otype() + " val=" + want.val());
            }
            return;
        }
        Number.DataObject gotObj = Number.objectFromAny(got);
        if (gotObj.isString || gotObj.otype != want.otype() || gotObj.val != want.val()) {
            throw new AssertionError(label + ": got otype=" + gotObj.otype + " val=" + gotObj.val
                    + ", want otype=" + want.otype() + " val=" + want.val());
        }
    }
}
