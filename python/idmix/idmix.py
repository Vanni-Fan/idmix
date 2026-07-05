"""XID v1.1 编解码器主入口。"""

from __future__ import annotations

import random

from .alphabet import RadixCodec
from .typed_value import TypedValue
from . import xid_codec

DEFAULT_ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"


class IdMix:
    """XID v1.1 编解码器。"""

    def __init__(
        self,
        alphabet: str = DEFAULT_ALPHABET,
        max_objects: int = 511,
        max_variants: int = 32,
        check_bits: int = 2,
    ) -> None:
        self._radix = RadixCodec(alphabet)
        self.max_objects = max_objects
        self.max_variants = max_variants
        self.check_bits = check_bits
        self.count_bits = 0
        self.variant_bits = 0
        self.check_mask = 0
        self.count_mask = 0
        self.variant_mask = 0
        self.count_shift = 0
        self.variant_shift = 0
        self._finalize_layout()

    @classmethod
    def new(cls, alphabet: str | None = None) -> IdMix:
        return cls(alphabet or DEFAULT_ALPHABET)

    def encode(self, *values: TypedValue) -> str:
        """将多个带类型整数编码为 XID 字符串。"""
        if not values:
            raise ValueError("at least one value is required")
        if len(values) > self.max_objects:
            raise ValueError(f"too many objects: {len(values)} (max {self.max_objects})")
        variant_id = random.randint(0, self.max_variants - 1)
        data = xid_codec.encode_binary(self, list(values), variant_id)
        return self._radix.encode_bytes(data)

    def decode(self, s: str) -> list[TypedValue]:
        """将 XID 字符串解码为带类型整数列表。"""
        data = self._radix.decode_bytes(s)
        return xid_codec.decode_binary(self, data)

    def _finalize_layout(self) -> None:
        variant_bits = 1 if self.max_variants <= 1 else _bit_len(self.max_variants - 1)
        count_bits = 1 if self.max_objects <= 1 else _bit_len(self.max_objects)
        total = self.check_bits + count_bits + variant_bits
        if total > 16:
            raise ValueError(
                f"checkBits({self.check_bits}) + countBits({count_bits}) + "
                f"variantBits({variant_bits}) = {total} exceeds 16-bit header"
            )
        self.count_bits = count_bits
        self.variant_bits = variant_bits
        self.check_mask = (1 << self.check_bits) - 1
        self.count_mask = ((1 << count_bits) - 1) << self.check_bits
        self.variant_mask = ((1 << variant_bits) - 1) << (self.check_bits + count_bits)
        self.count_shift = self.check_bits
        self.variant_shift = self.check_bits + count_bits


def _bit_len(n: int) -> int:
    if n <= 0:
        return 1
    return n.bit_length()
