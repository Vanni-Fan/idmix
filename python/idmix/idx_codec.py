"""IDX 二进制层编解码（自描述整数/短字符串序列）。"""

from __future__ import annotations

from typing import Any, Callable

from .number import (
    MAX_STRING_LEN,
    OTYPE_INT64,
    OTYPE_UINT64,
    DataObject,
    materialize_objects,
    normalize_objects,
)

SW_BYTES = (1, 2, 4, 8)

EMBEDDED_OTYPE = (
    (0, 1, 2, 3),  # uint8, uint16, uint32, uint64
    (4, 5, 6, 7),  # int8, int16, int32, int64
)


class Idx:
    """IDX 二进制编解码器，可独立于 idmix 文本层使用。"""

    def __init__(
        self,
        max_objects: int = 255,
        max_variants: int = 32,
        check_bits: int = 2,
    ) -> None:
        self.max_objects = max_objects
        self.max_variants = max_variants
        self.check_bits = check_bits
        self.check_mask = (1 << check_bits) - 1

    @classmethod
    def new(cls, *opts: Callable[[Idx], None]) -> Idx:
        idx = cls()
        for opt in opts:
            opt(idx)
        return idx

    def encode(self, *values: Any) -> bytes:
        if len(values) < 1:
            raise ValueError("at least one value is required")
        if len(values) > self.max_objects:
            raise ValueError(f"too many objects: {len(values)} (max {self.max_objects})")
        objects = normalize_objects(list(values))
        return self._encode_binary(objects, 0)

    def encode_with_variant(self, variant_id: int, *values: Any) -> bytes:
        if len(values) < 1:
            raise ValueError("at least one value is required")
        if len(values) > self.max_objects:
            raise ValueError(f"too many objects: {len(values)} (max {self.max_objects})")
        objects = normalize_objects(list(values))
        return self._encode_binary(objects, variant_id)

    def decode(self, data: bytes) -> list[Any]:
        objects = self._decode_binary(data)
        return materialize_objects(objects)

    def _encode_binary(self, objects: list[DataObject], variant_id: int) -> bytes:
        if variant_id < 0 or variant_id >= self.max_variants:
            raise ValueError(f"invalid variant_id {variant_id} (max {self.max_variants - 1})")

        obj_bytes = bytearray()
        for obj in objects:
            obj_bytes.extend(_encode_object(obj))

        mask = (variant_id * 0x9D + 0x37) & 0xFF
        for i in range(len(obj_bytes)):
            obj_bytes[i] ^= mask

        count = len(objects)
        header_len = 1 if count == 1 else 2
        data = bytearray(header_len + len(obj_bytes))
        if count == 1:
            data[0] = variant_id << self.check_bits
        else:
            data[0] = 0x80 | (variant_id << self.check_bits)
            data[1] = count
        data[header_len:] = obj_bytes

        xor_sum = 0
        for b in data:
            xor_sum ^= b
        check = xor_sum & self.check_mask
        data[0] |= check
        return bytes(data)

    def _decode_binary(self, data: bytes) -> list[DataObject]:
        if len(data) < 1:
            raise ValueError("invalid data: too short")

        byte0 = data[0]
        check = byte0 & self.check_mask
        multi = (byte0 & 0x80) != 0
        variant_id = (byte0 & 0x7F) >> self.check_bits

        if variant_id >= self.max_variants:
            raise ValueError(f"invalid variant_id {variant_id} (max {self.max_variants - 1})")

        header_len = 1
        count = 1
        if multi:
            if len(data) < 2:
                raise ValueError("invalid data: missing count byte")
            header_len = 2
            count = data[1]
            if count < 2 or count > self.max_objects:
                raise ValueError(f"invalid count {count}")

        verify = bytearray(data)
        verify[0] &= ~self.check_mask & 0xFF
        xor_sum = 0
        for b in verify:
            xor_sum ^= b
        if (xor_sum & self.check_mask) != check:
            raise ValueError("checksum mismatch")

        obj_data = bytearray(data[header_len:])
        mask = (variant_id * 0x9D + 0x37) & 0xFF
        for i in range(len(obj_data)):
            obj_data[i] ^= mask

        result: list[DataObject] = []
        pos = 0
        for i in range(count):
            if pos >= len(obj_data):
                raise ValueError("premature end of data")
            try:
                obj, n = _decode_object(bytes(obj_data[pos:]))
            except ValueError as e:
                raise ValueError(f"object[{i}]: {e}") from e
            result.append(obj)
            pos += n
        if pos != len(obj_data):
            raise ValueError("extra bytes after data objects")
        return result


def with_max_objects(n: int) -> Callable[[Idx], None]:
    def apply(idx: Idx) -> None:
        if n < 1 or n > 255:
            raise ValueError("maxObjects must be between 1 and 255")
        idx.max_objects = n

    return apply


def with_max_variants(n: int) -> Callable[[Idx], None]:
    def apply(idx: Idx) -> None:
        if n < 1 or n > 32:
            raise ValueError("maxVariants must be between 1 and 32")
        idx.max_variants = n

    return apply


def with_check_bits(n: int) -> Callable[[Idx], None]:
    def apply(idx: Idx) -> None:
        if n < 1 or n > 2:
            raise ValueError("checkBits must be 1 or 2")
        idx.check_bits = n
        idx.check_mask = (1 << n) - 1

    return apply


def _encode_object(obj: DataObject) -> bytes:
    if obj.is_string:
        n = len(obj.str)
        if n < 1 or n > MAX_STRING_LEN:
            raise ValueError(f"string length {n} out of range [1, {MAX_STRING_LEN}]")
        out = bytearray(1 + n)
        out[0] = 0xC0 | n
        out[1:] = obj.str
        return bytes(out)

    _validate_range(obj.otype, obj.val)
    head = _try_embedded_head(obj.otype, obj.val)
    if head is not None:
        return bytes([head])

    sw, payload = _payload_for_number(obj.otype, obj.val)
    head = 0x80 | (sw << 4) | obj.otype
    return bytes([head]) + payload


def _decode_object(data: bytes) -> tuple[DataObject, int]:
    if len(data) < 1:
        raise ValueError("truncated object header")
    head = data[0]
    if (head & 0x80) == 0:
        sign = (head >> 6) & 1
        wb = (head >> 4) & 0x03
        v = head & 0x0F
        otype = EMBEDDED_OTYPE[sign][wb]
        val = v if sign == 0 else -v - 1
        return DataObject(otype=otype, val=val), 1

    if (head & 0x40) != 0:
        n = head & 0x3F
        if n < 1 or n > MAX_STRING_LEN:
            raise ValueError(f"invalid string length {n}")
        if len(data) < 1 + n:
            raise ValueError("truncated string payload")
        return DataObject(is_string=True, str=bytes(data[1 : 1 + n])), 1 + n

    sw = (head >> 4) & 0x03
    otype = head & 0x0F
    if otype > OTYPE_INT64:
        raise ValueError(f"invalid otype {otype}")
    num_bytes = SW_BYTES[sw]
    if len(data) < 1 + num_bytes:
        raise ValueError("truncated object payload")
    val = _value_from_payload(otype, data[1 : 1 + num_bytes])
    _validate_range(otype, val)
    return DataObject(otype=otype, val=val), 1 + num_bytes


def _payload_for_number(otype: int, val: int) -> tuple[int, bytes]:
    if otype == OTYPE_UINT64:
        mag = val & 0xFFFFFFFFFFFFFFFF
        sw = _sw_from_magnitude(mag)
        return sw, _uint_to_le_bytes(mag, SW_BYTES[sw])
    if _is_unsigned(otype):
        if val < 0:
            raise ValueError(f"negative value {val} for unsigned otype {otype}")
        mag = val
        sw = _sw_from_magnitude(mag)
        return sw, _uint_to_le_bytes(mag, SW_BYTES[sw])
    sw = _sw_from_signed_value(val)
    return sw, _signed_to_le_bytes(val, SW_BYTES[sw])


def _value_from_payload(otype: int, payload: bytes) -> int:
    if _is_unsigned(otype):
        mag = _le_bytes_to_uint(payload)
        if otype != OTYPE_UINT64 and mag > 0x7FFFFFFFFFFFFFFF:
            raise ValueError(f"value out of range for otype {otype}")
        return mag
    return _le_bytes_to_signed(payload)


def _sw_from_signed_value(val: int) -> int:
    if -128 <= val <= 127:
        return 0
    if -32768 <= val <= 32767:
        return 1
    if -2147483648 <= val <= 2147483647:
        return 2
    return 3


def _signed_to_le_bytes(val: int, size: int) -> bytes:
    u = val & ((1 << (size * 8)) - 1)
    return bytes((u >> (8 * i)) & 0xFF for i in range(size))


def _le_bytes_to_signed(payload: bytes) -> int:
    u = _le_bytes_to_uint(payload)
    bit_width = len(payload) * 8
    if u >= 1 << (bit_width - 1):
        u -= 1 << bit_width
    return u


def _le_bytes_to_uint(payload: bytes) -> int:
    u = 0
    for i, b in enumerate(payload):
        u |= b << (8 * i)
    return u


def _is_unsigned(otype: int) -> bool:
    return otype <= OTYPE_UINT64


def _width_bits(otype: int) -> int:
    if otype in (0, 4):
        return 0
    if otype in (1, 5):
        return 1
    if otype in (2, 6):
        return 2
    return 3


def _magnitude_from_typed(otype: int, val: int) -> tuple[int, bool]:
    if _is_unsigned(otype):
        return val, False
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


def _uint_to_le_bytes(v: int, size: int) -> bytes:
    return bytes((v >> (8 * i)) & 0xFF for i in range(size))


def _validate_range(otype: int, val: int) -> None:
    limits = {
        0: (0, 0xFF),
        1: (0, 0xFFFF),
        2: (0, 0xFFFFFFFF),
        3: (None, None),
        4: (-128, 127),
        5: (-32768, 32767),
        6: (-2147483648, 2147483647),
        7: (None, None),
    }
    lo, hi = limits.get(otype, (None, None))
    if lo is not None and val < lo:
        raise ValueError(f"value {val} out of range for otype {otype}")
    if hi is not None and val > hi:
        raise ValueError(f"value {val} out of range for otype {otype}")
