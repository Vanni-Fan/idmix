/**
 * IdMix：IDX 二进制编码 + 可插拔文本层。
 */

import { RadixCodec, DEFAULT_ALPHABET } from './alphabet.js';
import { Idx } from './idx_codec.js';
import { normalizeObjects } from './number.js';

export { DEFAULT_ALPHABET };

export class IdMix {
  /** @param {object} [opts] @param {Idx} [opts.idx] @param {import('./codec.js').Codec} [opts.codec] */
  constructor(opts = {}) {
    this.idx = opts.idx ?? Idx.create();
    this.codec = opts.codec ?? RadixCodec.create(DEFAULT_ALPHABET);
  }

  /** @param {...(object | string | import('./codec.js').Codec | Idx)} args */
  static new(...args) {
    const opts = {};
    for (const arg of args) {
      if (typeof arg === 'string') opts.alphabet = arg;
      else if (arg instanceof Idx) opts.idx = arg;
      else if (arg && typeof arg.encode === 'function' && typeof arg.decode === 'function') opts.codec = arg;
      else if (arg && typeof arg === 'object') Object.assign(opts, arg);
    }
    if (opts.alphabet) opts.codec = RadixCodec.create(opts.alphabet);
    if (opts.maxObjects != null || opts.maxVariants != null || opts.checkBits != null) {
      opts.idx = Idx.create({
        maxObjects: opts.maxObjects,
        maxVariants: opts.maxVariants,
        checkBits: opts.checkBits,
      });
    }
    return new IdMix(opts);
  }

  /** @param {object} opts */
  static create(opts = {}) {
    const m = new IdMix({});
    if (opts.idx != null) {
      if (!(opts.idx instanceof Idx)) throw new Error('idx cannot be nil');
      m.idx = opts.idx;
    }
    if (opts.codec != null) {
      if (!opts.codec) throw new Error('codec cannot be nil');
      m.codec = opts.codec;
    }
    if (opts.alphabet != null) {
      m.codec = RadixCodec.create(opts.alphabet);
    }
    return m;
  }

  /** @param {Idx} idx */
  static withIdx(idx) {
    return { idx };
  }

  /** @param {import('./codec.js').Codec} codec */
  static withCodec(codec) {
    if (!codec) throw new Error('codec cannot be nil');
    return { codec };
  }

  /** @param {string} alphabet */
  static withAlphabet(alphabet) {
    return { alphabet };
  }

  getIdx() { return this.idx; }
  getCodec() { return this.codec; }

  /** @param {...unknown} values */
  encode(...values) {
    if (!values.length) throw new Error('at least one value is required');
    const variantId = Math.floor(Math.random() * this.idx.maxVariants);
    const data = this.encodeBinary(values, variantId);
    return this.codec.encode(data);
  }

  /** @param {string} s */
  decode(s) {
    const data = this.codec.decode(s);
    return this.idx.decode(data);
  }

  /** @param {number} variantID @param {...unknown} values */
  encodeWithVariant(variantID, ...values) {
    const data = this.encodeBinary(values, variantID);
    return this.codec.encode(data);
  }

  /** @param {unknown[]} values @param {number} variantID */
  encodeBinary(values, variantID) {
    const objects = normalizeObjects(values);
    return this.idx.encodeBinary(objects, variantID);
  }
}
