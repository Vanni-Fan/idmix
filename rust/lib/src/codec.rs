use std::sync::OnceLock;

use base64::{engine::general_purpose::STANDARD, Engine as _};

use crate::alphabet::RadixCodec;
use crate::error::IdMixError;
use crate::idmix::DEFAULT_ALPHABET;

/// Binary-to-text codec for the idmix text layer.
pub trait Codec: Send + Sync {
    fn encode(&self, data: &[u8]) -> Result<String, IdMixError>;
    fn decode(&self, s: &str) -> Result<Vec<u8>, IdMixError>;
}

/// Codec implemented by closure pair.
pub struct FuncCodec {
    pub encode_fn: Box<dyn Fn(&[u8]) -> Result<String, IdMixError> + Send + Sync>,
    pub decode_fn: Box<dyn Fn(&str) -> Result<Vec<u8>, IdMixError> + Send + Sync>,
}

impl FuncCodec {
    pub fn new(
        encode_fn: impl Fn(&[u8]) -> Result<String, IdMixError> + Send + Sync + 'static,
        decode_fn: impl Fn(&str) -> Result<Vec<u8>, IdMixError> + Send + Sync + 'static,
    ) -> Self {
        Self {
            encode_fn: Box::new(encode_fn),
            decode_fn: Box::new(decode_fn),
        }
    }
}

impl Codec for FuncCodec {
    fn encode(&self, data: &[u8]) -> Result<String, IdMixError> {
        (self.encode_fn)(data)
    }

    fn decode(&self, s: &str) -> Result<Vec<u8>, IdMixError> {
        (self.decode_fn)(s)
    }
}

/// Standard Base64 binary-to-text codec.
#[derive(Debug, Clone, Copy, Default)]
pub struct Base64Codec;

impl Base64Codec {
    pub fn new() -> Self {
        Self
    }
}

impl Codec for Base64Codec {
    fn encode(&self, data: &[u8]) -> Result<String, IdMixError> {
        Ok(STANDARD.encode(data))
    }

    fn decode(&self, s: &str) -> Result<Vec<u8>, IdMixError> {
        STANDARD
            .decode(s)
            .map_err(|e| IdMixError::msg(e.to_string()))
    }
}

impl Codec for RadixCodec {
    fn encode(&self, data: &[u8]) -> Result<String, IdMixError> {
        self.encode_bytes(data)
    }

    fn decode(&self, s: &str) -> Result<Vec<u8>, IdMixError> {
        self.decode_bytes(s)
    }
}

fn default_codec() -> &'static RadixCodec {
    static DEFAULT: OnceLock<RadixCodec> = OnceLock::new();
    DEFAULT.get_or_init(|| {
        RadixCodec::new(DEFAULT_ALPHABET).expect("default alphabet must be valid")
    })
}

fn resolve_codec(codec: Option<&dyn Codec>) -> &dyn Codec {
    match codec {
        Some(c) => c,
        None => default_codec(),
    }
}

/// Encode arbitrary binary to text using the default or provided codec.
pub fn encode_bytes(data: &[u8], codec: Option<&dyn Codec>) -> Result<String, IdMixError> {
    resolve_codec(codec).encode(data)
}

/// Decode text back to binary using the default or provided codec.
pub fn decode_string(s: &str, codec: Option<&dyn Codec>) -> Result<Vec<u8>, IdMixError> {
    resolve_codec(codec).decode(s)
}
