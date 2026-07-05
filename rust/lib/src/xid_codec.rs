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
    if is_unsigned(tv.otype) && tv.val >= 0 && tv.val <= 15 {
        let wb = width_bits(tv.otype);
        let head = (wb << 4) | (tv.val as u8);
        return Ok(vec![head]);
    }
    if is_signed(tv.otype) && tv.val >= -16 && tv.val <= -1 {
        let wb = width_bits(tv.otype);
        let v = (-tv.val - 1) as u8;
        let head = (1 << 6) | (wb << 4) | v;
        return Ok(vec![head]);
    }
    let (sw, payload) = minimal_complement_bytes(tv.otype, tv.val)?;
    let head = 0x80 | (sw << 4) | tv.otype;
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
    if (head >> 6) & 1 != 0 {
        return Err(IdMixError::msg("reserved bit set in extended mode"));
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
    let mut raw = 0u64;
    for i in 0..num_bytes {
        raw |= (data[1 + i] as u64) << (8 * i);
    }
    let val = reconstruct_int(otype, sw, raw)?;
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

fn target_bits(otype: u8) -> u32 {
    match otype {
        OTYPE_UINT8 | OTYPE_INT8 => 8,
        1 | 5 => 16,
        2 | 6 => 32,
        _ => 64,
    }
}

fn minimal_complement_bytes(otype: u8, val: i64) -> Result<(u8, Vec<u8>), IdMixError> {
    if val == 0 {
        return Ok((0, vec![0x00]));
    }
    if is_unsigned(otype) {
        if val < 0 {
            return Err(IdMixError::msg(format!(
                "negative value {val} for unsigned type"
            )));
        }
        let uval = val as u64;
        for sw in 0u8..4 {
            let size = SW_BYTES[sw as usize];
            if size < 8 && uval >= 1u64 << (size * 8) {
                continue;
            }
            let buf = uint_to_le_bytes(uval, size);
            if buf[size - 1] & 0x80 == 0 {
                return Ok((sw, buf));
            }
        }
        return Err(IdMixError::msg("value too large for unsigned type"));
    }

    let tbits = target_bits(otype);
    let mask = if tbits == 64 {
        u64::MAX
    } else {
        (1u64 << tbits) - 1
    };
    let uval = val as u64 & mask;

    if val < 0 {
        for sw in 0u8..4 {
            let size = SW_BYTES[sw as usize];
            let shift = (size * 8) as u32;
            if shift >= tbits {
                return Ok((sw, uint_to_le_bytes(uval, size)));
            }
            let lower = uval & ((1u64 << shift) - 1);
            let upper = uval >> shift;
            let upper_mask = (1u64 << (tbits - shift)) - 1;
            if upper != upper_mask {
                continue;
            }
            let high_byte = ((lower >> (shift - 8)) & 0xff) as u8;
            if high_byte & 0x80 == 0 {
                continue;
            }
            return Ok((sw, uint_to_le_bytes(lower, size)));
        }
    } else {
        for sw in 0u8..4 {
            let size = SW_BYTES[sw as usize];
            if size < 8 && uval >= 1u64 << (size * 8) {
                continue;
            }
            let buf = uint_to_le_bytes(uval, size);
            if buf[size - 1] & 0x80 == 0 {
                return Ok((sw, buf));
            }
        }
    }

    let sw = match tbits {
        8 => 0,
        16 => 1,
        32 => 2,
        _ => 3,
    };
    let size = SW_BYTES[sw as usize];
    Ok((sw, uint_to_le_bytes(uval, size)))
}

fn uint_to_le_bytes(v: u64, size: usize) -> Vec<u8> {
    let mut buf = vec![0u8; size];
    for i in 0..size {
        buf[i] = (v >> (8 * i)) as u8;
    }
    buf
}

fn reconstruct_int(otype: u8, sw: u8, raw: u64) -> Result<i64, IdMixError> {
    let tbits = target_bits(otype);
    let stored_bits = (SW_BYTES[sw as usize] * 8) as u32;
    if is_unsigned(otype) {
        let mask = if tbits == 64 {
            u64::MAX
        } else {
            (1u64 << tbits) - 1
        };
        return Ok((raw & mask) as i64);
    }
    let sign_bit = (raw >> (stored_bits - 1)) & 1;
    if tbits <= stored_bits {
        let mask = if tbits == 64 {
            u64::MAX
        } else {
            (1u64 << tbits) - 1
        };
        let mut val = raw & mask;
        if sign_bit == 1 && val & (1u64 << (tbits - 1)) != 0 {
            val -= 1u64 << tbits;
        }
        return Ok(val as i64);
    }
    let extended = if sign_bit == 1 {
        let extend_mask = (!((1u64 << stored_bits) - 1)) & ((1u64 << tbits) - 1);
        raw | extend_mask
    } else {
        raw
    };
    let mut val = extended;
    if val >= 1u64 << (tbits - 1) {
        val -= 1u64 << tbits;
    }
    Ok(val as i64)
}

fn validate_range(otype: u8, val: i64) -> Result<(), IdMixError> {
    let ok = match otype {
        OTYPE_UINT8 => (0..=u8::MAX as i64).contains(&val),
        1 => (0..=u16::MAX as i64).contains(&val),
        2 => (0..=u32::MAX as i64).contains(&val),
        OTYPE_UINT64 => val >= 0,
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
