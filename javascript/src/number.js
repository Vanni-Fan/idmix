/**
 * 将调用方传入的值规范化为内部 dataObject 表示。
 */

export const OType = Object.freeze({
  UINT8: 0, UINT16: 1, UINT32: 2, UINT64: 3,
  INT8: 4, INT16: 5, INT32: 6, INT64: 7,
});

export const MAX_STRING_LEN = 63;

const U64_MAX = 0xffffffffffffffffn;
const I64_MIN = -0x8000000000000000n;
const I64_MAX = 0x7fffffffffffffffn;

function toBigInt(v) {
  return typeof v === 'bigint' ? v : BigInt(v);
}

/** @typedef {{ otype: number, val: number | bigint }} TypedValue */

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

/** @param {unknown} v */
export function isTypedValue(v) {
  return v != null && typeof v === 'object' && 'otype' in v && 'val' in v && !('isString' in v);
}

/**
 * @param {unknown} v
 * @returns {{ isString: true, str: Uint8Array } | { isString: false, otype: number, val: number | bigint }}
 */
export function objectFromAny(v) {
  if (typeof v === 'string') {
    const bytes = new TextEncoder().encode(v);
    if (bytes.length === 0) throw new Error(`empty string is not allowed (max ${MAX_STRING_LEN} bytes)`);
    if (bytes.length > MAX_STRING_LEN) throw new Error(`string length ${bytes.length} exceeds max ${MAX_STRING_LEN}`);
    return { isString: true, str: bytes };
  }
  if (v instanceof Uint8Array) {
    if (v.length === 0) throw new Error(`empty byte slice is not allowed (max ${MAX_STRING_LEN} bytes)`);
    if (v.length > MAX_STRING_LEN) throw new Error(`byte slice length ${v.length} exceeds max ${MAX_STRING_LEN}`);
    return { isString: true, str: new Uint8Array(v) };
  }
  if (isTypedValue(v)) {
    return { isString: false, otype: v.otype, val: v.val };
  }
  if (typeof v === 'number' || typeof v === 'bigint') {
    return { isString: false, otype: OType.INT64, val: typeof v === 'bigint' ? v : BigInt(v) };
  }
  throw new Error(`unsupported type ${typeof v} (integer, typed value, or string up to ${MAX_STRING_LEN} bytes)`);
}

/** @param {unknown[]} values */
export function normalizeObjects(values) {
  return values.map((v, i) => {
    try {
      return objectFromAny(v);
    } catch (err) {
      throw new Error(`value[${i}]: ${err.message}`);
    }
  });
}

/** @param {{ isString: boolean, otype?: number, val?: number | bigint, str?: Uint8Array }} obj */
export function materializeObject(obj) {
  if (obj.isString) {
    return new TextDecoder().decode(obj.str);
  }
  const val = obj.val;
  switch (obj.otype) {
    case OType.UINT8: return u8(Number(val));
    case OType.UINT16: return u16(Number(val));
    case OType.UINT32: return u32(Number(val));
    case OType.UINT64: return u64(val);
    case OType.INT8: return i8(Number(val));
    case OType.INT16: return i16(Number(val));
    case OType.INT32: return i32(Number(val));
    case OType.INT64: return i64(val);
    default: throw new Error(`invalid otype ${obj.otype}`);
  }
}

/** @param {{ isString: boolean, otype?: number, val?: number | bigint, str?: Uint8Array }[]} objects */
export function materializeObjects(objects) {
  return objects.map((obj, i) => {
    try {
      return materializeObject(obj);
    } catch (err) {
      throw new Error(`value[${i}]: ${err.message}`);
    }
  });
}

/** @param {number} otype @param {string|number} val @param {string} [str] */
export function materializeFromCrossLang(otype, val, str) {
  if (str != null && str !== '') return str;
  const s = String(val);
  const bi = BigInt(s);
  switch (otype) {
    case 0: return u8(Number(bi));
    case 1: return u16(Number(bi));
    case 2: return u32(Number(bi));
    case 3: return bi > BigInt(Number.MAX_SAFE_INTEGER) ? u64(bi) : u64(Number(bi));
    case 4: return i8(Number(bi));
    case 5: return i16(Number(bi));
    case 6: return i32(Number(bi));
    case 7: return (bi < BigInt(Number.MIN_SAFE_INTEGER) || bi > BigInt(Number.MAX_SAFE_INTEGER)) ? i64(bi) : i64(Number(bi));
    default: throw new Error(`invalid otype ${otype}`);
  }
}

/** @param {unknown} a @param {unknown} b */
export function valuesEqual(a, b) {
  if (typeof a === 'string' && typeof b === 'string') return a === b;
  if (isTypedValue(a) && isTypedValue(b)) return typedValuesEqual(a, b);
  return false;
}
