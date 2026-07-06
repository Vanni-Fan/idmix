/**
 * XID v1.1 二进制层编解码。
 */

import { OType } from './typed_value.js';

const MASK8 = 0xff;
const MASK16 = 0xffff;
const MASK32 = 0xffffffff;

function maskForBits(tbits) {
  if (tbits === 64) return -1; // 用 BigInt 路径处理
  if (tbits === 32) return MASK32;
  if (tbits === 16) return MASK16;
  if (tbits === 8) return MASK8;
  return (1 << tbits) - 1;
}

function pow2(n) {
  return n >= 32 ? 2 ** n : (1 << n);
}

const SW_BYTES = [1, 2, 4, 8];
const EMBEDDED_OTYPE = [
  [OType.UINT8, OType.UINT16, OType.UINT32, OType.UINT64],
  [OType.INT8, OType.INT16, OType.INT32, OType.INT64],
];

/** @param {import('./idmix.js').IdMix} m @param {import('./typed_value.js').TypedValue[]} typed @param {number} variantId */
export function encodeBinary(m, typed, variantId) {
  const parts = [];
  for (const tv of typed) parts.push(...encodeObject(tv));
  let objects = new Uint8Array(parts);
  const mask = (variantId * 0x9d + 0x37) & 0xff;
  objects = new Uint8Array(objects.map(b => b ^ mask));

  const count = typed.length;
  let header = (variantId << m.variantShift) | (count << m.countShift);
  const data = new Uint8Array(2 + objects.length);
  data[0] = header & 0xff;
  data[1] = (header >> 8) & 0xff;
  data.set(objects, 2);

  let xorSum = 0;
  for (const b of data) xorSum ^= b;
  header |= xorSum & m.checkMask;
  data[0] = header & 0xff;
  data[1] = (header >> 8) & 0xff;
  return data;
}

/** @param {import('./idmix.js').IdMix} m @param {Uint8Array} data */
export function decodeBinary(m, data) {
  if (data.length < 2) throw new Error('invalid data: too short');
  const header = data[0] | (data[1] << 8);
  const check = header & m.checkMask;
  const count = (header & m.countMask) >> m.countShift;
  const variantId = (header & m.variantMask) >> m.variantShift;

  if (variantId >= m.maxVariants) throw new Error(`invalid variant_id ${variantId}`);
  if (count > m.maxObjects) throw new Error(`invalid count ${count}`);

  const verify = new Uint8Array(data);
  verify[0] &= ~m.checkMask;
  let xorSum = 0;
  for (const b of verify) xorSum ^= b;
  if ((xorSum & m.checkMask) !== check) throw new Error('checksum mismatch');

  let objects = new Uint8Array(data.slice(2));
  const mask = (variantId * 0x9d + 0x37) & 0xff;
  objects = new Uint8Array(objects.map(b => b ^ mask));

  const result = [];
  let pos = 0;
  for (let i = 0; i < count; i++) {
    if (pos >= objects.length) throw new Error('premature end of data');
    const [tv, n] = decodeObject(objects.slice(pos));
    result.push(tv);
    pos += n;
  }
  if (pos !== objects.length) throw new Error('extra bytes after data objects');
  return result;
}

/** @param {import('./typed_value.js').TypedValue} tv */
function encodeObject(tv) {
  validateRange(tv.otype, tv.val);
  const embedded = tryEmbeddedHead(tv.otype, tv.val);
  if (embedded !== null) return [embedded & 0xff];
  const [mag, neg] = magnitudeFromTyped(tv.otype, tv.val);
  const sw = swFromMagnitude(mag);
  const payload = uintToLEBytes(mag, SW_BYTES[sw]);
  let head = 0x80 | (sw << 4) | tv.otype;
  if (neg) head |= 1 << 6;
  return [head, ...payload];
}

/** @param {Uint8Array} data */
function decodeObject(data) {
  if (!data.length) throw new Error('truncated object header');
  const head = data[0];
  if ((head & 0x80) === 0) {
    const sign = (head >> 6) & 1;
    const wb = (head >> 4) & 3;
    const v = head & 0x0f;
    const otype = EMBEDDED_OTYPE[sign][wb];
    const val = sign === 0 ? v : -v - 1;
    return [{ otype, val }, 1];
  }
  const sw = (head >> 4) & 3;
  const otype = head & 0x0f;
  if (otype > OType.INT64) throw new Error(`invalid otype ${otype}`);
  const numBytes = SW_BYTES[sw];
  if (data.length < 1 + numBytes) throw new Error('truncated object payload');
  let mag = 0n;
  for (let i = 0; i < numBytes; i++) mag |= BigInt(data[1 + i]) << BigInt(8 * i);
  const neg = ((head >> 6) & 1) !== 0;
  const val = valueFromMagnitude(mag, neg);
  validateRange(otype, val);
  return [{ otype, val }, 1 + numBytes];
}

function isUnsigned(otype) { return otype <= OType.UINT64; }
function isSigned(otype) { return otype >= OType.INT8; }

function widthBits(otype) {
  const m = { [OType.UINT8]: 0, [OType.INT8]: 0, [OType.UINT16]: 1, [OType.INT16]: 1,
              [OType.UINT32]: 2, [OType.INT32]: 2 };
  return m[otype] ?? 3;
}

function magnitudeFromTyped(otype, val) {
  const bval = typeof val === 'bigint' ? val : BigInt(val);
  if (isUnsigned(otype)) return [BigInt.asUintN(64, bval), false];
  if (bval < 0n) return [-bval, true];
  return [bval, false];
}

function swFromMagnitude(mag) {
  const m = typeof mag === 'bigint' ? mag : BigInt(mag);
  if (m < 256n) return 0;
  if (m < 65536n) return 1;
  if (m < 4294967296n) return 2;
  return 3;
}

function tryEmbeddedHead(otype, val) {
  const [mag, neg] = magnitudeFromTyped(otype, val);
  if (mag >= 17n) return null;
  const wb = widthBits(otype);
  if (mag === 16n) {
    if (neg) return (1 << 6) | (wb << 4) | 15;
    return null;
  }
  if (neg) return (1 << 6) | (wb << 4) | Number(mag - 1n);
  return (wb << 4) | Number(mag);
}

function valueFromMagnitude(mag, neg) {
  const m = typeof mag === 'bigint' ? mag : BigInt(mag);
  if (!neg) return m <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(m) : m;
  if (m === 1n << 63n) return -(1n << 63n);
  const v = -m;
  return v >= BigInt(Number.MIN_SAFE_INTEGER) && v <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(v) : v;
}

function uintToLEBytes(v, size) {
  const bval = typeof v === 'bigint' ? v : BigInt(v);
  const buf = [];
  for (let i = 0; i < size; i++) buf.push(Number((bval >> BigInt(8 * i)) & 0xffn));
  return buf;
}

function validateRange(otype, val) {
  const bval = typeof val === 'bigint' ? val : BigInt(val);
  const ranges = {
    [OType.UINT8]: [0n, 0xffn], [OType.UINT16]: [0n, 0xffffn], [OType.UINT32]: [0n, 0xffffffffn],
    [OType.UINT64]: [0n, 0xffffffffffffffffn], [OType.INT8]: [-128n, 127n], [OType.INT16]: [-32768n, 32767n],
    [OType.INT32]: [-2147483648n, 2147483647n], [OType.INT64]: [-0x8000000000000000n, 0x7fffffffffffffffn],
  };
  const [lo, hi] = ranges[otype] ?? [0n, 0n];
  if (bval < lo || bval > hi) throw new Error(`value ${val} out of range for otype ${otype}`);
}
