/**
 * XID 文本层：自定义进制编解码。
 */

export class RadixCodec {
  /** @param {string} alphabet */
  constructor(alphabet) {
    const chars = [...alphabet];
    if (chars.length < 2) throw new Error('alphabet must have at least 2 unique characters');
    this.base = chars.length;
    this.chars = chars;
    /** @type {Map<string, number>} */
    this.fromCustom = new Map();
    for (let i = 0; i < chars.length; i++) {
      if (this.fromCustom.has(chars[i])) {
        throw new Error(`alphabet contains duplicate character ${chars[i]}`);
      }
      this.fromCustom.set(chars[i], i);
    }
  }

  /** @param {Uint8Array} data */
  encodeBytes(data) {
    if (data.length === 0) return this.chars[0];
    const wrapped = new Uint8Array(2 + data.length);
    wrapped[0] = (data.length >> 8) & 0xff;
    wrapped[1] = data.length & 0xff;
    wrapped.set(data, 2);
    let n = 0n;
    for (const b of wrapped) n = (n << 8n) | BigInt(b);
    return this.intToString(n);
  }

  /** @param {string} s @returns {Uint8Array} */
  decodeBytes(s) {
    if (!s) throw new Error('empty string');
    const n = this.stringToInt(s);
    let raw = n.toString(16);
    if (raw.length % 2) raw = '0' + raw;
    const bytes = new Uint8Array(raw.length / 2);
    for (let i = 0; i < bytes.length; i++) {
      bytes[i] = parseInt(raw.slice(i * 2, i * 2 + 2), 16);
    }
    for (const pad of [0, 1]) {
      const buf = new Uint8Array(pad + bytes.length);
      buf.set(bytes, pad);
      if (buf.length < 2) continue;
      const dataLen = (buf[0] << 8) | buf[1];
      if (buf.length !== 2 + dataLen) continue;
      return buf.slice(2);
    }
    throw new Error('invalid encoded data length');
  }

  /** @param {bigint} n */
  intToString(n) {
    if (n === 0n) return this.chars[0];
    const base = BigInt(this.base);
    const chars = [];
    while (n > 0n) {
      const rem = Number(n % base);
      chars.push(this.chars[rem]);
      n /= base;
    }
    return chars.reverse().join('');
  }

  /** @param {string} s */
  stringToInt(s) {
    let n = 0n;
    const base = BigInt(this.base);
    for (const ch of s) {
      if (!this.fromCustom.has(ch)) throw new Error(`invalid character ${ch}`);
      n = n * base + BigInt(this.fromCustom.get(ch));
    }
    return n;
  }
}
