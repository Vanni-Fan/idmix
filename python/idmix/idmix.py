"""IdMix 高级封装：IDX 二进制编解码 + 可插拔文本 Codec。"""

from __future__ import annotations

import random
from typing import Any, Callable

from .alphabet import RadixCodec
from .codec import Codec, DEFAULT_ALPHABET, default_codec_instance, encode_bytes, decode_string
from .idx_codec import Idx, with_check_bits, with_max_objects, with_max_variants
from .number import normalize_objects


class IdMix:
    """组合 IDX 二进制编解码与文本 Codec。"""

    def __init__(self, idx: Idx | None = None, codec: Codec | None = None) -> None:
        self._idx = idx or Idx.new()
        self._codec = codec or default_codec_instance()

    @classmethod
    def new(cls, *opts: Callable[[IdMix], None]) -> IdMix:
        m = cls()
        for opt in opts:
            opt(m)
        return m

    @property
    def idx(self) -> Idx:
        return self._idx

    @property
    def codec(self) -> Codec:
        return self._codec

    def encode(self, *values: Any) -> str:
        if len(values) < 1:
            raise ValueError("at least one value is required")
        variant_id = random.randint(0, self._idx.max_variants - 1)
        data = self._encode_binary(list(values), variant_id)
        return self._codec.encode(data)

    def decode(self, s: str) -> list[Any]:
        data = self._codec.decode(s)
        return self._idx.decode(data)

    def encode_with_variant(self, variant_id: int, *values: Any) -> str:
        data = self._encode_binary(list(values), variant_id)
        return self._codec.encode(data)

    def _encode_binary(self, values: list[Any], variant_id: int) -> bytes:
        objects = normalize_objects(values)
        return self._idx._encode_binary(objects, variant_id)


def with_codec(codec: Codec) -> Callable[[IdMix], None]:
    def apply(m: IdMix) -> None:
        if codec is None:
            raise ValueError("codec cannot be nil")
        m._codec = codec

    return apply


def with_alphabet(alphabet: str) -> Callable[[IdMix], None]:
    def apply(m: IdMix) -> None:
        m._codec = RadixCodec.new(alphabet)

    return apply


def with_idx(idx: Idx) -> Callable[[IdMix], None]:
    def apply(m: IdMix) -> None:
        if idx is None:
            raise ValueError("idx cannot be nil")
        m._idx = idx

    return apply
