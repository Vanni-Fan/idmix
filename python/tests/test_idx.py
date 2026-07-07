#!/usr/bin/env python3
"""Idx 配置项与字符串长度边界测试。"""

from __future__ import annotations

import unittest

from idmix import IdMix, Idx, with_check_bits, with_idx, with_max_objects, with_max_variants
from idmix.number import MAX_STRING_LEN


class TestStringLengthBoundaries(unittest.TestCase):
    def setUp(self) -> None:
        self.idx = Idx.new()

    def test_empty_string_rejected(self) -> None:
        with self.assertRaises(ValueError):
            self.idx.encode("")

    def test_empty_bytes_rejected(self) -> None:
        with self.assertRaises(ValueError):
            self.idx.encode(b"")

    def test_len_1_ok(self) -> None:
        data = self.idx.encode_with_variant(0, "x")
        out = self.idx.decode(data)
        self.assertEqual(out[0], "x")

    def test_len_63_ok(self) -> None:
        ok63 = "a" * MAX_STRING_LEN
        data = self.idx.encode_with_variant(0, ok63)
        self.assertEqual(len(data), 1 + 1 + MAX_STRING_LEN)
        out = self.idx.decode(data)
        self.assertEqual(out[0], ok63)

    def test_len_64_rejected(self) -> None:
        too_long = "b" * (MAX_STRING_LEN + 1)
        with self.assertRaises(ValueError):
            self.idx.encode(too_long)

    def test_len_64_bytes_rejected(self) -> None:
        too_long = b"b" * (MAX_STRING_LEN + 1)
        with self.assertRaises(ValueError):
            self.idx.encode(too_long)

    def test_idmix_end_to_end_63(self) -> None:
        ok63 = "a" * MAX_STRING_LEN
        m = IdMix.new()
        s = m.encode_with_variant(0, ok63)
        out = m.decode(s)
        self.assertEqual(out[0], ok63)

    def test_idmix_len_64_rejected(self) -> None:
        too_long = "b" * (MAX_STRING_LEN + 1)
        m = IdMix.new()
        with self.assertRaises(ValueError):
            m.encode(too_long)


class TestIdxMaxObjects(unittest.TestCase):
    def test_new_invalid(self) -> None:
        for n in (0, 256):
            with self.subTest(n=n):
                with self.assertRaises(ValueError):
                    Idx.new(with_max_objects(n))

    def test_encode_over_limit(self) -> None:
        idx = Idx.new(with_max_objects(2))
        with self.assertRaises(ValueError):
            idx.encode(1, 2, 3)

    def test_encode_at_limit(self) -> None:
        idx = Idx.new(with_max_objects(2))
        data = idx.encode_with_variant(0, 1, 2)
        self.assertNotEqual(data[0] & 0x80, 0)
        self.assertEqual(data[1], 2)
        out = idx.decode(data)
        self.assertEqual(len(out), 2)

    def test_max_255_new_ok(self) -> None:
        idx = Idx.new(with_max_objects(255))
        self.assertEqual(idx.max_objects, 255)


class TestIdxMaxVariants(unittest.TestCase):
    def test_new_invalid(self) -> None:
        for n in (0, 33):
            with self.subTest(n=n):
                with self.assertRaises(ValueError):
                    Idx.new(with_max_variants(n))

    def test_encode_variant_out_of_range(self) -> None:
        idx = Idx.new(with_max_variants(4))
        with self.assertRaises(ValueError):
            idx.encode_with_variant(4, 1)

    def test_encode_variant_at_limit(self) -> None:
        idx = Idx.new(with_max_variants(4))
        data = idx.encode_with_variant(3, 42)
        out = idx.decode(data)
        self.assertEqual(out[0], 42)

    def test_decode_rejects_high_variant(self) -> None:
        large = Idx.new(with_max_variants(32))
        data = large.encode_with_variant(20, 7)
        small = Idx.new(with_max_variants(16))
        with self.assertRaises(ValueError):
            small.decode(data)

    def test_idmix_respects_max_variants(self) -> None:
        idx = Idx.new(with_max_variants(8))
        m = IdMix.new(with_idx(idx))
        with self.assertRaises(ValueError):
            m.encode_with_variant(8, 1)


class TestIdxCheckBits(unittest.TestCase):
    def test_new_invalid(self) -> None:
        for n in (0, 3):
            with self.subTest(n=n):
                with self.assertRaises(ValueError):
                    Idx.new(with_check_bits(n))

    def test_check_bits_1_roundtrip(self) -> None:
        idx = Idx.new(with_check_bits(1))
        self.assertEqual(idx.check_bits, 1)
        self.assertEqual(idx.check_mask, 0x01)
        data = idx.encode_with_variant(0, 5, -1)
        out = idx.decode(data)
        self.assertEqual(out[0], 5)

    def test_check_bits_1_rejects_tamper(self) -> None:
        idx = Idx.new(with_check_bits(1))
        data = bytearray(idx.encode_with_variant(0, 1))
        data[-1] ^= 0x01
        with self.assertRaises(ValueError):
            idx.decode(bytes(data))

    def test_check_bits_2_default(self) -> None:
        idx = Idx.new()
        self.assertEqual(idx.check_bits, 2)
        self.assertEqual(idx.check_mask, 0x03)

    def test_idmix_with_check_bits_1(self) -> None:
        idx = Idx.new(with_check_bits(1))
        m = IdMix.new(with_idx(idx))
        s = m.encode_with_variant(0, 99)
        out = m.decode(s)
        self.assertEqual(out[0], 99)


if __name__ == "__main__":
    unittest.main()
