"""带类型整数的内部表示与对外 Value 类型。"""

from __future__ import annotations

from dataclasses import dataclass
from enum import IntEnum
from typing import Union


class OType(IntEnum):
    """原始类型索引，写入扩展模式对象头低 4 位。"""

    UINT8 = 0
    UINT16 = 1
    UINT32 = 2
    UINT64 = 3
    INT8 = 4
    INT16 = 5
    INT32 = 6
    INT64 = 7


@dataclass(frozen=True)
class TypedValue:
    """内部统一整数表示：otype + 数值。"""

    otype: int
    val: int


# 对外暴露的带类型值（工厂函数）
def u8(v: int) -> TypedValue:
    return TypedValue(OType.UINT8, v)


def u16(v: int) -> TypedValue:
    return TypedValue(OType.UINT16, v)


def u32(v: int) -> TypedValue:
    return TypedValue(OType.UINT32, v)


def u64(v: int) -> TypedValue:
    if v < 0 or v > 0xFFFFFFFFFFFFFFFF:
        raise ValueError(f"uint64 value {v} out of range")
    return TypedValue(OType.UINT64, v)


def i8(v: int) -> TypedValue:
    return TypedValue(OType.INT8, v)


def i16(v: int) -> TypedValue:
    return TypedValue(OType.INT16, v)


def i32(v: int) -> TypedValue:
    return TypedValue(OType.INT32, v)


def i64(v: int) -> TypedValue:
    return TypedValue(OType.INT64, v)


Value = TypedValue
