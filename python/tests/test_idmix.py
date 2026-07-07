#!/usr/bin/env python3
"""idmix 核心行为测试。运行: python -m unittest discover -s tests -v"""

from __future__ import annotations

import sys
import unittest

from idmix import (
    Base64Codec,
    FuncCodec,
    IdMix,
    Idx,
    RadixCodec,
    decode_string,
    encode_bytes,
    with_alphabet,
    with_check_bits,
    with_codec,
    with_idx,
    with_max_objects,
    u16,
    u32,
    u64,
    u8,
    i16,
    i32,
    i64,
    i8,
)
from idmix.idx_codec import _encode_object
from idmix.number import object_from_any

from cross_language import (
    EXTREME_INT32_MIN,
    EXTREME_INT64_MAX,
    EXTREME_INT64_MIN,
    EXTREME_UINT32_MAX,
    EXTREME_UINT64_MAX,
    load_cross_language_vectors,
    materialize_input_from_otype,
    materialize_otype_val,
)


def _hex(b: bytes) -> str:
    return " ".join(f"{x:02X}" for x in b) if b else "(empty)"


def log_round_trip(case: unittest.TestCase, m: IdMix, title: str, values: list) -> list:
    case.maxDiff = None
    print(f"\n{'─' * 40}")
    print(f">> {title}")
    encoded = m.encode_with_variant(0, *values)
    raw = m.codec.decode(encoded)
    print(f"  二进制: {_hex(raw)} ({len(raw)} bytes)")
    print(f"  字符串: {encoded!r} (len={len(encoded)})")
    decoded = m.decode(encoded)
    for i, (got, want) in enumerate(zip(decoded, values)):
        want_cmp = want.val if hasattr(want, "val") else want
        got_cmp = got
        mark = "OK" if got_cmp == want_cmp or str(got_cmp) == str(want_cmp) else "FAIL"
        print(f"  校验[{i}]: {mark}  want={want_cmp!r} => got={got!r}")
        if isinstance(want_cmp, str):
            case.assertEqual(got, want_cmp)
        elif hasattr(want, "val"):
            case.assertEqual(got, want.val)
        else:
            case.assertEqual(got, want)
    return decoded


class TestIdMix(unittest.TestCase):
    def test_spec_example_binary(self) -> None:
        m = IdMix.new()
        data = m._encode_binary([u16(5), i64(-1), u32(40)], 0)
        want = bytes([0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F])
        print(f"\n>> 规范二进制块 (variant=0): {_hex(data)}")
        self.assertEqual(data, want)

    def test_round_trip_basic(self) -> None:
        m = IdMix.new()
        log_round_trip(self, m, "规范示例", [u16(5), i64(-1), u32(40)])

    def test_round_trip_uint32_large(self) -> None:
        m = IdMix.new()
        v = u32(2_000_000_000)
        out = log_round_trip(self, m, "单值 u32(2000000000)", [v])
        self.assertEqual(out[0], 2_000_000_000)

    def test_custom_alphabet(self) -> None:
        m = IdMix.new(with_alphabet("abcd"))
        log_round_trip(self, m, "四进制 abcd", [u16(100), i32(-10), u8(3)])

    def test_round_trip_all_types(self) -> None:
        m = IdMix.new()
        inputs = [
            u8(0), u8(15), u16(128), u32(0x7FFFFFFF), u64(1 << 40),
            i8(-16), i8(-1), i8(127), i16(-128), i32(0), i64(-1), 42,
        ]
        s = m.encode_with_variant(0, *inputs)
        out = m.decode(s)
        want = [0, 15, 128, 0x7FFFFFFF, 1 << 40, -16, -1, 127, -128, 0, -1, 42]
        for i, w in enumerate(want):
            self.assertEqual(out[i], w, f"[{i}]")

    def test_string_round_trip(self) -> None:
        m = IdMix.new()
        out = log_round_trip(self, m, "字符串 + 整数", ["hello", u16(5), "世界"])
        self.assertEqual(out[0], "hello")
        self.assertEqual(out[1], 5)
        self.assertEqual(out[2], "世界")

    def test_embedded_modes(self) -> None:
        cases = [
            (u8(10), 1),
            (i16(-5), 1),
            (u32(16), 2),
            (i32(0), 1),
        ]
        for val, want_len in cases:
            with self.subTest(val=val):
                obj = object_from_any(val)
                encoded = _encode_object(obj)
                self.assertEqual(len(encoded), want_len)

    def test_single_object_one_byte_header(self) -> None:
        idx = Idx.new()
        cases = [
            ("embedded_uint8", u8(10), 1),
            ("embedded_int16_neg", i16(-5), 1),
            ("extended_uint32", u32(1000), 3),
            ("extended_string", "hi", 3),
        ]
        for name, val, obj_len in cases:
            with self.subTest(name=name):
                data = idx.encode_with_variant(0, val)
                self.assertEqual(len(data), 1 + obj_len)
                self.assertEqual(data[0] & 0x80, 0)
                out = idx.decode(data)
                want = val.val if hasattr(val, "val") else val
                self.assertEqual(out[0], want)

        multi = idx.encode_with_variant(0, u8(1), u8(2))
        self.assertNotEqual(multi[0] & 0x80, 0)
        self.assertEqual(multi[1], 2)

    def test_radix_round_trip(self) -> None:
        raw = bytes([0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F])
        for alphabet, want_error in [
            ("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", False),
            ("abcd", False),
            ("0123456789abc", False),
            ("abca", True),
            ("a", True),
        ]:
            with self.subTest(alphabet=alphabet):
                if want_error:
                    with self.assertRaises(ValueError):
                        RadixCodec.new(alphabet)
                    continue
                rc = RadixCodec.new(alphabet)
                enc = encode_bytes(raw, rc)
                dec = decode_string(enc, rc)
                self.assertEqual(dec, raw)

    def test_custom_codec(self) -> None:
        raw = bytes([0x80, 0x03, 0x22, 0x47, 0xB5, 0x1F])
        b64 = Base64Codec.new()
        s = encode_bytes(raw, b64)
        self.assertEqual(decode_string(s, b64), raw)
        m = IdMix.new(with_codec(b64))
        enc = m.encode_with_variant(0, u16(5), i64(-1), u32(40))
        out = m.decode(enc)
        self.assertEqual(out[0], 5)

        inner = RadixCodec.new(
            "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
        )
        key = 0x5A

        def xor_encode(data: bytes) -> str:
            return inner.encode(bytes(b ^ key for b in data))

        def xor_decode(s: str) -> bytes:
            buf = inner.decode(s)
            return bytes(b ^ key for b in buf)

        xor = FuncCodec(encode_fn=xor_encode, decode_fn=xor_decode)
        self.assertEqual(decode_string(encode_bytes(raw, xor), xor), raw)

    def test_encode_bytes_standalone(self) -> None:
        raw = bytes([0xDE, 0xAD, 0xBE, 0xEF])
        s = encode_bytes(raw)
        self.assertEqual(decode_string(s), raw)
        rc = RadixCodec.new("abcd")
        s2 = encode_bytes(raw, rc)
        self.assertEqual(decode_string(s2, rc), raw)

    def test_checksum_rejects(self) -> None:
        m = IdMix.new()
        data = bytearray(m._encode_binary([u32(1)], 0))
        data[1 if len(data) > 1 else 0] ^= 0x01
        tampered = encode_bytes(bytes(data))
        with self.assertRaises(ValueError):
            m.decode(tampered)

    def test_new_validation(self) -> None:
        with self.assertRaises(ValueError):
            Idx.new(with_max_objects(256))
        with self.assertRaises(ValueError):
            IdMix.new(with_alphabet("abca"))
        idx = Idx.new(with_max_objects(100))
        m = IdMix.new(with_idx(idx), with_alphabet("abcd"))
        self.assertEqual(m.idx.max_objects, 100)

    def test_encode_errors(self) -> None:
        m = IdMix.new()
        with self.assertRaises(ValueError):
            m.encode()
        with self.assertRaises((TypeError, ValueError)):
            m.encode(3.14)

    def test_decode_invalid_char(self) -> None:
        m = IdMix.new(with_alphabet("abcd"))
        with self.assertRaises(ValueError):
            m.decode("axy")

    def test_multiple_encodings_differ(self) -> None:
        m = IdMix.new()
        seen: set[str] = set()
        for _ in range(50):
            seen.add(m.encode(u32(42)))
        self.assertGreaterEqual(len(seen), 2)

    def test_reject_rate_approx(self) -> None:
        m = IdMix.new(with_alphabet("abcd"))
        chars = "abcd"
        passed = 0
        n = 5000
        for i in range(n):
            s = "".join(chars[(i * 3 + j * 7) % 4] for j in range(8))
            try:
                m.decode(s)
                passed += 1
            except ValueError:
                pass
        rate = passed * 100 // n
        self.assertLessEqual(rate, 30, f"pass rate {rate}% too high")

    def test_extreme_values_round_trip(self) -> None:
        m = IdMix.new()
        cases = [
            ("uint32_max", [u32(EXTREME_UINT32_MAX)]),
            ("int32_min", [i32(EXTREME_INT32_MIN)]),
            ("int64_min", [i64(EXTREME_INT64_MIN)]),
            ("int64_max", [i64(EXTREME_INT64_MAX)]),
            ("uint64_max", [u64(EXTREME_UINT64_MAX)]),
        ]
        for name, values in cases:
            with self.subTest(name=name):
                log_round_trip(self, m, name, values)

    def test_cross_language_vectors(self) -> None:
        data = load_cross_language_vectors()
        m = IdMix.new(with_alphabet(data["alphabet"]))
        for case in data["cases"]:
            with self.subTest(case["name"]):
                decoded = m.decode(case["encoded"])
                self.assertEqual(len(decoded), len(case["values"]))
                for i, entry in enumerate(case["values"]):
                    expected = materialize_otype_val(entry)
                    self.assertEqual(decoded[i], expected, f"[{i}] {case['name']}")

    def test_cross_language_encode_deterministic(self) -> None:
        data = load_cross_language_vectors()
        m = IdMix.new(with_alphabet(data["alphabet"]))
        for case in data["cases"]:
            with self.subTest(case["name"]):
                inputs = [materialize_input_from_otype(v) for v in case["values"]]
                enc = m.encode_with_variant(case["variant"], *inputs)
                self.assertEqual(enc, case["encoded"])


if __name__ == "__main__":
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromModule(sys.modules[__name__])
    runner = unittest.TextTestRunner(verbosity=2)
    raise SystemExit(not runner.run(suite).wasSuccessful())
