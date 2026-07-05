/**
 * XID v1.1 编解码器主入口。
 */

import { RadixCodec } from './alphabet.js';
import * as xidCodec from './xid_codec.js';

export const DEFAULT_ALPHABET = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';

export class IdMix {
  /**
   * @param {object} [opts]
   * @param {string} [opts.alphabet]
   * @param {number} [opts.maxObjects]
   * @param {number} [opts.maxVariants]
   * @param {number} [opts.checkBits]
   */
  constructor(opts = {}) {
    const alphabet = opts.alphabet ?? DEFAULT_ALPHABET;
    this.radix = new RadixCodec(alphabet);
    this.maxObjects = opts.maxObjects ?? 511;
    this.maxVariants = opts.maxVariants ?? 32;
    this.checkBits = opts.checkBits ?? 2;
    this.countBits = 0;
    this.variantBits = 0;
    this.checkMask = 0;
    this.countMask = 0;
    this.variantMask = 0;
    this.countShift = 0;
    this.variantShift = 0;
    this._finalizeLayout();
  }

  static new(alphabet) {
    return new IdMix({ alphabet });
  }

  /** @param {...import('./typed_value.js').TypedValue} values */
  encode(...values) {
    if (!values.length) throw new Error('at least one value is required');
    if (values.length > this.maxObjects) throw new Error(`too many objects: ${values.length}`);
    const variantId = Math.floor(Math.random() * this.maxVariants);
    const data = xidCodec.encodeBinary(this, values, variantId);
    return this.radix.encodeBytes(data);
  }

  /** @param {string} s @returns {import('./typed_value.js').TypedValue[]} */
  decode(s) {
    const data = this.radix.decodeBytes(s);
    return xidCodec.decodeBinary(this, data);
  }

  _finalizeLayout() {
    const variantBits = this.maxVariants <= 1 ? 1 : bitLen(this.maxVariants - 1);
    const countBits = this.maxObjects <= 1 ? 1 : bitLen(this.maxObjects);
    const total = this.checkBits + countBits + variantBits;
    if (total > 16) throw new Error(`header layout exceeds 16 bits: ${total}`);
    this.countBits = countBits;
    this.variantBits = variantBits;
    this.checkMask = (1 << this.checkBits) - 1;
    this.countMask = ((1 << countBits) - 1) << this.checkBits;
    this.variantMask = ((1 << variantBits) - 1) << (this.checkBits + countBits);
    this.countShift = this.checkBits;
    this.variantShift = this.checkBits + countBits;
  }
}

function bitLen(n) {
  if (n <= 0) return 1;
  return 32 - Math.clz32(n);
}
