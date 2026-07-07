use std::sync::Arc;

use rand::Rng;

use crate::alphabet::RadixCodec;
use crate::codec::{self, Codec};
use crate::error::IdMixError;
use crate::idx_codec::Idx;
use crate::number::normalize_objects;
use crate::typed_value::Value;

pub const DEFAULT_ALPHABET: &str =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

/// IdMix combines IDX binary encoding with a pluggable text codec.
pub struct IdMix {
    idx: Idx,
    codec: Arc<dyn Codec>,
}

impl IdMix {
    pub fn new() -> Result<Self, IdMixError> {
        Self::builder().build()
    }

    pub fn builder() -> IdMixBuilder {
        IdMixBuilder::default()
    }

    pub fn idx(&self) -> &Idx {
        &self.idx
    }

    pub fn codec(&self) -> &dyn Codec {
        self.codec.as_ref()
    }

    pub fn encode(&self, values: &[Value]) -> Result<String, IdMixError> {
        if values.is_empty() {
            return Err(IdMixError::msg("at least one value is required"));
        }
        let variant_id = rand::thread_rng().gen_range(0..self.idx.max_variants as i32);
        self.encode_with_variant(variant_id, values)
    }

    pub fn encode_with_variant(&self, variant_id: i32, values: &[Value]) -> Result<String, IdMixError> {
        let data = self.encode_binary(variant_id, values)?;
        self.codec.encode(&data)
    }

    pub fn decode(&self, s: &str) -> Result<Vec<Value>, IdMixError> {
        let data = self.codec.decode(s)?;
        self.idx.decode(&data)
    }

    pub(crate) fn encode_binary(&self, variant_id: i32, values: &[Value]) -> Result<Vec<u8>, IdMixError> {
        if values.is_empty() {
            return Err(IdMixError::msg("at least one value is required"));
        }
        let objects = normalize_objects(values)?;
        self.idx.encode_binary(&objects, variant_id)
    }
}

#[derive(Default)]
pub struct IdMixBuilder {
    idx: Option<Idx>,
    codec: Option<Arc<dyn Codec>>,
    alphabet: Option<String>,
}

impl IdMixBuilder {
    pub fn idx(mut self, idx: Idx) -> Self {
        self.idx = Some(idx);
        self
    }

    pub fn codec(mut self, codec: Arc<dyn Codec>) -> Self {
        self.codec = Some(codec);
        self.alphabet = None;
        self
    }

    pub fn alphabet(mut self, alphabet: impl Into<String>) -> Self {
        self.alphabet = Some(alphabet.into());
        self.codec = None;
        self
    }

    pub fn build(self) -> Result<IdMix, IdMixError> {
        let idx = self.idx.unwrap_or_default();
        let codec: Arc<dyn Codec> = match (self.codec, self.alphabet) {
            (Some(c), _) => c,
            (None, Some(alphabet)) => Arc::new(RadixCodec::new(&alphabet)?),
            (None, None) => Arc::new(RadixCodec::new(DEFAULT_ALPHABET)?),
        };
        Ok(IdMix { idx, codec })
    }
}

pub use codec::{encode_bytes, decode_string, Base64Codec, FuncCodec};
