use rand::Rng;

use crate::alphabet::RadixCodec;
use crate::error::IdMixError;
use crate::typed_value::{materialize_values, normalize_values, Value};
use crate::xid_codec::{decode_binary, encode_binary};

pub const DEFAULT_ALPHABET: &str =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

const DEFAULT_MAX_OBJECTS: usize = 511;
const DEFAULT_MAX_VARIANTS: usize = 32;
const DEFAULT_CHECK_BITS: u32 = 2;

/// XID v1.1 编解码器。
pub struct IdMix {
    pub(crate) radix: RadixCodec,
    pub(crate) max_objects: usize,
    pub(crate) max_variants: usize,
    pub(crate) check_bits: u32,
    pub(crate) count_bits: u32,
    pub(crate) variant_bits: u32,
    pub(crate) check_mask: u16,
    pub(crate) count_mask: u16,
    pub(crate) variant_mask: u16,
    pub(crate) count_shift: u16,
    pub(crate) variant_shift: u16,
}

impl IdMix {
    pub fn new() -> Result<Self, IdMixError> {
        Self::builder().build()
    }

    pub fn builder() -> IdMixBuilder {
        IdMixBuilder::default()
    }

    pub fn encode(&self, values: &[Value]) -> Result<String, IdMixError> {
        if values.is_empty() {
            return Err(IdMixError::msg("at least one value is required"));
        }
        if values.len() > self.max_objects {
            return Err(IdMixError::msg(format!(
                "too many objects: {} (max {})",
                values.len(),
                self.max_objects
            )));
        }
        let typed = normalize_values(values)?;
        let variant_id = rand::thread_rng().gen_range(0..self.max_variants as i32);
        let data = encode_binary(self, &typed, variant_id)?;
        self.radix.encode_bytes(&data)
    }

    pub fn decode(&self, s: &str) -> Result<Vec<Value>, IdMixError> {
        let data = self.radix.decode_bytes(s)?;
        let typed = decode_binary(self, &data)?;
        materialize_values(&typed)
    }
}

pub struct IdMixBuilder {
    alphabet: String,
    max_objects: usize,
    max_variants: usize,
    check_bits: u32,
}

impl Default for IdMixBuilder {
    fn default() -> Self {
        Self {
            alphabet: DEFAULT_ALPHABET.to_string(),
            max_objects: DEFAULT_MAX_OBJECTS,
            max_variants: DEFAULT_MAX_VARIANTS,
            check_bits: DEFAULT_CHECK_BITS,
        }
    }
}

impl IdMixBuilder {
    pub fn alphabet(mut self, alphabet: impl Into<String>) -> Self {
        self.alphabet = alphabet.into();
        self
    }

    pub fn max_objects(mut self, n: usize) -> Self {
        self.max_objects = n;
        self
    }

    pub fn max_variants(mut self, n: usize) -> Self {
        self.max_variants = n;
        self
    }

    pub fn check_bits(mut self, n: u32) -> Self {
        self.check_bits = n;
        self
    }

    pub fn build(self) -> Result<IdMix, IdMixError> {
        if self.max_objects < 1 {
            return Err(IdMixError::msg("maxObjects must be at least 1"));
        }
        if self.max_variants < 1 {
            return Err(IdMixError::msg("maxVariants must be at least 1"));
        }
        if !(1..=8).contains(&self.check_bits) {
            return Err(IdMixError::msg("checkBits must be between 1 and 8"));
        }
        let radix = RadixCodec::new(&self.alphabet)?;
        let mut m = IdMix {
            radix,
            max_objects: self.max_objects,
            max_variants: self.max_variants,
            check_bits: self.check_bits,
            count_bits: 0,
            variant_bits: 0,
            check_mask: 0,
            count_mask: 0,
            variant_mask: 0,
            count_shift: 0,
            variant_shift: 0,
        };
        m.finalize_layout()?;
        Ok(m)
    }
}

impl IdMix {
    fn finalize_layout(&mut self) -> Result<(), IdMixError> {
        let variant_bits = if self.max_variants <= 1 {
            1
        } else {
            bit_len((self.max_variants - 1) as u32)
        };
        let count_bits = if self.max_objects <= 1 {
            1
        } else {
            bit_len(self.max_objects as u32)
        };
        let total = self.check_bits + count_bits + variant_bits;
        if total > 16 {
            return Err(IdMixError::msg(format!(
                "checkBits({}) + countBits({}) + variantBits({}) = {total} exceeds 16-bit header",
                self.check_bits, count_bits, variant_bits
            )));
        }
        self.count_bits = count_bits;
        self.variant_bits = variant_bits;
        self.check_mask = ((1u16 << self.check_bits) - 1) as u16;
        self.count_mask = (((1u16 << count_bits) - 1) << self.check_bits) as u16;
        self.variant_mask =
            (((1u16 << variant_bits) - 1) << (self.check_bits + count_bits)) as u16;
        self.count_shift = self.check_bits as u16;
        self.variant_shift = (self.check_bits + count_bits) as u16;
        Ok(())
    }
}

fn bit_len(n: u32) -> u32 {
    if n == 0 {
        1
    } else {
        32 - n.leading_zeros()
    }
}
