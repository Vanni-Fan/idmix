pub mod alphabet;
pub mod error;
pub mod idmix;
pub mod typed_value;
pub mod xid_codec;

pub use error::IdMixError;
pub use idmix::{IdMix, IdMixBuilder, DEFAULT_ALPHABET};
pub use typed_value::{TypedValue, Value};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn spec_example_binary() {
        let m = IdMix::new().unwrap();
        let typed = vec![
            TypedValue {
                otype: typed_value::OTYPE_UINT16,
                val: 5,
            },
            TypedValue {
                otype: typed_value::OTYPE_INT64,
                val: -1,
            },
            TypedValue {
                otype: typed_value::OTYPE_UINT32,
                val: 40,
            },
        ];
        let data = xid_codec::encode_binary(&m, &typed, 0).unwrap();
        let want = [0x0f, 0x00, 0x22, 0x47, 0xb5, 0x1f];
        assert_eq!(data, want);
    }

    #[test]
    fn round_trip_basic() {
        let m = IdMix::new().unwrap();
        let values = [Value::U16(5), Value::I64(-1), Value::U32(40)];
        let s = m.encode(&values).unwrap();
        let out = m.decode(&s).unwrap();
        assert_eq!(out, values);
    }

    #[test]
    fn round_trip_uint32_large_single() {
        let m = IdMix::new().unwrap();
        let values = [Value::U32(2_000_000_000)];
        let s = m.encode(&values).unwrap();
        let out = m.decode(&s).unwrap();
        assert_eq!(out, values);
    }
}
