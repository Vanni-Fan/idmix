/**
 * IDX v1.2 二进制层编解码（自描述整数/短字符串序列）。
 */

import { MAX_STRING_LEN, normalizeObjects, materializeObjects } from './number.js';

const SW_BYTES = [1, 2, 4, 8];

const OTYPE_UINT8 = 0;
const OTYPE_UINT16 = 1;
const OTYPE_UINT32 = 2;
const OTYPE_UINT64 = 3;
const OTYPE_INT8 = 4;
const OTYPE_INT16 = 5;
const OTYPE_INT32 = 6;
const OTYPE_INT64 = 7;

const EMBEDDED_OTYPE = [
  [OTYPE_UINT8, OTYPE_UINT16, OTYPE_UINT32, OTYPE_UINT64],
  [OTYPE_INT8, OTYPE_INT16, OTYPE_INT32, OTYPE_INT64],
];

export class Idx {
  /** @param {object} [opts] */
  constructor(opts = {}) {
    this.maxObjects = opts.maxObjects ?? 255;
    this.maxVariants = opts.maxVariants ?? 32;
    this.checkBits = opts.checkBits ?? 2;
    this.checkMask = (1 << this.checkBits) - 1;
  }

  /** @param {object} opts */
  static new(opts = {}) {
    return new Idx(opts);
  }

  /** @param {object} opts @returns {Idx} */
  static create(opts = {}) {
    const idx = new Idx(opts);
    if (idx.maxObjects < 1 || idx.maxObjects > 255) throw new Error('maxObjects must be between 1 and 255');
    if (idx.maxVariants < 1 || idx.maxVariants > 32) throw new Error('maxVariants must be between 1 and 32');
    if (idx.checkBits < 1 || idx.checkBits > 2) throw new Error('checkBits must be 1 or 2');
    idx.checkMask = (1 << idx.checkBits) - 1;
    return idx;
  }

  /** @param {...unknown} values */
  encode(...values) {
    if (!values.length) throw new Error('at least one value is required');
    if (values.length > this.maxObjects) throw new Error(`too many objects: ${values.length} (max ${this.maxObjects})`);
    const objects = normalizeObjects(values);
    return this.encodeBinary(objects, 0);
  }

  /** @param {number} variantID @param {...unknown} values */
  encodeWithVariant(variantID, ...values) {
    if (!values.length) throw new Error('at least one value is required');
    if (values.length > this.maxObjects) throw new Error(`too many objects: ${values.length} (max ${this.maxObjects})`);
    const objects = normalizeObjects(values);
    return this.encodeBinary(objects, variantID);
  }

  /** @param {Uint8Array} data */
  decode(data) {
    const objects = this.decodeBinary(data);
    return materializeObjects(objects);
  }

  /** @param {ReturnType<typeof normalizeObjects>} objects @param {number} variantID */
  encodeBinary(objects, variantID) {
    if (variantID < 0 || variantID >= this.maxVariants) {
      throw new Error(`invalid variant_id ${variantID} (max ${this.maxVariants - 1})`);
    }

    const parts = [];
    for (const obj of objects) parts.push(...encodeObject(obj));

    const objBytes = new Uint8Array(parts);
    const mask = (variantID * 0x9d + 0x37) & 0xff;
    for (let i = 0; i < objBytes.length; i++) objBytes[i] ^= mask;

    const count = objects.length;
    const headerLen = count > 1 ? 2 : 1;
    const data = new Uint8Array(headerLen + objBytes.length);
    if (count === 1) {
      data[0] = variantID << this.checkBits;
    } else {
      data[0] = 0x80 | (variantID << this.checkBits);
      data[1] = count;
    }
    data.set(objBytes, headerLen);

    let xorSum = 0;
    for (const b of data) xorSum ^= b;
    data[0] |= xorSum & this.checkMask;
    return data;
  }

  /** @param {Uint8Array} data */
  decodeBinary(data) {
    if (data.length < 1) throw new Error('invalid data: too short');

    const byte0 = data[0];
    const check = byte0 & this.checkMask;
    const multi = (byte0 & 0x80) !== 0;
    const variantID = (byte0 & 0x7f) >> this.checkBits;

    if (variantID >= this.maxVariants) {
      throw new Error(`invalid variant_id ${variantID} (max ${this.maxVariants - 1})`);
    }

    let headerLen = 1;
    let count = 1;
    if (multi) {
      if (data.length < 2) throw new Error('invalid data: missing count byte');
      headerLen = 2;
      count = data[1];
      if (count < 2 || count > this.maxObjects) throw new Error(`invalid count ${count}`);
    }

    const verify = new Uint8Array(data);
    verify[0] &= ~this.checkMask;
    let xorSum = 0;
    for (const b of verify) xorSum ^= b;
    if ((xorSum & this.checkMask) !== check) throw new Error('checksum mismatch');

    const objData = new Uint8Array(data.slice(headerLen));
    const mask = (variantID * 0x9d + 0x37) & 0xff;
    for (let i = 0; i < objData.length; i++) objData[i] ^= mask;

    const result = [];
    let pos = 0;
    for (let i = 0; i < count; i++) {
      if (pos >= objData.length) throw new Error('premature end of data');
      const [obj, n] = decodeObject(objData.slice(pos));
      result.push(obj);
      pos += n;
    }
    if (pos !== objData.length) throw new Error('extra bytes after data objects');
    return result;
  }
}

/** @param {{ isString: boolean, otype?: number, val?: number | bigint, str?: Uint8Array }} obj */
function encodeObject(obj) {
  if (obj.isString) {
    const n = obj.str.length;
    if (n < 1 || n > MAX_STRING_LEN) throw new Error(`string length ${n} out of range [1, ${MAX_STRING_LEN}]`);
    const out = new Uint8Array(1 + n);
    out[0] = 0xc0 | n;
    out.set(obj.str, 1);
    return out;
  }

  validateRange(obj.otype, obj.val);
  const embedded = tryEmbeddedHead(obj.otype, obj.val);
  if (embedded !== null) return new Uint8Array([embedded]);

  const [sw, payload] = payloadForNumber(obj.otype, obj.val);
  const head = 0x80 | (sw << 4) | obj.otype;
  const out = new Uint8Array(1 + payload.length);
  out[0] = head;
  out.set(payload, 1);
  return out;
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
    return [{ isString: false, otype, val }, 1];
  }

  if ((head & 0x40) !== 0) {
    const n = head & 0x3f;
    if (n < 1 || n > MAX_STRING_LEN) throw new Error(`invalid string length ${n}`);
    if (data.length < 1 + n) throw new Error('truncated string payload');
    const str = new Uint8Array(n);
    str.set(data.slice(1, 1 + n));
    return [{ isString: true, str }, 1 + n];
  }

  const sw = (head >> 4) & 3;
  const otype = head & 0x0f;
  if (otype > OTYPE_INT64) throw new Error(`invalid otype ${otype}`);
  const numBytes = SW_BYTES[sw];
  if (data.length < 1 + numBytes) throw new Error('truncated object payload');
  const val = valueFromPayload(otype, data.slice(1, 1 + numBytes));
  validateRange(otype, val);
  return [{ isString: false, otype, val }, 1 + numBytes];
}

/** @param {number} otype @param {number | bigint} val */
function payloadForNumber(otype, val) {
  if (otype === OTYPE_UINT64) {
    const mag = toBigInt(val);
    const sw = swFromMagnitude(mag);
    return [sw, uintToLEBytes(mag, SW_BYTES[sw])];
  }
  if (isUnsigned(otype)) {
    const bval = toBigInt(val);
    if (bval < 0n) throw new Error(`negative value ${val} for unsigned otype ${otype}`);
    const sw = swFromMagnitude(bval);
    return [sw, uintToLEBytes(bval, SW_BYTES[sw])];
  }
  const sw = swFromSignedValue(val);
  return [sw, signedToLEBytes(val, SW_BYTES[sw])];
}

/** @param {number} otype @param {Uint8Array} payload */
function valueFromPayload(otype, payload) {
  if (isUnsigned(otype)) {
    const mag = leBytesToUint(payload);
    if (otype !== OTYPE_UINT64 && mag > BigInt(Number.MAX_SAFE_INTEGER)) {
      throw new Error(`value out of range for otype ${otype}`);
    }
    return mag <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(mag) : mag;
  }
  return leBytesToSigned(payload);
}

/** @param {number | bigint} val */
function swFromSignedValue(val) {
  const bval = toBigInt(val);
  if (bval >= -128n && bval <= 127n) return 0;
  if (bval >= -32768n && bval <= 32767n) return 1;
  if (bval >= -2147483648n && bval <= 2147483647n) return 2;
  return 3;
}

/** @param {number | bigint} val @param {number} size */
function signedToLEBytes(val, size) {
  const buf = new Uint8Array(size);
  const view = new DataView(buf.buffer);
  const bval = toBigInt(val);
  switch (size) {
    case 1: view.setInt8(0, Number(bval)); break;
    case 2: view.setInt16(0, Number(bval), true); break;
    case 4: view.setInt32(0, Number(bval), true); break;
    case 8: view.setBigInt64(0, bval, true); break;
    default: throw new Error('invalid payload size');
  }
  return buf;
}

/** @param {Uint8Array} payload */
function leBytesToSigned(payload) {
  const buf = payload instanceof Uint8Array ? payload : new Uint8Array(payload);
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  switch (buf.length) {
    case 1: return view.getInt8(0);
    case 2: return view.getInt16(0, true);
    case 4: return view.getInt32(0, true);
    case 8: {
      const v = view.getBigInt64(0, true);
      return v >= BigInt(Number.MIN_SAFE_INTEGER) && v <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(v) : v;
    }
    default: throw new Error('invalid payload size');
  }
}

/** @param {Uint8Array} payload */
function leBytesToUint(payload) {
  let u = 0n;
  for (let i = 0; i < payload.length; i++) u |= BigInt(payload[i]) << BigInt(8 * i);
  return u;
}

function isUnsigned(otype) { return otype <= OTYPE_UINT64; }

function widthBits(otype) {
  switch (otype) {
    case OTYPE_UINT8: case OTYPE_INT8: return 0;
    case OTYPE_UINT16: case OTYPE_INT16: return 1;
    case OTYPE_UINT32: case OTYPE_INT32: return 2;
    default: return 3;
  }
}

/** @param {number} otype @param {number | bigint} val */
function magnitudeFromTyped(otype, val) {
  const bval = toBigInt(val);
  if (isUnsigned(otype)) return [bval, false];
  if (bval < 0n) return [-bval, true];
  return [bval, false];
}

/** @param {number | bigint} mag */
function swFromMagnitude(mag) {
  const m = toBigInt(mag);
  if (m < 256n) return 0;
  if (m < 65536n) return 1;
  if (m < 4294967296n) return 2;
  return 3;
}

/** @param {number} otype @param {number | bigint} val */
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

/** @param {number | bigint} v @param {number} size */
function uintToLEBytes(v, size) {
  const bval = toBigInt(v);
  const buf = new Uint8Array(size);
  for (let i = 0; i < size; i++) buf[i] = Number((bval >> BigInt(8 * i)) & 0xffn);
  return buf;
}

/** @param {number | bigint} v */
function toBigInt(v) {
  return typeof v === 'bigint' ? v : BigInt(v);
}

/** @param {number} otype @param {number | bigint} val */
function validateRange(otype, val) {
  const bval = toBigInt(val);
  const ranges = {
    [OTYPE_UINT8]: [0n, 0xffn], [OTYPE_UINT16]: [0n, 0xffffn], [OTYPE_UINT32]: [0n, 0xffffffffn],
    [OTYPE_UINT64]: [0n, 0xffffffffffffffffn], [OTYPE_INT8]: [-128n, 127n], [OTYPE_INT16]: [-32768n, 32767n],
    [OTYPE_INT32]: [-2147483648n, 2147483647n], [OTYPE_INT64]: [-0x8000000000000000n, 0x7fffffffffffffffn],
  };
  const [lo, hi] = ranges[otype] ?? [0n, 0n];
  if (bval < lo || bval > hi) throw new Error(`value ${val} out of range for otype ${otype}`);
}
