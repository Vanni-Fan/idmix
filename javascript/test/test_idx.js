/**
 * Idx 配置项与字符串长度边界测试。
 * 运行: node --test test/test_idx.js
 */

import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { Idx } from '../src/idx_codec.js';
import { IdMix } from '../src/idmix.js';
import { u8, u16, i64 } from '../src/number.js';

describe('Idx', () => {
  it('rejects empty string', () => {
    const idx = Idx.create();
    assert.throws(() => idx.encode(''), /empty string/);
  });

  it('rejects empty Uint8Array', () => {
    const idx = Idx.create();
    assert.throws(() => idx.encode(new Uint8Array()), /empty byte slice/);
  });

  it('len 1 string ok', () => {
    const idx = Idx.create();
    const data = idx.encodeWithVariant(0, 'x');
    const out = idx.decode(data);
    assert.equal(out[0], 'x');
  });

  it('len 63 string ok', () => {
    const idx = Idx.create();
    const ok63 = 'a'.repeat(63);
    const data = idx.encodeWithVariant(0, ok63);
    assert.equal(data.length, 1 + 1 + 63);
    const out = idx.decode(data);
    assert.equal(out[0], ok63);
  });

  it('len 64 string rejected', () => {
    const idx = Idx.create();
    const tooLong = 'b'.repeat(64);
    assert.throws(() => idx.encode(tooLong), /exceeds max 63/);
  });

  it('len 64 bytes rejected', () => {
    const idx = Idx.create();
    assert.throws(() => idx.encode(new Uint8Array(64)), /exceeds max 63/);
  });

  it('idmix end-to-end 63-byte string', () => {
    const m = IdMix.new();
    const ok63 = 'a'.repeat(63);
    const str = m.encodeWithVariant(0, ok63);
    const list = m.decode(str);
    assert.equal(list[0], ok63);
  });

  it('idmix len 64 rejected', () => {
    const m = IdMix.new();
    assert.throws(() => m.encode('b'.repeat(64)), /exceeds max 63/);
  });

  it('maxObjects invalid config', () => {
    assert.throws(() => Idx.create({ maxObjects: 0 }), /maxObjects/);
    assert.throws(() => Idx.create({ maxObjects: 256 }), /maxObjects/);
  });

  it('encode over maxObjects limit', () => {
    const idx = Idx.create({ maxObjects: 2 });
    assert.throws(() => idx.encode(u8(1), u8(2), u8(3)), /too many objects/);
  });

  it('encode at maxObjects limit uses 2-byte header', () => {
    const idx = Idx.create({ maxObjects: 2 });
    const data = idx.encodeWithVariant(0, u8(1), u8(2));
    assert.equal(data[0] & 0x80, 0x80);
    assert.equal(data[1], 2);
    const out = idx.decode(data);
    assert.equal(out.length, 2);
  });

  it('maxVariants invalid config', () => {
    assert.throws(() => Idx.create({ maxVariants: 0 }), /maxVariants/);
    assert.throws(() => Idx.create({ maxVariants: 33 }), /maxVariants/);
  });

  it('encode variant out of range', () => {
    const idx = Idx.create({ maxVariants: 4 });
    assert.throws(() => idx.encodeWithVariant(4, u8(1)), /invalid variant_id/);
  });

  it('encode variant at limit', () => {
    const idx = Idx.create({ maxVariants: 4 });
    const data = idx.encodeWithVariant(3, u8(42));
    const out = idx.decode(data);
    assert.equal(out[0].otype, 0);
    assert.equal(out[0].val, 42);
  });

  it('decode rejects high variant when maxVariants smaller', () => {
    const large = Idx.create({ maxVariants: 32 });
    const data = large.encodeWithVariant(20, u8(7));
    const small = Idx.create({ maxVariants: 16 });
    assert.throws(() => small.decode(data), /invalid variant_id/);
  });

  it('checkBits invalid config', () => {
    assert.throws(() => Idx.create({ checkBits: 0 }), /checkBits/);
    assert.throws(() => Idx.create({ checkBits: 3 }), /checkBits/);
  });

  it('checkBits 1 roundtrip', () => {
    const idx = Idx.create({ checkBits: 1 });
    assert.equal(idx.checkMask, 0x01);
    const data = idx.encodeWithVariant(0, u16(5), i64(-1));
    const out = idx.decode(data);
    assert.equal(out.length, 2);
  });

  it('checkBits 1 rejects tamper', () => {
    const idx = Idx.create({ checkBits: 1 });
    const data = new Uint8Array(idx.encodeWithVariant(0, u8(1)));
    data[data.length - 1] ^= 0x01;
    assert.throws(() => idx.decode(data), /checksum mismatch/);
  });

  it('checkBits 2 default', () => {
    const idx = Idx.create();
    assert.equal(idx.checkBits, 2);
    assert.equal(idx.checkMask, 0x03);
  });
});
