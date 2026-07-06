use crate::error::IdMixError;
use crate::idmix::IdMix;
use crate::typed_value::{TypedValue, OTYPE_INT64, OTYPE_INT8, OTYPE_UINT64, OTYPE_UINT8};

const SW_BYTES: [usize; 4] = [1, 2, 4, 8];

const EMBEDDED_OTYPE: [[u8; 4]; 2] = [
    [OTYPE_UINT8, 1, 2, OTYPE_UINT64],
    [OTYPE_INT8, 5, 6, OTYPE_INT64],
];

pub fn encode_binary(m: &IdMix, typed: &[TypedValue], variant_id: i32) -> Result<Vec<u8>, IdMixError> {
    let mut objects = Vec::new();
    for &tv in typed {
        objects.extend(encode_object(tv)?);
    }
    let mask = ((variant_id * 0x9d + 0x37) & 0xff) as u8;
    for b in &mut objects {
        *b ^= mask;
    }

    let count = typed.len();
    let mut header =
        ((variant_id as u16) << m.variant_shift) | ((count as u16) << m.count_shift);
    let mut data = vec![0u8; 2 + objects.len()];
    data[0] = (header & 0xff) as u8;
    data[1] = (header >> 8) as u8;
    data[2..].copy_from_slice(&objects);

    let xor_sum = data.iter().fold(0u8, |acc, &b| acc ^ b);
    let check = (xor_sum as u16) & m.check_mask;
    header |= check;
    data[0] = (header & 0xff) as u8;
    data[1] = (header >> 8) as u8;
    Ok(data)
}

pub fn decode_binary(m: &IdMix, data: &[u8]) -> Result<Vec<TypedValue>, IdMixError> {
    if data.len() < 2 {
        return Err(IdMixError::msg("invalid data: too short"));
    }
    let header = u16::from_le_bytes([data[0], data[1]]);
    let check = header & m.check_mask;
    let count = ((header & m.count_mask) >> m.count_shift) as usize;
    let variant_id = ((header & m.variant_mask) >> m.variant_shift) as i32;

    if variant_id >= m.max_variants as i32 {
        return Err(IdMixError::msg(format!(
            "invalid variant_id {variant_id} (max {})",
            m.max_variants - 1
        )));
    }
    if count > m.max_objects {
        return Err(IdMixError::msg(format!(
            "invalid count {count} (max {})",
            m.max_objects
        )));
    }

    let mut verify = data.to_vec();
    verify[0] &= !(m.check_mask as u8);
    let xor_sum = verify.iter().fold(0u8, |acc, &b| acc ^ b);
    if (xor_sum as u16) & m.check_mask != check {
        return Err(IdMixError::msg("checksum mismatch"));
    }

    let mut objects = data[2..].to_vec();
    let mask = ((variant_id * 0x9d + 0x37) & 0xff) as u8;
    for b in &mut objects {
        *b ^= mask;
    }

    let mut result = Vec::with_capacity(count);
    let mut pos = 0;
    for _ in 0..count {
        if pos >= objects.len() {
            return Err(IdMixError::msg("premature end of data"));
        }
        let (tv, n) = decode_object(&objects[pos..])?;
        result.push(tv);
        pos += n;
    }
    if pos != objects.len() {
        return Err(IdMixError::msg("extra bytes after data objects"));
    }
    Ok(result)
}

fn encode_object(tv: TypedValue) -> Result<Vec<u8>, IdMixError> {
    validate_range(tv.otype, tv.val)?;
    if let Some(head) = try_embedded_head(tv.otype, tv.val) {
        return Ok(vec![head]);
    }
    let (mag, neg) = magnitude_from_typed(tv);
    let sw = sw_from_magnitude(mag);
    let payload = uint_to_le_bytes(mag, SW_BYTES[sw as usize]);
    let mut head = 0x80 | (sw << 4) | tv.otype;
    if neg {
        head |= 1 << 6;
    }
    let mut out = Vec::with_capacity(1 + payload.len());
    out.push(head);
    out.extend_from_slice(&payload);
    Ok(out)
}

fn decode_object(data: &[u8]) -> Result<(TypedValue, usize), IdMixError> {
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
        return Ok((TypedValue { otype, val }, 1));
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
    let mut mag = 0u64;
    for i in 0..num_bytes {
        mag |= (data[1 + i] as u64) << (8 * i);
    }
    let neg = (head >> 6) & 1 != 0;
    let val = value_from_magnitude(mag, neg);
    validate_range(otype, val)?;
    Ok((TypedValue { otype, val }, 1 + num_bytes))
}

fn is_unsigned(otype: u8) -> bool {
    otype <= OTYPE_UINT64
}

fn is_signed(otype: u8) -> bool {
    otype >= OTYPE_INT8
}

fn width_bits(otype: u8) -> u8 {
    match otype {
        OTYPE_UINT8 | OTYPE_INT8 => 0,
        1 | 5 => 1,
        2 | 6 => 2,
        _ => 3,
    }
}

fn magnitude_from_typed(tv: TypedValue) -> (u64, bool) {
    if is_unsigned(tv.otype) {
        return (tv.val as u64, false);
    }
    if tv.val < 0 {
        return (tv.val.wrapping_neg() as u64, true);
    }
    (tv.val as u64, false)
}

fn sw_from_magnitude(mag: u64) -> u8 {
    if mag < 256 {
        return 0;
    }
    if mag < 65536 {
        return 1;
    }
    if mag < 4294967296 {
        return 2;
    }
    3
}

fn try_embedded_head(otype: u8, val: i64) -> Option<u8> {
    let (mag, neg) = magnitude_from_typed(TypedValue { otype, val });
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

fn value_from_magnitude(mag: u64, neg: bool) -> i64 {
    if !neg {
        return mag as i64;
    }
    if mag == 1u64 << 63 {
        return i64::MIN;
    }
    -(mag as i64)
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
        1 => (0..=u16::MAX as i64).contains(&val),
        2 => (0..=u32::MAX as i64).contains(&val),
        OTYPE_UINT64 => true,
        OTYPE_INT8 => (i8::MIN as i64..=i8::MAX as i64).contains(&val),
        5 => (i16::MIN as i64..=i16::MAX as i64).contains(&val),
        6 => (i32::MIN as i64..=i32::MAX as i64).contains(&val),
        OTYPE_INT64 => true,
        _ => false,
    };
    if ok {
        Ok(())
    } else {
        Err(IdMixError::msg(format!("value {val} out of range for otype {otype}")))
    }
}
