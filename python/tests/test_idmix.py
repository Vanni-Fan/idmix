#!/usr/bin/env python3
"""idmix 测试套件。运行: python -m unittest discover -s tests -v"""

from __future__ import annotations

import sys
import unittest

from idmix import IdMix, OType, TypedValue, i64, u16, u32, u64, u8, i32, i16, i8
from idmix import xid_codec


def _hex(b: bytes) -> str:
    return " ".join(f"{x:02X}" for x in b) if b else "(empty)"


def log_round_trip(case: unittest.TestCase, m: IdMix, title: str, values: list[TypedValue]) -> list[TypedValue]:
    """编码→解码往返，并输出详细日志（类似 go test -v）。"""
    case.maxDiff = None
    print(f"\n{'─' * 40}")
    print(f">> {title}")
    print(f"  字符表: {m._radix.chars!r} (进制={m._radix.base})")
    for i, v in enumerate(values):
        print(f"  编码输入[{i}] otype={v.otype} val={v.val}")

    encoded = m.encode(*values)
    raw = m._radix.decode_bytes(encoded)
    print(f"  二进制: {_hex(raw)} ({len(raw)} bytes)")
    print(f"  字符串: {encoded!r} (len={len(encoded)})")

    decoded = m.decode(encoded)
    for i, v in enumerate(decoded):
        print(f"  解码输出[{i}] otype={v.otype} val={v.val}")

    for i, (got, want) in enumerate(zip(decoded, values)):
        mark = "OK" if got == want else "FAIL"
        print(f"  校验[{i}]: {mark}  want({want.otype},{want.val}) => got({got.otype},{got.val})")
        case.assertEqual(got, want)
    return decoded


class TestIdMix(unittest.TestCase):
    """XID v1.1 核心行为测试。"""

    def test_spec_example_binary(self) -> None:
        """规范二进制块与 arithmetic.md 第 7 节一致 (variant=0)。"""
        m = IdMix.new()
        typed = [u16(5), i64(-1), u32(40)]
        data = xid_codec.encode_binary(m, typed, 0)
        want = bytes([0x0F, 0x00, 0x22, 0x47, 0xB5, 0x1F])
        print(f"\n>> 规范二进制块 (variant=0): {_hex(data)}")
        self.assertEqual(data, want)
        print("  与 arithmetic.md 第7节示例一致 OK")

    def test_round_trip_basic(self) -> None:
        m = IdMix.new()
        log_round_trip(self, m, "规范示例: u16(5), i64(-1), u32(40)", [u16(5), i64(-1), u32(40)])

    def test_round_trip_uint32_large(self) -> None:
        m = IdMix.new()
        v = u32(2_000_000_000)
        out = log_round_trip(self, m, "单值 u32(2000000000)", [v])
        self.assertEqual(out[0].val, 2_000_000_000)

    def test_custom_alphabet(self) -> None:
        m = IdMix.new("abcd")
        log_round_trip(self, m, "四进制 abcd", [u16(100), i32(-10), u8(3)])

    def test_checksum_rejects(self) -> None:
        m = IdMix.new()
        data = bytearray(xid_codec.encode_binary(m, [u32(1)], 0))
        print(f"\n>> 校验和拒绝测试")
        print(f"  原始: {_hex(bytes(data))}")
        data[2] ^= 0x01
        print(f"  篡改: {_hex(bytes(data))}")
        tampered = m._radix.encode_bytes(bytes(data))
        with self.assertRaises(ValueError):
            m.decode(tampered)
        print("  解码拒绝 OK")

    def test_multiple_encodings_differ(self) -> None:
        m = IdMix.new()
        seen: set[str] = set()
        for _ in range(50):
            seen.add(m.encode(u32(42)))
        print(f"\n>> 变体多态: u32(42) 编码 50 次 => {len(seen)} 种不同字符串")
        self.assertGreaterEqual(len(seen), 2)


if __name__ == "__main__":
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromModule(sys.modules[__name__])
    runner = unittest.TextTestRunner(verbosity=2)
    raise SystemExit(not runner.run(suite).wasSuccessful())
