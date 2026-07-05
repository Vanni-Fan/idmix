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
  if (isUnsigned(tv.otype) && tv.val >= 0 && tv.val <= 15) {
    return [((widthBits(tv.otype) << 4) | tv.val) & 0xff];
  }
  if (isSigned(tv.otype) && tv.val >= -16 && tv.val <= -1) {
    const v = -tv.val - 1;
    return [((1 << 6) | (widthBits(tv.otype) << 4) | v) & 0xff];
  }
  const [sw, payload] = minimalComplementBytes(tv.otype, tv.val);
  return [0x80 | (sw << 4) | tv.otype, ...payload];
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
  if ((head >> 6) & 1) throw new Error('reserved bit set in extended mode');
  const sw = (head >> 4) & 3;
  const otype = head & 0x0f;
  if (otype > OType.INT64) throw new Error(`invalid otype ${otype}`);
  const numBytes = SW_BYTES[sw];
  if (data.length < 1 + numBytes) throw new Error('truncated object payload');
  let raw = 0;
  for (let i = 0; i < numBytes; i++) raw |= data[1 + i] << (8 * i);
  const val = reconstructInt(otype, sw, raw >>> 0);
  return [{ otype, val }, 1 + numBytes];
}

function isUnsigned(otype) { return otype <= OType.UINT64; }
function isSigned(otype) { return otype >= OType.INT8; }

function widthBits(otype) {
  const m = { [OType.UINT8]: 0, [OType.INT8]: 0, [OType.UINT16]: 1, [OType.INT16]: 1,
              [OType.UINT32]: 2, [OType.INT32]: 2 };
  return m[otype] ?? 3;
}

function targetBits(otype) {
  const m = { [OType.UINT8]: 8, [OType.INT8]: 8, [OType.UINT16]: 16, [OType.INT16]: 16,
              [OType.UINT32]: 32, [OType.INT32]: 32 };
  return m[otype] ?? 64;
}

function minimalComplementBytes(otype, val) {
  if (val === 0) return [0, [0]];
  if (isUnsigned(otype)) {
    if (val < 0) throw new Error(`negative value ${val} for unsigned type`);
    for (let sw = 0; sw < 4; sw++) {
      const size = SW_BYTES[sw];
      if (size < 8 && val >= pow2(size * 8)) continue;
      const buf = uintToLEBytes(val >>> 0, size);
      if ((buf[size - 1] & 0x80) === 0) return [sw, buf];
    }
    throw new Error('value too large for unsigned type');
  }
  const tbits = targetBits(otype);
  const mask = tbits === 64 ? 0xffffffffffffffffn : BigInt(maskForBits(tbits));
  let uval = BigInt(val) & mask;
  if (val < 0) {
    for (let sw = 0; sw < 4; sw++) {
      const size = SW_BYTES[sw];
      const shift = size * 8;
      if (shift >= tbits) return [sw, uintToLEBytes(Number(uval), size)];
      const lower = Number(uval & ((1n << BigInt(shift)) - 1n));
      const upper = Number(uval >> BigInt(shift));
      const upperMask = maskForBits(tbits - shift);
      if (upper !== upperMask) continue;
      const highByte = (lower >> (shift - 8)) & 0xff;
      if ((highByte & 0x80) === 0) continue;
      return [sw, uintToLEBytes(lower, size)];
    }
  } else {
    const u = Number(uval);
    for (let sw = 0; sw < 4; sw++) {
      const size = SW_BYTES[sw];
      if (size < 8 && u >= pow2(size * 8)) continue;
      const buf = uintToLEBytes(u, size);
      if ((buf[size - 1] & 0x80) === 0) return [sw, buf];
    }
  }
  const sw = { 8: 0, 16: 1, 32: 2 }[tbits] ?? 3;
  return [sw, uintToLEBytes(Number(uval), SW_BYTES[sw])];
}

function uintToLEBytes(v, size) {
  const buf = [];
  for (let i = 0; i < size; i++) buf.push((v >> (8 * i)) & 0xff);
  return buf;
}

function reconstructInt(otype, sw, raw) {
  const tbits = targetBits(otype);
  const storedBits = SW_BYTES[sw] * 8;
  if (isUnsigned(otype)) {
    const mask = maskForBits(tbits);
    return (raw >>> 0) & mask;
  }
  const signBit = (raw >> (storedBits - 1)) & 1;
  if (tbits <= storedBits) {
    const mask = maskForBits(tbits);
    let val = raw & mask;
    if (signBit === 1 && tbits < 32 && (val & pow2(tbits - 1))) val -= pow2(tbits);
    return val;
  }
  let extended;
  if (signBit === 1) {
    const extendMask = (~(pow2(storedBits) - 1)) & maskForBits(tbits);
    extended = raw | extendMask;
  } else extended = raw;
  if (extended >= pow2(tbits - 1)) extended -= pow2(tbits);
  return extended;
}

function validateRange(otype, val) {
  const ranges = {
    [OType.UINT8]: [0, 0xff], [OType.UINT16]: [0, 0xffff], [OType.UINT32]: [0, 0xffffffff],
    [OType.UINT64]: [0, Infinity], [OType.INT8]: [-128, 127], [OType.INT16]: [-32768, 32767],
    [OType.INT32]: [-2147483648, 2147483647], [OType.INT64]: [-Infinity, Infinity],
  };
  const [lo, hi] = ranges[otype] ?? [0, 0];
  if (val < lo || val > hi) throw new Error(`value ${val} out of range for otype ${otype}`);
}
