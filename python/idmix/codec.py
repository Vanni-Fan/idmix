"""idmix 文本层可插拔编解码接口及内置实现。"""

from __future__ import annotations

import base64
from typing import Callable, Protocol, runtime_checkable

from .alphabet import RadixCodec

DEFAULT_ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

_ERR_NIL_CODEC_FUNC = ValueError("codec function is nil")

_default_codec: RadixCodec | None = None


@runtime_checkable
class Codec(Protocol):
    def encode(self, data: bytes) -> str: ...
    def decode(self, s: str) -> bytes: ...


class FuncCodec:
    """由函数实现的 Codec，便于包装 AES/XOR 等自定义逻辑。"""

    def __init__(
        self,
        encode_fn: Callable[[bytes], str] | None = None,
        decode_fn: Callable[[str], bytes] | None = None,
    ) -> None:
        self.encode_fn = encode_fn
        self.decode_fn = decode_fn

    def encode(self, data: bytes) -> str:
        if self.encode_fn is None:
            raise _ERR_NIL_CODEC_FUNC
        return self.encode_fn(data)

    def decode(self, s: str) -> bytes:
        if self.decode_fn is None:
            raise _ERR_NIL_CODEC_FUNC
        return self.decode_fn(s)


class Base64Codec:
    """使用标准 Base64 的二进制↔文本编解码器。"""

    @classmethod
    def new(cls) -> Base64Codec:
        return cls()

    def encode(self, data: bytes) -> str:
        return base64.standard_b64encode(data).decode("ascii")

    def decode(self, s: str) -> bytes:
        return base64.standard_b64decode(s)


def default_codec_instance() -> RadixCodec:
    global _default_codec
    if _default_codec is None:
        _default_codec = RadixCodec.new(DEFAULT_ALPHABET)
    return _default_codec


def resolve_codec(codec: Codec | None = None) -> Codec:
    if codec is not None:
        return codec
    return default_codec_instance()


def encode_bytes(data: bytes, codec: Codec | None = None) -> str:
    """将任意二进制编码为文本；codec 不传时使用默认 RadixCodec。"""
    return resolve_codec(codec).encode(data)


def decode_string(s: str, codec: Codec | None = None) -> bytes:
    """将文本还原为二进制；codec 不传时使用默认 RadixCodec。"""
    return resolve_codec(codec).decode(s)
