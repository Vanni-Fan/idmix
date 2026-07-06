//! idmix vs sqids 编码长度与性能对比（Rust）。

use std::time::Instant;

use idmix::{IdMix, Value, DEFAULT_ALPHABET};
use sqids::Sqids;

const TEST_UINT32_LARGE_SINGLE: u32 = 2_000_000_000;
const EXTREME_UINT32_MAX: u32 = 4_294_967_295;
const EXTREME_INT64_MAX: i64 = 9_223_372_036_854_775_807;
const EXTREME_UINT64_MAX: u64 = 18_446_744_073_709_551_615;

struct CompareCase {
    name: &'static str,
    numbers: Vec<u64>,
    idmix_values: Vec<Value>,
}

fn default_cases() -> Vec<CompareCase> {
    vec![
        CompareCase {
            name: "经典示例 [1,2,3]",
            numbers: vec![1, 2, 3],
            idmix_values: vec![Value::U32(1), Value::U32(2), Value::U32(3)],
        },
        CompareCase {
            name: "单个小ID [42]",
            numbers: vec![42],
            idmix_values: vec![Value::U32(42)],
        },
        CompareCase {
            name: "单个大ID [123456789012345]",
            numbers: vec![123_456_789_012_345],
            idmix_values: vec![Value::U64(123_456_789_012_345)],
        },
        CompareCase {
            name: "单值 uint32 [2000000000]",
            numbers: vec![TEST_UINT32_LARGE_SINGLE as u64],
            idmix_values: vec![Value::U32(TEST_UINT32_LARGE_SINGLE)],
        },
        CompareCase {
            name: "AccessKey三元组",
            numbers: vec![1001, 1_690_000_000, 3],
            idmix_values: vec![Value::U32(1001), Value::U64(1_690_000_000), Value::U8(3)],
        },
        CompareCase {
            name: "小整数密集 [0..9]",
            numbers: (0..10).map(|n| n as u64).collect(),
            idmix_values: (0..10).map(|n| Value::U8(n)).collect(),
        },
        CompareCase {
            name: "极值 uint32_max",
            numbers: vec![EXTREME_UINT32_MAX as u64],
            idmix_values: vec![Value::U32(EXTREME_UINT32_MAX)],
        },
        CompareCase {
            name: "极值 int64_max",
            numbers: vec![EXTREME_INT64_MAX as u64],
            idmix_values: vec![Value::I64(EXTREME_INT64_MAX)],
        },
        CompareCase {
            name: "极值 uint64_max",
            numbers: vec![EXTREME_UINT64_MAX],
            idmix_values: vec![Value::U64(EXTREME_UINT64_MAX)],
        },
        CompareCase {
            name: "极值三元组 [u32max,i64max,u64max]",
            numbers: vec![
                EXTREME_UINT32_MAX as u64,
                EXTREME_INT64_MAX as u64,
                EXTREME_UINT64_MAX,
            ],
            idmix_values: vec![
                Value::U32(EXTREME_UINT32_MAX),
                Value::I64(EXTREME_INT64_MAX),
                Value::U64(EXTREME_UINT64_MAX),
            ],
        },
    ]
}

struct LengthStat {
    min: usize,
    max: usize,
    avg: f64,
    sample: String,
}

fn measure_idmix(m: &IdMix, values: &[Value]) -> LengthStat {
    let rounds = 32;
    let mut min = usize::MAX;
    let mut max = 0usize;
    let mut total = 0usize;
    let mut sample = String::new();
    for i in 0..rounds {
        let s = m.encode(values).expect("encode");
        let l = s.len();
        total += l;
        min = min.min(l);
        max = max.max(l);
        if i == 0 {
            sample = s;
        }
    }
    LengthStat {
        min,
        max,
        avg: total as f64 / rounds as f64,
        sample,
    }
}

fn bench_once(rounds: u32, mut f: impl FnMut()) -> (f64, f64) {
    let start = Instant::now();
    for _ in 0..rounds {
        f();
    }
    let elapsed = start.elapsed();
    let ops = rounds as f64 / elapsed.as_secs_f64();
    let ns = elapsed.as_nanos() as f64 / rounds as f64;
    (ops, ns)
}

#[test]
fn compare_sqids_length() {
    let idmix = IdMix::new().unwrap();
    let sqids = Sqids::default();

    println!("idmix vs sqids (Rust) — 编码长度");
    for c in default_cases() {
        let sid = sqids.encode(&c.numbers).unwrap();
        let istat = measure_idmix(&idmix, &c.idmix_values);
        println!(
            "{:<28} sqids={:2} ({})  idmix={:2}~{:2} ({})",
            c.name,
            sid.len(),
            &sid[..sid.len().min(8)],
            istat.min,
            istat.max,
            &istat.sample[..istat.sample.len().min(8)],
        );
    }
}

#[test]
fn compare_sqids_performance() {
    const ROUNDS: u32 = 20_000;
    let idmix = IdMix::new().unwrap();
    let sqids = Sqids::default();

    let perf_cases = [
        (
            "Encode [1,2,3]",
            vec![1u64, 2, 3],
            vec![Value::U32(1), Value::U32(2), Value::U32(3)],
        ),
        (
            "Encode [1001,1690000000,3]",
            vec![1001, 1_690_000_000, 3],
            vec![Value::U32(1001), Value::U64(1_690_000_000), Value::U8(3)],
        ),
        (
            "Encode uint32 [2000000000]",
            vec![TEST_UINT32_LARGE_SINGLE as u64],
            vec![Value::U32(TEST_UINT32_LARGE_SINGLE)],
        ),
        (
            "Encode 极值 uint32_max",
            vec![EXTREME_UINT32_MAX as u64],
            vec![Value::U32(EXTREME_UINT32_MAX)],
        ),
        (
            "Encode 极值 int64_max",
            vec![EXTREME_INT64_MAX as u64],
            vec![Value::I64(EXTREME_INT64_MAX)],
        ),
        (
            "Encode 极值 uint64_max",
            vec![EXTREME_UINT64_MAX],
            vec![Value::U64(EXTREME_UINT64_MAX)],
        ),
        (
            "Encode 极值三元组 [u32max,i64max,u64max]",
            vec![
                EXTREME_UINT32_MAX as u64,
                EXTREME_INT64_MAX as u64,
                EXTREME_UINT64_MAX,
            ],
            vec![
                Value::U32(EXTREME_UINT32_MAX),
                Value::I64(EXTREME_INT64_MAX),
                Value::U64(EXTREME_UINT64_MAX),
            ],
        ),
    ];

    println!("idmix vs sqids (Rust) — 性能 ({} 次)", ROUNDS);
    for (name, numbers, values) in perf_cases {
        let sid = sqids.encode(&numbers).unwrap();
        let iid = idmix.encode(&values).unwrap();

        let (sq_enc_ops, sq_enc_ns) = bench_once(ROUNDS, || {
            let _ = sqids.encode(&numbers).unwrap();
        });
        let (id_enc_ops, id_enc_ns) = bench_once(ROUNDS, || {
            let _ = idmix.encode(&values).unwrap();
        });
        let (sq_dec_ops, sq_dec_ns) = bench_once(ROUNDS, || {
            let _ = sqids.decode(&sid);
        });
        let (id_dec_ops, id_dec_ns) = bench_once(ROUNDS, || {
            let _ = idmix.decode(&iid).unwrap();
        });

        println!("▶ {name}");
        println!(
            "  编码 sqids: {sq_enc_ops:8.0} ops/s ({sq_enc_ns:6.0} ns)  idmix: {id_enc_ops:8.0} ops/s ({id_enc_ns:6.0} ns) [{:.2}x]",
            id_enc_ops / sq_enc_ops
        );
        println!(
            "  解码 sqids: {sq_dec_ops:8.0} ops/s ({sq_dec_ns:6.0} ns)  idmix: {id_dec_ops:8.0} ops/s ({id_dec_ns:6.0} ns) [{:.2}x]",
            id_dec_ops / sq_dec_ops
        );
        println!("  字符串长度 sqids={} idmix={}", sid.len(), iid.len());
    }

    let _ = DEFAULT_ALPHABET;
}
