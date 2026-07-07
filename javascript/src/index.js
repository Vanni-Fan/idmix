/**
 * @vanni.fan/idmix — IDX v1.2 JavaScript implementation.
 */

export { DEFAULT_ALPHABET, IdMix } from './idmix.js';
export { Idx } from './idx_codec.js';
export { RadixCodec } from './alphabet.js';
export {
  encodeBytes,
  decodeString,
  resolveCodec,
  createFuncCodec,
  createBase64Codec,
  Base64Codec,
} from './codec.js';
export {
  OType,
  MAX_STRING_LEN,
  u8, u16, u32, u64,
  i8, i16, i32, i64,
  isTypedValue,
  typedValuesEqual,
  valuesEqual,
  materializeFromCrossLang,
} from './number.js';
