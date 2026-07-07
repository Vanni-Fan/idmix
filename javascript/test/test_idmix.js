/**
 * idmix 测试套件。运行: npm test
 */

import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { IdMix } from '../src/idmix.js';
import { RadixCodec } from '../src/alphabet.js';
import { encodeBytes, decodeString } from '../src/codec.js';
import { u8, u16, u32, i32, i64, u64, typedValuesEqual, valuesEqual } from '../src/number.js';
import { loadCrossLanguageVectors, materializeCrossLangValue, EXTREME } from './cross_language.js';

/** @param {Uint8Array} b */
function hex(b) {
  return [...b].map(x => x.toString(16).padStart(2, '0').toUpperCase()).join(' ') || '(empty)';
}

/** @param {import('../src/idmix.js').IdMix} m @param {string} title @param {unknown[]} values */
function logRoundTrip(m, title, values) {
  console.log(`\n${'─'.repeat(40)}`);
  console.log(`▶ ${title}`);
  console.log(`  字符表: ${JSON.stringify(m.getCodec().alphabet())} (进制=${m.getCodec().baseSize()})`);
  values.forEach((v, i) => {
    const desc = typeof v === 'string' ? `str=${JSON.stringify(v)}` : `otype=${v.otype} val=${v.val}`;
    console.log(`  编码输入[${i}] ${desc}`);
  });
  const encoded = m.encode(...values);
  const raw = m.getCodec().decode(encoded);
  console.log(`  二进制: ${hex(raw)} (${raw.length} bytes)`);
  console.log(`  字符串: ${JSON.stringify(encoded)} (len=${encoded.length})`);
  const decoded = m.decode(encoded);
  decoded.forEach((v, i) => {
    const desc = typeof v === 'string' ? `str=${JSON.stringify(v)}` : `otype=${v.otype} val=${v.val}`;
    console.log(`  解码输出[${i}] ${desc}`);
  });
  values.forEach((want, i) => {
    const got = decoded[i];
    const mark = valuesEqual(got, want) ? '✓' : '✗';
    console.log(`  校验[${i}]: ${mark}`);
    assert.ok(valuesEqual(got, want));
  });
  return decoded;
}

describe('IdMix', () => {
  it('spec example binary (variant=0)', () => {
    const m = IdMix.new();
    const typed = [u16(5), i64(-1), u32(40)];
    const data = m.encodeBinary(typed, 0);
    const want = new Uint8Array([0x80, 0x03, 0x22, 0x47, 0xb5, 0x1f]);
    console.log(`\n▶ 规范二进制块 (variant=0): ${hex(data)}`);
    assert.deepEqual(data, want);
    console.log('  与 arithmetic.md 第7节示例一致 ✓');
  });

  it('round trip basic', () => {
    logRoundTrip(IdMix.new(), '规范示例: u16(5), i64(-1), u32(40)', [u16(5), i64(-1), u32(40)]);
  });

  it('round trip with strings', () => {
    logRoundTrip(IdMix.new(), '字符串: "hello", u16(5), "世界"', ['hello', u16(5), '世界']);
  });

  it('round trip uint32 large single', () => {
    const out = logRoundTrip(IdMix.new(), '单值 u32(2000000000)', [u32(2_000_000_000)]);
    assert.equal(out[0].val, 2_000_000_000);
  });

  it('custom alphabet', () => {
    logRoundTrip(IdMix.new('abcd'), '四进制 abcd', [u16(100), i32(-10), u8(3)]);
  });

  it('checksum rejects tampering', () => {
    const m = IdMix.new();
    const data = new Uint8Array(m.encodeBinary([u32(1)], 0));
    console.log(`\n▶ 校验和拒绝测试`);
    console.log(`  原始: ${hex(data)}`);
    data[data.length - 1] ^= 0x01;
    console.log(`  篡改: ${hex(data)}`);
    const tampered = m.getCodec().encode(data);
    assert.throws(() => m.decode(tampered), /checksum mismatch/);
    console.log('  解码拒绝 ✓');
  });

  it('multiple encodings differ (polymorphism)', () => {
    const m = IdMix.new();
    const seen = new Set();
    for (let i = 0; i < 50; i++) seen.add(m.encode(u32(42)));
    console.log(`\n▶ 变体多态: u32(42) 编码 50 次 => ${seen.size} 种不同字符串`);
    assert.ok(seen.size >= 2);
  });

  it('extreme values round trip', () => {
    const m = IdMix.new();
    const cases = [
      ['uint32_max', [u32(EXTREME.UINT32_MAX)]],
      ['int32_min', [i32(EXTREME.INT32_MIN)]],
      ['int64_min', [i64(EXTREME.INT64_MIN)]],
      ['int64_max', [i64(EXTREME.INT64_MAX)]],
      ['uint64_max', [u64(EXTREME.UINT64_MAX)]],
    ];
    for (const [name, values] of cases) {
      const decoded = logRoundTrip(m, name, values);
      values.forEach((want, i) => assert.ok(typedValuesEqual(decoded[i], want)));
    }
  });

  it('cross-language vectors decode', () => {
    const f = loadCrossLanguageVectors();
    const m = IdMix.new(f.alphabet);
    for (const c of f.cases) {
      console.log(`\n▶ cross-language decode: ${c.name}`);
      const decoded = m.decode(c.encoded);
      assert.equal(decoded.length, c.values.length, c.name);
      c.values.forEach((want, i) => {
        const expected = materializeCrossLangValue(want);
        assert.ok(valuesEqual(decoded[i], expected), `[${i}] ${c.name}`);
      });
    }
  });

  it('cross-language vectors encode deterministic', () => {
    const f = loadCrossLanguageVectors();
    const m = IdMix.new(f.alphabet);
    for (const c of f.cases) {
      const inputs = c.values.map(materializeCrossLangValue);
      const enc = m.encodeWithVariant(c.variant, ...inputs);
      assert.equal(enc, c.encoded, c.name);
    }
  });

  it('radix round trip via encodeBytes/decodeString', () => {
    const raw = new Uint8Array([0x80, 0x03, 0x22, 0x47, 0xb5, 0x1f]);
    const rc = RadixCodec.create('abcd');
    const enc = encodeBytes(raw, rc);
    const dec = decodeString(enc, rc);
    assert.deepEqual(dec, raw);
  });
});
