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
    if (head := _try_embedded_head(tv.otype, tv.val)) is not None:
        return bytes([head])
    mag, neg = _magnitude_from_typed(tv.otype, tv.val)
    sw = _sw_from_magnitude(mag)
    payload = _uint_to_le_bytes(mag, SW_BYTES[sw])
    head = 0x80 | (sw << 4) | tv.otype
    if neg:
        head |= 1 << 6
    return bytes([head]) + payload


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
    sw = (head >> 4) & 0x03
    otype = head & 0x0F
    if otype > OType.INT64:
        raise ValueError(f"invalid otype {otype}")
    num_bytes = SW_BYTES[sw]
    if len(data) < 1 + num_bytes:
        raise ValueError("truncated object payload")
    mag = 0
    for i in range(num_bytes):
        mag |= data[1 + i] << (8 * i)
    neg = ((head >> 6) & 1) != 0
    val = _value_from_magnitude(mag, neg)
    validate_range(otype, val)
    return TypedValue(otype, val), 1 + num_bytes


def _is_unsigned(otype: int) -> bool:
    return otype <= OType.UINT64


def _is_signed(otype: int) -> bool:
    return otype >= OType.INT8


def _width_bits(otype: int) -> int:
    return {OType.UINT8: 0, OType.INT8: 0, OType.UINT16: 1, OType.INT16: 1,
            OType.UINT32: 2, OType.INT32: 2}.get(otype, 3)


def _magnitude_from_typed(otype: int, val: int) -> tuple[int, bool]:
    if _is_unsigned(otype):
        return val & 0xFFFFFFFFFFFFFFFF, False
    if val < 0:
        return -val, True
    return val, False


def _sw_from_magnitude(mag: int) -> int:
    if mag < 256:
        return 0
    if mag < 65536:
        return 1
    if mag < 4294967296:
        return 2
    return 3


def _try_embedded_head(otype: int, val: int) -> int | None:
    mag, neg = _magnitude_from_typed(otype, val)
    if mag >= 17:
        return None
    wb = _width_bits(otype)
    if mag == 16:
        if neg:
            return (1 << 6) | (wb << 4) | 15
        return None
    if neg:
        return (1 << 6) | (wb << 4) | (mag - 1)
    return (wb << 4) | mag


def _value_from_magnitude(mag: int, neg: bool) -> int:
    if not neg:
        return mag
    if mag == 1 << 63:
        return -(1 << 63)
    return -mag


def _uint_to_le_bytes(v: int, size: int) -> bytes:
    return bytes((v >> (8 * i)) & 0xFF for i in range(size))


def _validate_range(otype: int, val: int) -> None:
    limits = {
        OType.UINT8: (0, 0xFF),
        OType.UINT16: (0, 0xFFFF),
        OType.UINT32: (0, 0xFFFFFFFF),
        OType.UINT64: (None, None),
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
