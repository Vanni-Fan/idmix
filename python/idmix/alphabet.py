"""XID 文本层：自定义进制编解码。"""

from __future__ import annotations

import struct


class RadixCodec:
    """将二进制块编码为自定义进制字符串，反之亦然。"""

    def __init__(self, alphabet: str) -> None:
        chars = list(alphabet)
        if len(chars) < 2:
            raise ValueError("alphabet must have at least 2 unique characters")
        self.base = len(chars)
        self.chars = chars
        self._from_custom: dict[str, int] = {}
        for i, ch in enumerate(chars):
            if ch in self._from_custom:
                raise ValueError(f"alphabet contains duplicate character {ch!r}")
            self._from_custom[ch] = i

    def encode_bytes(self, data: bytes) -> str:
        """二进制块 → 自定义进制字符串。"""
        if not data:
            return self.chars[0]
        wrapped = struct.pack(">H", len(data)) + data
        n = int.from_bytes(wrapped, "big")
        return self._int_to_string(n)

    def decode_bytes(self, s: str) -> bytes:
        """自定义进制字符串 → 二进制块。"""
        if not s:
            raise ValueError("empty string")
        n = self._string_to_int(s)
        raw = n.to_bytes((n.bit_length() + 7) // 8 or 1, "big")
        if raw == b"\x00" and n == 0:
            raw = b""
        for pad in (0, 1):
            buf = b"\x00" * pad + raw
            if len(buf) < 2:
                continue
            data_len = struct.unpack(">H", buf[:2])[0]
            if len(buf) != 2 + data_len:
                continue
            return buf[2:]
        raise ValueError("invalid encoded data length")

    def _int_to_string(self, n: int) -> str:
        if n == 0:
            return self.chars[0]
        chars: list[str] = []
        base = self.base
        while n > 0:
            n, rem = divmod(n, base)
            chars.append(self.chars[rem])
        return "".join(reversed(chars))

    def _string_to_int(self, s: str) -> int:
        n = 0
        for ch in s:
            if ch not in self._from_custom:
                raise ValueError(f"invalid character {ch!r}")
            n = n * self.base + self._from_custom[ch]
        return n
