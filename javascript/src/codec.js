/**
 * idmix 文本层：可插拔 Codec 接口及内置实现。
 */

import { RadixCodec, DEFAULT_ALPHABET } from './alphabet.js';

/** @typedef {{ encode(data: Uint8Array): string, decode(s: string): Uint8Array }} Codec */

let defaultCodec = null;

function defaultCodecInstance() {
  if (!defaultCodec) defaultCodec = RadixCodec.create(DEFAULT_ALPHABET);
  return defaultCodec;
}

/** @param {Codec} [codec] */
export function resolveCodec(codec) {
  return codec ?? defaultCodecInstance();
}

/** @param {Uint8Array} data @param {Codec} [codec] */
export function encodeBytes(data, codec) {
  return resolveCodec(codec).encode(data);
}

/** @param {string} s @param {Codec} [codec] */
export function decodeString(s, codec) {
  return resolveCodec(codec).decode(s);
}

/** @param {{ encodeFn?: (data: Uint8Array) => string, decodeFn?: (s: string) => Uint8Array }} fns */
export function createFuncCodec(fns) {
  return {
    encode(data) {
      if (!fns.encodeFn) throw new Error('codec function is nil');
      return fns.encodeFn(data);
    },
    decode(s) {
      if (!fns.decodeFn) throw new Error('codec function is nil');
      return fns.decodeFn(s);
    },
  };
}

export class Base64Codec {
  encode(data) {
    return Buffer.from(data).toString('base64');
  }

  decode(s) {
    return new Uint8Array(Buffer.from(s, 'base64'));
  }
}

export function createBase64Codec() {
  return new Base64Codec();
}

export { DEFAULT_ALPHABET };
