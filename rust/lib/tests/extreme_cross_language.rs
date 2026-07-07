//! 极值与跨语言向量测试。

use idmix::{IdMix, Value, DEFAULT_ALPHABET};

const EXTREME_UINT32_MAX: u32 = 4294967295;
const EXTREME_INT32_MIN: i32 = -2147483648;
const EXTREME_INT64_MIN: i64 = i64::MIN;
const EXTREME_INT64_MAX: i64 = i64::MAX;
const EXTREME_UINT64_MAX: u64 = u64::MAX;

struct CrossCase {
    name: &'static str,
    encoded: &'static str,
    values: &'static [Value],
}

const CROSS_CASES: &[CrossCase] = &[
    CrossCase {
        name: "spec_example",
        encoded: "ixHjl0FK7",
        values: &[Value::U16(5), Value::I64(-1), Value::U32(40)],
    },
    CrossCase {
        name: "uint32_max",
        encoded: "hUdZLNKGa",
        values: &[Value::U32(EXTREME_UINT32_MAX)],
    },
    CrossCase {
        name: "int32_min",
        encoded: "hUdElRoHP",
        values: &[Value::I32(EXTREME_INT32_MIN)],
    },
    CrossCase {
        name: "int64_min",
        encoded: "8B10qg6x0EAf3b",
        values: &[Value::I64(EXTREME_INT64_MIN)],
    },
    CrossCase {
        name: "int64_max",
        encoded: "8B2cU8kbWpQ2RM",
        values: &[Value::I64(EXTREME_INT64_MAX)],
    },
    CrossCase {
        name: "uint64_max",
        encoded: "8B3CPRsv0Owa6S",
        values: &[Value::U64(EXTREME_UINT64_MAX)],
    },
    CrossCase {
        name: "mixed_extremes",
        encoded: "bULoRnNZJinEZGKD78wIigIaw6QplS8B0HGNCKO2L6",
        values: &[
            Value::U32(EXTREME_UINT32_MAX),
            Value::I32(EXTREME_INT32_MIN),
            Value::I64(EXTREME_INT64_MIN),
            Value::I64(EXTREME_INT64_MAX),
        ],
    },
    CrossCase {
        name: "embedded_small",
        encoded: "ixHorRWmh",
        values: &[Value::U8(15), Value::I8(-16), Value::U16(0), Value::I16(-1)],
    },
    CrossCase {
        name: "access_key",
        encoded: "eNe8RmcNtYw60Xjc",
        values: &[Value::U32(1001), Value::U64(1_690_000_000), Value::U8(3)],
    },
];

#[test]
fn extreme_values_round_trip() {
    let m = IdMix::new().unwrap();
    let cases = [
        ("uint32_max", vec![Value::U32(EXTREME_UINT32_MAX)]),
        ("int32_min", vec![Value::I32(EXTREME_INT32_MIN)]),
        ("int64_min", vec![Value::I64(EXTREME_INT64_MIN)]),
        ("int64_max", vec![Value::I64(EXTREME_INT64_MAX)]),
        ("uint64_max", vec![Value::U64(EXTREME_UINT64_MAX)]),
    ];
    for (name, values) in cases {
        let s = m.encode(&values).expect(name);
        let out = m.decode(&s).expect(name);
        assert_eq!(out, values, "{name}");
    }
}

#[test]
fn cross_language_vectors() {
    let m = IdMix::builder().alphabet(DEFAULT_ALPHABET).build().unwrap();
    for c in CROSS_CASES {
        let out = m.decode(c.encoded).expect(c.name);
        assert_eq!(out, c.values, "{}", c.name);
    }
}

#[test]
fn cross_language_string_example() {
    let m = IdMix::builder().alphabet(DEFAULT_ALPHABET).build().unwrap();
    let values = [
        Value::String("hello".into()),
        Value::U16(5),
        Value::String("世界".into()),
    ];
    let encoded = "ceOqw5RPaTfgnfXyp7Sdepb";
    let out = m.decode(encoded).unwrap();
    assert_eq!(out, values);
    let enc = m.encode_with_variant(0, &values).unwrap();
    assert_eq!(enc, encoded);
}

#[test]
fn cross_language_encode_deterministic() {
    let m = IdMix::builder().alphabet(DEFAULT_ALPHABET).build().unwrap();
    for c in CROSS_CASES {
        let enc = m.encode_with_variant(0, c.values).expect(c.name);
        assert_eq!(enc, c.encoded, "{}", c.name);
    }
}
