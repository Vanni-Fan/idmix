pub mod alphabet;
pub mod codec;
pub mod error;
pub mod idmix;
pub mod idx_codec;
pub mod number;
pub mod typed_value;

pub use codec::{encode_bytes, decode_string, Base64Codec, Codec, FuncCodec};
pub use error::IdMixError;
pub use idmix::{IdMix, IdMixBuilder, DEFAULT_ALPHABET};
pub use idx_codec::{Idx, IdxBuilder, MAX_STRING_LEN};
pub use typed_value::{TypedValue, Value};

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use super::*;

    #[test]
    fn spec_example_binary() {
        let m = IdMix::new().unwrap();
        let values = [Value::U16(5), Value::I64(-1), Value::U32(40)];
        let data = m.encode_binary(0, &values).unwrap();
        let want = [0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F];
        assert_eq!(data.as_slice(), want);
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

    #[test]
    fn round_trip_string() {
        let m = IdMix::new().unwrap();
        let values = [
            Value::String("hello".into()),
            Value::U16(5),
            Value::String("世界".into()),
        ];
        let s = m.encode_with_variant(0, &values).unwrap();
        let out = m.decode(&s).unwrap();
        assert_eq!(out, values);
    }

    #[test]
    fn single_object_one_byte_header() {
        let idx = Idx::new().unwrap();
        let data = idx.encode_with_variant(0, &[Value::U32(42)]).unwrap();
        assert_eq!(data.len(), 3);
        assert_eq!(data[0] & 0x80, 0);
    }

    #[test]
    fn custom_codec_base64() {
        let m = IdMix::builder()
            .codec(Arc::new(Base64Codec::new()))
            .build()
            .unwrap();
        let values = [Value::U16(5), Value::I64(-1), Value::U32(40)];
        let s = m.encode_with_variant(0, &values).unwrap();
        let out = m.decode(&s).unwrap();
        assert_eq!(out[0], Value::U16(5));
    }

    #[test]
    fn encode_bytes_standalone() {
        let raw = [0xDE, 0xAD, 0xBE, 0xEF];
        let s = encode_bytes(&raw, None).unwrap();
        let out = decode_string(&s, None).unwrap();
        assert_eq!(out, raw);
    }
}
