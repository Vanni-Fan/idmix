"""XID v1.1 二进制层编解码。"""

from __future__ import annotations

import struct
from typing import TYPE_CHECKING

from .typed_value import OType, TypedValue

if TYPE_CHECKING:
    from .idmix import IdMix

SW_BYTES = (1, 2, 4, 8)

EMBEDDED_OTYPE = (
    (OType.UINT8, OType.UINT16, OType.UINT32, OType.UINT64),
    (OType.INT8, OType.INT16, OType.INT32, OType.INT64),
)


def encode_binary(m: IdMix, typed: list[TypedValue], variant_id: int) -> bytes:
    """typedValue 序列 → XID 二进制块。"""
    objects = bytearray()
    for tv in typed:
        objects.extend(encode_object(tv))
    mask = (variant_id * 0x9D + 0x37) & 0xFF
    objects = bytes(b ^ mask for b in objects)

    count = len(typed)
    header = (variant_id << m.variant_shift) | (count << m.count_shift)
    data = bytearray(struct.pack("<H", header) + objects)

    xor_sum = 0
    for b in data:
        xor_sum ^= b
    check = xor_sum & m.check_mask
    header |= check
    data[0:2] = struct.pack("<H", header)
    return bytes(data)


def decode_binary(m: IdMix, data: bytes) -> list[TypedValue]:
    """XID 二进制块 → typedValue 序列。"""
    if len(data) < 2:
        raise ValueError("invalid data: too short")
    header = struct.unpack("<H", data[:2])[0]
    check = header & m.check_mask
    count = (header & m.count_mask) >> m.count_shift
    variant_id = (header & m.variant_mask) >> m.variant_shift

    if variant_id >= m.max_variants:
        raise ValueError(f"invalid variant_id {variant_id}")
    if count > m.max_objects:
        raise ValueError(f"invalid count {count}")

    verify = bytearray(data)
    verify[0] &= ~m.check_mask & 0xFF
    xor_sum = 0
    for b in verify:
        xor_sum ^= b
    if (xor_sum & m.check_mask) != check:
        raise ValueError("checksum mismatch")

    objects = bytearray(data[2:])
    mask = (variant_id * 0x9D + 0x37) & 0xFF
    objects = bytes(b ^ mask for b in objects)

    result: list[TypedValue] = []
    pos = 0
    for i in range(count):
        if pos >= len(objects):
            raise ValueError("premature end of data")
        tv, n = decode_object(objects[pos:])
        result.append(tv)
        pos += n
    if pos != len(objects):
        raise ValueError("extra bytes after data objects")
    return result


def encode_object(tv: TypedValue) -> bytes:
    validate_range(tv.otype, tv.val)
    if _is_unsigned(tv.otype) and 0 <= tv.val <= 15:
        wb = _width_bits(tv.otype)
        return bytes([(wb << 4) | tv.val])
    if _is_signed(tv.otype) and -16 <= tv.val <= -1:
        wb = _width_bits(tv.otype)
        v = -tv.val - 1
        return bytes([(1 << 6) | (wb << 4) | v])
    sw, payload = _minimal_complement_bytes(tv.otype, tv.val)
    return bytes([0x80 | (sw << 4) | tv.otype]) + payload


def decode_object(data: bytes) -> tuple[TypedValue, int]:
    if not data:
        raise ValueError("truncated object header")
    head = data[0]
    if (head & 0x80) == 0:
        sign = (head >> 6) & 1
        wb = (head >> 4) & 0x03
        v = head & 0x0F
        otype = EMBEDDED_OTYPE[sign][wb]
        val = v if sign == 0 else -v - 1
        return TypedValue(otype, val), 1
    if (head >> 6) & 1:
        raise ValueError("reserved bit set in extended mode")
    sw = (head >> 4) & 0x03
    otype = head & 0x0F
    if otype > OType.INT64:
        raise ValueError(f"invalid otype {otype}")
    num_bytes = SW_BYTES[sw]
    if len(data) < 1 + num_bytes:
        raise ValueError("truncated object payload")
    raw = 0
    for i in range(num_bytes):
        raw |= data[1 + i] << (8 * i)
    val = _reconstruct_int(otype, sw, raw)
    return TypedValue(otype, val), 1 + num_bytes


def _is_unsigned(otype: int) -> bool:
    return otype <= OType.UINT64


def _is_signed(otype: int) -> bool:
    return otype >= OType.INT8


def _width_bits(otype: int) -> int:
    return {OType.UINT8: 0, OType.INT8: 0, OType.UINT16: 1, OType.INT16: 1,
            OType.UINT32: 2, OType.INT32: 2}.get(otype, 3)


def _target_bits(otype: int) -> int:
    return {OType.UINT8: 8, OType.INT8: 8, OType.UINT16: 16, OType.INT16: 16,
            OType.UINT32: 32, OType.INT32: 32}.get(otype, 64)


def _minimal_complement_bytes(otype: int, val: int) -> tuple[int, bytes]:
    if val == 0:
        return 0, b"\x00"
    if _is_unsigned(otype):
        if val < 0:
            raise ValueError(f"negative value {val} for unsigned type")
        uval = val
        for sw in range(4):
            size = SW_BYTES[sw]
            if size < 8 and uval >= (1 << (size * 8)):
                continue
            buf = _uint_to_le_bytes(uval, size)
            if buf[-1] & 0x80 == 0:
                return sw, buf
        raise ValueError("value too large for unsigned type")

    tbits = _target_bits(otype)
    mask = (1 << tbits) - 1 if tbits < 64 else (1 << 64) - 1
    uval = val & mask

    if val < 0:
        for sw in range(4):
            size = SW_BYTES[sw]
            shift = size * 8
            if shift >= tbits:
                return sw, _uint_to_le_bytes(uval, size)
            lower = uval & ((1 << shift) - 1)
            upper = uval >> shift
            upper_mask = (1 << (tbits - shift)) - 1
            if upper != upper_mask:
                continue
            high_byte = (lower >> (shift - 8)) & 0xFF
            if high_byte & 0x80 == 0:
                continue
            return sw, _uint_to_le_bytes(lower, size)
    else:
        for sw in range(4):
            size = SW_BYTES[sw]
            if size < 8 and uval >= (1 << (size * 8)):
                continue
            buf = _uint_to_le_bytes(uval, size)
            if buf[-1] & 0x80 == 0:
                return sw, buf

    sw = {8: 0, 16: 1, 32: 2}.get(tbits, 3)
    return sw, _uint_to_le_bytes(uval, SW_BYTES[sw])


def _uint_to_le_bytes(v: int, size: int) -> bytes:
    return bytes((v >> (8 * i)) & 0xFF for i in range(size))


def _reconstruct_int(otype: int, sw: int, raw: int) -> int:
    tbits = _target_bits(otype)
    stored_bits = SW_BYTES[sw] * 8
    if _is_unsigned(otype):
        mask = (1 << tbits) - 1 if tbits < 64 else (1 << 64) - 1
        return raw & mask
    sign_bit = (raw >> (stored_bits - 1)) & 1
    if tbits <= stored_bits:
        mask = (1 << tbits) - 1 if tbits < 64 else (1 << 64) - 1
        val = raw & mask
        if sign_bit == 1 and val & (1 << (tbits - 1)):
            val -= 1 << tbits
        return val
    if sign_bit == 1:
        extend_mask = ((~((1 << stored_bits) - 1)) & ((1 << tbits) - 1)) if tbits < 64 else 0
        extended = raw | extend_mask
    else:
        extended = raw
    if extended >= (1 << (tbits - 1)):
        extended -= 1 << tbits
    return extended


def _validate_range(otype: int, val: int) -> None:
    limits = {
        OType.UINT8: (0, 0xFF),
        OType.UINT16: (0, 0xFFFF),
        OType.UINT32: (0, 0xFFFFFFFF),
        OType.UINT64: (0, None),
        OType.INT8: (-128, 127),
        OType.INT16: (-32768, 32767),
        OType.INT32: (-2147483648, 2147483647),
        OType.INT64: (None, None),
    }
    lo, hi = limits.get(otype, (None, None))
    if lo is not None and val < lo:
        raise ValueError(f"value {val} out of range for otype {otype}")
    if hi is not None and val > hi:
        raise ValueError(f"value {val} out of range for otype {otype}")


def validate_range(otype: int, val: int) -> None:
    _validate_range(otype, val)
