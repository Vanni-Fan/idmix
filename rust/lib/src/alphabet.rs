use std::collections::HashMap;

use num_bigint::BigUint;
use num_traits::{ToPrimitive, Zero};

use crate::error::IdMixError;

pub struct RadixCodec {
    pub base: usize,
    chars: Vec<char>,
    from_custom: HashMap<char, usize>,
}

impl RadixCodec {
    pub fn new(alphabet: &str) -> Result<Self, IdMixError> {
        let chars: Vec<char> = alphabet.chars().collect();
        if chars.len() < 2 {
            return Err(IdMixError::msg(
                "alphabet must have at least 2 unique characters",
            ));
        }
        let mut from_custom = HashMap::with_capacity(chars.len());
        for (i, &ch) in chars.iter().enumerate() {
            if from_custom.insert(ch, i).is_some() {
                return Err(IdMixError::msg(format!(
                    "alphabet contains duplicate character {ch:?}"
                )));
            }
        }
        Ok(Self {
            base: chars.len(),
            chars,
            from_custom,
        })
    }

    pub fn encode_bytes(&self, data: &[u8]) -> Result<String, IdMixError> {
        if data.is_empty() {
            return Ok(self.chars[0].to_string());
        }
        let mut wrapped = Vec::with_capacity(2 + data.len());
        let len = data.len() as u16;
        wrapped.push((len >> 8) as u8);
        wrapped.push((len & 0xff) as u8);
        wrapped.extend_from_slice(data);
        let n = BigUint::from_bytes_be(&wrapped);
        Ok(self.int_to_string(&n))
    }

    pub fn decode_bytes(&self, s: &str) -> Result<Vec<u8>, IdMixError> {
        if s.is_empty() {
            return Err(IdMixError::msg("empty string"));
        }
        let n = self.string_to_int(s)?;
        let raw = n.to_bytes_be();
        for pad in 0..=1 {
            let mut buf = vec![0u8; pad + raw.len()];
            buf[pad..].copy_from_slice(&raw);
            if buf.len() < 2 {
                continue;
            }
            let data_len = u16::from_be_bytes([buf[0], buf[1]]) as usize;
            if buf.len() != 2 + data_len {
                continue;
            }
            return Ok(buf[2..].to_vec());
        }
        Err(IdMixError::msg("invalid encoded data length"))
    }

    fn int_to_string(&self, n: &BigUint) -> String {
        if n.is_zero() {
            return self.chars[0].to_string();
        }
        let base = BigUint::from(self.base as u64);
        let zero = BigUint::from(0u8);
        let mut num = n.clone();
        let mut chars = Vec::new();
        while num > zero {
            let rem = &num % &base;
            num /= &base;
            let idx = rem.to_u32().unwrap_or(0) as usize;
            chars.push(self.chars[idx]);
        }
        chars.iter().rev().collect()
    }

    fn string_to_int(&self, s: &str) -> Result<BigUint, IdMixError> {
        let base = BigUint::from(self.base as u64);
        let mut n = BigUint::from(0u8);
        for ch in s.chars() {
            let idx = *self
                .from_custom
                .get(&ch)
                .ok_or_else(|| IdMixError::msg(format!("invalid character {ch:?}")))?;
            n *= &base;
            n += BigUint::from(idx as u64);
        }
        Ok(n)
    }
}
