/**
 * 带类型整数的内部表示。
 * otype 索引：0=uint8, 1=uint16, 2=uint32, 3=uint64, 4=int8, 5=int16, 6=int32, 7=int64
 */

export const OType = Object.freeze({
  UINT8: 0, UINT16: 1, UINT32: 2, UINT64: 3,
  INT8: 4, INT16: 5, INT32: 6, INT64: 7,
});

/** @typedef {{ otype: number, val: number }} TypedValue */

/** @returns {TypedValue} */
export function u8(v) { return { otype: OType.UINT8, val: v }; }
export function u16(v) { return { otype: OType.UINT16, val: v }; }
export function u32(v) { return { otype: OType.UINT32, val: v }; }
export function u64(v) {
  if (v > Number.MAX_SAFE_INTEGER) throw new Error(`uint64 ${v} overflows safe integer`);
  return { otype: OType.UINT64, val: v };
}
export function i8(v) { return { otype: OType.INT8, val: v }; }
export function i16(v) { return { otype: OType.INT16, val: v }; }
export function i32(v) { return { otype: OType.INT32, val: v }; }
export function i64(v) { return { otype: OType.INT64, val: v }; }
