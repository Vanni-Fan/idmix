use crate::error::IdMixError;
use crate::number::{materialize_objects, normalize_objects};
use crate::typed_value::Value;

pub const MAX_STRING_LEN: usize = 63;

const OTYPE_UINT8: u8 = 0;
const OTYPE_UINT16: u8 = 1;
const OTYPE_UINT32: u8 = 2;
const OTYPE_UINT64: u8 = 3;
const OTYPE_INT8: u8 = 4;
const OTYPE_INT16: u8 = 5;
const OTYPE_INT32: u8 = 6;
const OTYPE_INT64: u8 = 7;

const SW_BYTES: [usize; 4] = [1, 2, 4, 8];

const EMBEDDED_OTYPE: [[u8; 4]; 2] = [
    [OTYPE_UINT8, OTYPE_UINT16, OTYPE_UINT32, OTYPE_UINT64],
    [OTYPE_INT8, OTYPE_INT16, OTYPE_INT32, OTYPE_INT64],
];

/// Internal unified representation for integers or short strings.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DataObject {
    pub is_string: bool,
    pub otype: u8,
    pub val: i64,
    pub str: Vec<u8>,
}

impl DataObject {
    pub fn string(bytes: &[u8]) -> Self {
        Self {
            is_string: true,
            otype: 0,
            val: 0,
            str: bytes.to_vec(),
        }
    }

    pub fn integer(otype: u8, val: i64) -> Self {
        Self {
            is_string: false,
            otype,
            val,
            str: Vec::new(),
        }
    }
}

/// IDX v1.2 binary encoder/decoder.
#[derive(Debug, Clone)]
pub struct Idx {
    pub max_objects: usize,
    pub max_variants: usize,
    pub check_bits: u32,
    pub check_mask: u8,
}

impl Default for Idx {
    fn default() -> Self {
        Self {
            max_objects: 255,
            max_variants: 32,
            check_bits: 2,
            check_mask: 0x03,
        }
    }
}

impl Idx {
    pub fn new() -> Result<Self, IdMixError> {
        Self::builder().build()
    }

    pub fn builder() -> IdxBuilder {
        IdxBuilder::default()
    }

    pub fn encode(&self, values: &[Value]) -> Result<Vec<u8>, IdMixError> {
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
        let objects = normalize_objects(values)?;
        self.encode_binary(&objects, 0)
    }

    pub fn encode_with_variant(&self, variant_id: i32, values: &[Value]) -> Result<Vec<u8>, IdMixError> {
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
        let objects = normalize_objects(values)?;
        self.encode_binary(&objects, variant_id)
    }

    pub fn decode(&self, data: &[u8]) -> Result<Vec<Value>, IdMixError> {
        let objects = self.decode_binary(data)?;
        materialize_objects(&objects)
    }

    pub(crate) fn encode_binary(&self, objects: &[DataObject], variant_id: i32) -> Result<Vec<u8>, IdMixError> {
        if variant_id < 0 || variant_id >= self.max_variants as i32 {
            return Err(IdMixError::msg(format!(
                "invalid variant_id {variant_id} (max {})",
                self.max_variants - 1
            )));
        }

        let mut obj_bytes = Vec::with_capacity(objects.len() * 2);
        for obj in objects {
            obj_bytes.extend(encode_object(obj)?);
        }

        let mask = ((variant_id * 0x9d + 0x37) & 0xff) as u8;
        for b in &mut obj_bytes {
            *b ^= mask;
        }

        let count = objects.len();
        let header_len = if count > 1 { 2 } else { 1 };
        let mut data = vec![0u8; header_len + obj_bytes.len()];

        if count == 1 {
            data[0] = (variant_id as u8) << self.check_bits;
        } else {
            data[0] = 0x80 | ((variant_id as u8) << self.check_bits);
            data[1] = count as u8;
        }
        data[header_len..].copy_from_slice(&obj_bytes);

        let xor_sum = data.iter().fold(0u8, |acc, &b| acc ^ b);
        let check = xor_sum & self.check_mask;
        data[0] |= check;
        Ok(data)
    }

    pub(crate) fn decode_binary(&self, data: &[u8]) -> Result<Vec<DataObject>, IdMixError> {
        if data.is_empty() {
            return Err(IdMixError::msg("invalid data: too short"));
        }

        let byte0 = data[0];
        let check = byte0 & self.check_mask;
        let multi = byte0 & 0x80 != 0;
        let variant_id = ((byte0 & 0x7f) >> self.check_bits) as i32;

        if variant_id >= self.max_variants as i32 {
            return Err(IdMixError::msg(format!(
                "invalid variant_id {variant_id} (max {})",
                self.max_variants - 1
            )));
        }

        let (header_len, count) = if multi {
            if data.len() < 2 {
                return Err(IdMixError::msg("invalid data: missing count byte"));
            }
            let count = data[1] as usize;
            if count < 2 || count > self.max_objects {
                return Err(IdMixError::msg(format!("invalid count {count}")));
            }
            (2, count)
        } else {
            (1, 1)
        };

        let mut verify = data.to_vec();
        verify[0] &= !self.check_mask;
        let xor_sum = verify.iter().fold(0u8, |acc, &b| acc ^ b);
        if xor_sum & self.check_mask != check {
            return Err(IdMixError::msg("checksum mismatch"));
        }

        let mut obj_data = data[header_len..].to_vec();
        let mask = ((variant_id * 0x9d + 0x37) & 0xff) as u8;
        for b in &mut obj_data {
            *b ^= mask;
        }

        let mut result = Vec::with_capacity(count);
        let mut pos = 0;
        for _ in 0..count {
            if pos >= obj_data.len() {
                return Err(IdMixError::msg("premature end of data"));
            }
            let (obj, n) = decode_object(&obj_data[pos..])?;
            result.push(obj);
            pos += n;
        }
        if pos != obj_data.len() {
            return Err(IdMixError::msg("extra bytes after data objects"));
        }
        Ok(result)
    }
}

#[derive(Debug)]
pub struct IdxBuilder {
    max_objects: usize,
    max_variants: usize,
    check_bits: u32,
}

impl Default for IdxBuilder {
    fn default() -> Self {
        Self {
            max_objects: 255,
            max_variants: 32,
            check_bits: 2,
        }
    }
}

impl IdxBuilder {
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

    pub fn build(self) -> Result<Idx, IdMixError> {
        if self.max_objects < 1 || self.max_objects > 255 {
            return Err(IdMixError::msg("maxObjects must be between 1 and 255"));
        }
        if self.max_variants < 1 || self.max_variants > 32 {
            return Err(IdMixError::msg("maxVariants must be between 1 and 32"));
        }
        if !(1..=2).contains(&self.check_bits) {
            return Err(IdMixError::msg("checkBits must be 1 or 2"));
        }
        Ok(Idx {
            max_objects: self.max_objects,
            max_variants: self.max_variants,
            check_bits: self.check_bits,
            check_mask: ((1u32 << self.check_bits) - 1) as u8,
        })
    }
}

fn encode_object(obj: &DataObject) -> Result<Vec<u8>, IdMixError> {
    if obj.is_string {
        let n = obj.str.len();
        if n < 1 || n > MAX_STRING_LEN {
            return Err(IdMixError::msg(format!(
                "string length {n} out of range [1, {MAX_STRING_LEN}]"
            )));
        }
        let mut out = vec![0u8; 1 + n];
        out[0] = 0xC0 | (n as u8);
        out[1..].copy_from_slice(&obj.str);
        return Ok(out);
    }

    validate_range(obj.otype, obj.val)?;
    if let Some(head) = try_embedded_head(obj.otype, obj.val) {
        return Ok(vec![head]);
    }

    let (sw, payload) = payload_for_number(obj.otype, obj.val)?;
    let head = 0x80 | (sw << 4) | obj.otype;
    let mut out = Vec::with_capacity(1 + payload.len());
    out.push(head);
    out.extend_from_slice(&payload);
    Ok(out)
}

fn decode_object(data: &[u8]) -> Result<(DataObject, usize), IdMixError> {
    if data.is_empty() {
        return Err(IdMixError::msg("truncated object header"));
    }
    let head = data[0];
    if head & 0x80 == 0 {
        let sign = (head >> 6) & 1;
        let wb = ((head >> 4) & 0x03) as usize;
        let v = head & 0x0f;
        let otype = EMBEDDED_OTYPE[sign as usize][wb];
        let val = if sign == 0 {
            v as i64
        } else {
            -(v as i64) - 1
        };
        return Ok((DataObject::integer(otype, val), 1));
    }

    if head & 0x40 != 0 {
        let n = (head & 0x3f) as usize;
        if n < 1 || n > MAX_STRING_LEN {
            return Err(IdMixError::msg(format!("invalid string length {n}")));
        }
        if data.len() < 1 + n {
            return Err(IdMixError::msg("truncated string payload"));
        }
        return Ok((DataObject::string(&data[1..1 + n]), 1 + n));
    }

    let sw = (head >> 4) & 0x03;
    let otype = head & 0x0f;
    if otype > OTYPE_INT64 {
        return Err(IdMixError::msg(format!("invalid otype {otype}")));
    }
    let num_bytes = SW_BYTES[sw as usize];
    if data.len() < 1 + num_bytes {
        return Err(IdMixError::msg("truncated object payload"));
    }
    let val = value_from_payload(otype, &data[1..1 + num_bytes])?;
    validate_range(otype, val)?;
    Ok((DataObject::integer(otype, val), 1 + num_bytes))
}

fn payload_for_number(otype: u8, val: i64) -> Result<(u8, Vec<u8>), IdMixError> {
    if otype == OTYPE_UINT64 {
        let mag = val as u64;
        let sw = sw_from_magnitude(mag);
        return Ok((sw, uint_to_le_bytes(mag, SW_BYTES[sw as usize])));
    }
    if is_unsigned(otype) {
        if val < 0 {
            return Err(IdMixError::msg(format!(
                "negative value {val} for unsigned otype {otype}"
            )));
        }
        let mag = val as u64;
        let sw = sw_from_magnitude(mag);
        return Ok((sw, uint_to_le_bytes(mag, SW_BYTES[sw as usize])));
    }
    let sw = sw_from_signed_value(val);
    Ok((sw, signed_to_le_bytes(val, SW_BYTES[sw as usize])))
}

fn value_from_payload(otype: u8, payload: &[u8]) -> Result<i64, IdMixError> {
    if is_unsigned(otype) {
        let mag = le_bytes_to_uint(payload);
        if otype != OTYPE_UINT64 && mag > i64::MAX as u64 {
            return Err(IdMixError::msg(format!(
                "value out of range for otype {otype}"
            )));
        }
        return Ok(mag as i64);
    }
    Ok(le_bytes_to_signed(payload))
}

fn sw_from_signed_value(val: i64) -> u8 {
    if (i8::MIN as i64..=i8::MAX as i64).contains(&val) {
        return 0;
    }
    if (i16::MIN as i64..=i16::MAX as i64).contains(&val) {
        return 1;
    }
    if (i32::MIN as i64..=i32::MAX as i64).contains(&val) {
        return 2;
    }
    3
}

fn signed_to_le_bytes(val: i64, size: usize) -> Vec<u8> {
    let mut buf = vec![0u8; size];
    let u = val as u64;
    for i in 0..size {
        buf[i] = (u >> (8 * i)) as u8;
    }
    buf
}

fn le_bytes_to_signed(payload: &[u8]) -> i64 {
    let mut u = 0u64;
    for (i, &b) in payload.iter().enumerate() {
        u |= (b as u64) << (8 * i);
    }
    let size = payload.len();
    let shift = 64 - size * 8;
    ((u << shift) as i64) >> shift
}

fn le_bytes_to_uint(payload: &[u8]) -> u64 {
    let mut u = 0u64;
    for (i, &b) in payload.iter().enumerate() {
        u |= (b as u64) << (8 * i);
    }
    u
}

fn is_unsigned(otype: u8) -> bool {
    otype <= OTYPE_UINT64
}

fn width_bits(otype: u8) -> u8 {
    match otype {
        OTYPE_UINT8 | OTYPE_INT8 => 0,
        OTYPE_UINT16 | OTYPE_INT16 => 1,
        OTYPE_UINT32 | OTYPE_INT32 => 2,
        _ => 3,
    }
}

fn magnitude_from_typed(otype: u8, val: i64) -> (u64, bool) {
    if is_unsigned(otype) {
        return (val as u64, false);
    }
    if val < 0 {
        return (val.wrapping_neg() as u64, true);
    }
    (val as u64, false)
}

fn sw_from_magnitude(mag: u64) -> u8 {
    if mag < 256 {
        return 0;
    }
    if mag < 65536 {
        return 1;
    }
    if mag < 4_294_967_296 {
        return 2;
    }
    3
}

fn try_embedded_head(otype: u8, val: i64) -> Option<u8> {
    let (mag, neg) = magnitude_from_typed(otype, val);
    if mag >= 17 {
        return None;
    }
    let wb = width_bits(otype);
    if mag == 16 {
        if neg {
            return Some((1 << 6) | (wb << 4) | 15);
        }
        return None;
    }
    if neg {
        Some((1 << 6) | (wb << 4) | ((mag - 1) as u8))
    } else {
        Some((wb << 4) | (mag as u8))
    }
}

fn uint_to_le_bytes(v: u64, size: usize) -> Vec<u8> {
    let mut buf = vec![0u8; size];
    for i in 0..size {
        buf[i] = (v >> (8 * i)) as u8;
    }
    buf
}

fn validate_range(otype: u8, val: i64) -> Result<(), IdMixError> {
    let ok = match otype {
        OTYPE_UINT8 => (0..=u8::MAX as i64).contains(&val),
        OTYPE_UINT16 => (0..=u16::MAX as i64).contains(&val),
        OTYPE_UINT32 => (0..=u32::MAX as i64).contains(&val),
        OTYPE_UINT64 => true,
        OTYPE_INT8 => (i8::MIN as i64..=i8::MAX as i64).contains(&val),
        OTYPE_INT16 => (i16::MIN as i64..=i16::MAX as i64).contains(&val),
        OTYPE_INT32 => (i32::MIN as i64..=i32::MAX as i64).contains(&val),
        OTYPE_INT64 => true,
        _ => false,
    };
    if ok {
        Ok(())
    } else {
        Err(IdMixError::msg(format!(
            "value {val} out of range for otype {otype}"
        )))
    }
}
