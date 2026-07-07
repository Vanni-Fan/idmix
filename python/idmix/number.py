"""类型转换层：将调用方值规范化为内部 dataObject 表示。"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

MAX_STRING_LEN = 63

OTYPE_UINT8 = 0
OTYPE_UINT16 = 1
OTYPE_UINT32 = 2
OTYPE_UINT64 = 3
OTYPE_INT8 = 4
OTYPE_INT16 = 5
OTYPE_INT32 = 6
OTYPE_INT64 = 7


@dataclass
class DataObject:
    is_string: bool = False
    otype: int = 0
    val: int = 0
    str: bytes = b""


def normalize_objects(values: list[Any]) -> list[DataObject]:
    out: list[DataObject] = []
    for i, v in enumerate(values):
        try:
            out.append(object_from_any(v))
        except (TypeError, ValueError) as e:
            raise ValueError(f"value[{i}]: {e}") from e
    return out


@dataclass(frozen=True)
class TypedInt:
    """带 otype 的整数，用于与 Go 类型一致的确定性编码。"""

    otype: int
    val: int


def u8(v: int) -> TypedInt:
    return TypedInt(OTYPE_UINT8, v)


def u16(v: int) -> TypedInt:
    return TypedInt(OTYPE_UINT16, v)


def u32(v: int) -> TypedInt:
    return TypedInt(OTYPE_UINT32, v)


def u64(v: int) -> TypedInt:
    return TypedInt(OTYPE_UINT64, v)


def i8(v: int) -> TypedInt:
    return TypedInt(OTYPE_INT8, v)


def i16(v: int) -> TypedInt:
    return TypedInt(OTYPE_INT16, v)


def i32(v: int) -> TypedInt:
    return TypedInt(OTYPE_INT32, v)


def i64(v: int) -> TypedInt:
    return TypedInt(OTYPE_INT64, v)


def object_from_any(v: Any) -> DataObject:
    if isinstance(v, TypedInt):
        return DataObject(otype=v.otype, val=v.val)
    if isinstance(v, str):
        if len(v) == 0:
            raise ValueError(f"empty string is not allowed (max {MAX_STRING_LEN} bytes)")
        if len(v.encode("utf-8")) > MAX_STRING_LEN:
            raise ValueError(f"string length {len(v)} exceeds max {MAX_STRING_LEN}")
        return DataObject(is_string=True, str=v.encode("utf-8"))
    if isinstance(v, (bytes, bytearray)):
        if len(v) == 0:
            raise ValueError(f"empty byte slice is not allowed (max {MAX_STRING_LEN} bytes)")
        if len(v) > MAX_STRING_LEN:
            raise ValueError(f"byte slice length {len(v)} exceeds max {MAX_STRING_LEN}")
        return DataObject(is_string=True, str=bytes(v))
    if isinstance(v, bool):
        raise TypeError(
            f"unsupported type {type(v)!r} (integer or string up to {MAX_STRING_LEN} bytes)"
        )
    if isinstance(v, int):
        return DataObject(otype=OTYPE_INT64, val=v)
    raise TypeError(
        f"unsupported type {type(v)!r} (integer or string up to {MAX_STRING_LEN} bytes)"
    )


def materialize_objects(objects: list[DataObject]) -> list[Any]:
    out: list[Any] = []
    for i, obj in enumerate(objects):
        if obj.is_string:
            out.append(obj.str.decode("utf-8"))
            continue
        try:
            out.append(materialize_value(obj))
        except ValueError as e:
            raise ValueError(f"value[{i}]: {e}") from e
    return out


def materialize_value(obj: DataObject) -> int:
    return obj.val
