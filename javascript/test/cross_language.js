/**
 * 跨语言测试向量加载与断言辅助。
 */

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { typedValuesEqual, u8, u16, u32, u64, i8, i16, i32, i64 } from '../src/typed_value.js';

const VECTORS_PATH = join(dirname(fileURLToPath(import.meta.url)), '../../testdata/cross_language_vectors.json');

/** @typedef {{ name: string, variant: number, values: { otype: number, val: number }[], encoded: string }} CrossLangCase */
/** @typedef {{ alphabet: string, cases: CrossLangCase[] }} CrossLangFile */

export function loadCrossLanguageVectors() {
  /** @type {CrossLangFile} */
  return JSON.parse(readFileSync(VECTORS_PATH, 'utf8'));
}

/** @param {number} otype @param {string|number} val */
export function materializeOtypeVal(otype, val) {
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

export { typedValuesEqual };

export const EXTREME = {
  UINT32_MAX: 4294967295,
  INT32_MIN: -2147483648,
  INT64_MIN: -9223372036854775808n,
  INT64_MAX: 9223372036854775807n,
  UINT64_MAX: 18446744073709551615n,
};
