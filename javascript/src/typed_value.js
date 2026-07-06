/**
 * 带类型整数的内部表示。
 * otype 索引：0=uint8, 1=uint16, 2=uint32, 3=uint64, 4=int8, 5=int16, 6=int32, 7=int64
 * val 可为 number 或 bigint（64 位极值需 bigint）。
 */

export const OType = Object.freeze({
  UINT8: 0, UINT16: 1, UINT32: 2, UINT64: 3,
  INT8: 4, INT16: 5, INT32: 6, INT64: 7,
});

/** @typedef {{ otype: number, val: number | bigint }} TypedValue */

const U64_MAX = 0xffffffffffffffffn;
const I64_MIN = -0x8000000000000000n;
const I64_MAX = 0x7fffffffffffffffn;

function toBigInt(v) {
  return typeof v === 'bigint' ? v : BigInt(v);
}

/** @returns {TypedValue} */
export function u8(v) { return { otype: OType.UINT8, val: v }; }
export function u16(v) { return { otype: OType.UINT16, val: v }; }
export function u32(v) { return { otype: OType.UINT32, val: v }; }
export function u64(v) {
  const n = toBigInt(v);
  if (n < 0n || n > U64_MAX) throw new Error(`uint64 ${v} out of range`);
  return { otype: OType.UINT64, val: n <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(n) : n };
}
export function i8(v) { return { otype: OType.INT8, val: v }; }
export function i16(v) { return { otype: OType.INT16, val: v }; }
export function i32(v) { return { otype: OType.INT32, val: v }; }
export function i64(v) {
  const n = toBigInt(v);
  if (n < I64_MIN || n > I64_MAX) throw new Error(`int64 ${v} out of range`);
  return { otype: OType.INT64, val: n >= BigInt(Number.MIN_SAFE_INTEGER) && n <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(n) : n };
}

/** @param {TypedValue} a @param {TypedValue} b */
export function typedValuesEqual(a, b) {
  if (a.otype !== b.otype) return false;
  return toBigInt(a.val) === toBigInt(b.val);
}
