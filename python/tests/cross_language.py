"""跨语言测试向量加载。"""

from __future__ import annotations

import json
from pathlib import Path

from idmix import TypedValue, i32, i64, u32, u64, u8, u16, i8, i16

VECTORS_PATH = Path(__file__).resolve().parents[2] / "testdata" / "cross_language_vectors.json"

EXTREME_UINT32_MAX = 4294967295
EXTREME_INT32_MIN = -2147483648
EXTREME_INT64_MIN = -9223372036854775808
EXTREME_INT64_MAX = 9223372036854775807
EXTREME_UINT64_MAX = 18446744073709551615


def load_cross_language_vectors() -> dict:
    return json.loads(VECTORS_PATH.read_text(encoding="utf-8"))


def materialize_otype_val(otype: int, val: str | int) -> TypedValue:
    n = int(val)
    factories = {
        0: u8,
        1: u16,
        2: u32,
        3: u64,
        4: i8,
        5: i16,
        6: i32,
        7: i64,
    }
    return factories[otype](n)
