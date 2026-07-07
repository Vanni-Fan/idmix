/**
 * 跨语言测试向量加载与断言辅助。
 */

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { materializeFromCrossLang, typedValuesEqual } from '../src/number.js';

const VECTORS_PATH = join(dirname(fileURLToPath(import.meta.url)), '../../testdata/cross_language_vectors.json');

/** @typedef {{ otype: number, val: string, str?: string }} CrossLangValue */
/** @typedef {{ name: string, variant: number, values: CrossLangValue[], encoded: string }} CrossLangCase */
/** @typedef {{ alphabet: string, cases: CrossLangCase[] }} CrossLangFile */

export function loadCrossLanguageVectors() {
  /** @type {CrossLangFile} */
  return JSON.parse(readFileSync(VECTORS_PATH, 'utf8'));
}

/** @param {CrossLangValue} v */
export function materializeCrossLangValue(v) {
  return materializeFromCrossLang(v.otype, v.val, v.str);
}

export { typedValuesEqual };

export const EXTREME = {
  UINT32_MAX: 4294967295,
  INT32_MIN: -2147483648,
  INT64_MIN: -9223372036854775808n,
  INT64_MAX: 9223372036854775807n,
  UINT64_MAX: 18446744073709551615n,
};
