//! Idx configuration and string boundary tests.

use idmix::{IdMix, Idx, Value, MAX_STRING_LEN};

fn repeat(c: char, n: usize) -> String {
    c.to_string().repeat(n)
}

#[test]
fn string_length_boundaries() {
    let idx = Idx::new().unwrap();
    let ok63 = repeat('a', MAX_STRING_LEN);
    let too_long = repeat('b', MAX_STRING_LEN + 1);

    assert!(idx.encode(&[Value::String(String::new())]).is_err());
    assert!(idx.encode(&[Value::Bytes(vec![])]).is_err());

    let data = idx
        .encode_with_variant(0, &[Value::String("x".into())])
        .unwrap();
    let out = idx.decode(&data).unwrap();
    assert_eq!(out[0], Value::String("x".into()));

    let data = idx
        .encode_with_variant(0, &[Value::String(ok63.clone())])
        .unwrap();
    assert_eq!(data.len(), 1 + 1 + MAX_STRING_LEN);
    let out = idx.decode(&data).unwrap();
    assert_eq!(out[0], Value::String(ok63));

    assert!(idx
        .encode(&[Value::String(too_long.clone())])
        .is_err());
    assert!(idx
        .encode(&[Value::Bytes(too_long.into_bytes())])
        .is_err());

    let m = IdMix::new().unwrap();
    let ok63 = repeat('a', MAX_STRING_LEN);
    let s = m
        .encode_with_variant(0, &[Value::String(ok63.clone())])
        .unwrap();
    let out = m.decode(&s).unwrap();
    assert_eq!(out[0], Value::String(ok63));

    let too_long = repeat('b', MAX_STRING_LEN + 1);
    assert!(m
        .encode(&[Value::String(too_long)])
        .is_err());
}

#[test]
fn idx_max_objects() {
    assert!(Idx::builder().max_objects(0).build().is_err());
    assert!(Idx::builder().max_objects(256).build().is_err());

    let idx = Idx::builder().max_objects(2).build().unwrap();
    assert!(idx
        .encode(&[Value::U8(1), Value::U8(2), Value::U8(3)])
        .is_err());

    let data = idx
        .encode_with_variant(0, &[Value::U8(1), Value::U8(2)])
        .unwrap();
    assert_ne!(data[0] & 0x80, 0);
    assert_eq!(data[1], 2);
    let out = idx.decode(&data).unwrap();
    assert_eq!(out.len(), 2);

    let idx = Idx::builder().max_objects(255).build().unwrap();
    assert_eq!(idx.max_objects, 255);
}

#[test]
fn idx_max_variants() {
    assert!(Idx::builder().max_variants(0).build().is_err());
    assert!(Idx::builder().max_variants(33).build().is_err());

    let idx = Idx::builder().max_variants(4).build().unwrap();
    assert!(idx
        .encode_with_variant(4, &[Value::U8(1)])
        .is_err());

    let data = idx
        .encode_with_variant(3, &[Value::U8(42)])
        .unwrap();
    let out = idx.decode(&data).unwrap();
    assert_eq!(out[0], Value::U8(42));

    let large = Idx::builder().max_variants(32).build().unwrap();
    let data = large
        .encode_with_variant(20, &[Value::U8(7)])
        .unwrap();
    let small = Idx::builder().max_variants(16).build().unwrap();
    assert!(small.decode(&data).is_err());

    let idx = Idx::builder().max_variants(8).build().unwrap();
    let m = IdMix::builder().idx(idx).build().unwrap();
    assert!(m
        .encode_with_variant(8, &[Value::U32(1)])
        .is_err());
}

#[test]
fn idx_check_bits() {
    assert!(Idx::builder().check_bits(0).build().is_err());
    assert!(Idx::builder().check_bits(3).build().is_err());

    let idx = Idx::builder().check_bits(1).build().unwrap();
    assert_eq!(idx.check_bits, 1);
    assert_eq!(idx.check_mask, 0x01);

    let data = idx
        .encode_with_variant(0, &[Value::U16(5), Value::I64(-1)])
        .unwrap();
    let out = idx.decode(&data).unwrap();
    assert_eq!(out[0], Value::U16(5));

    let mut tampered = data.clone();
    let last = tampered.len() - 1;
    tampered[last] ^= 0x01;
    assert!(idx.decode(&tampered).is_err());

    let idx = Idx::new().unwrap();
    assert_eq!(idx.check_bits, 2);
    assert_eq!(idx.check_mask, 0x03);

    let idx = Idx::builder().check_bits(1).build().unwrap();
    let m = IdMix::builder().idx(idx).build().unwrap();
    let s = m
        .encode_with_variant(0, &[Value::U32(99)])
        .unwrap();
    let out = m.decode(&s).unwrap();
    assert_eq!(out[0], Value::U32(99));
}
