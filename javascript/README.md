# idmix — JavaScript 实现

> **English:** [README_en.md](README_en.md)

JavaScript 参考实现，提供 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

```
npm install @vanni.fan/idmix
```

**要求**：Node.js 18+（ESM、`node:test`）

## 架构概览

本包分三层，**可独立使用**：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（高级封装）                                            │
│  encode / decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → Codec.encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec（接口）       │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
│  可单独使用      │                 │  可单独使用          │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 只需紧凑自描述二进制，输出 `Uint8Array` |
| **encodeBytes / decodeString** | 包装 Protobuf/CBOR/任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

---

## 快速开始

### 完整流程（IdMix）

```javascript
import { IdMix, u16, i64, u32 } from '@vanni.fan/idmix';

const m = IdMix.new();

// 整数 + 短字符串（≤63 字节）
const s = m.encode(u16(5), i64(-1), u32(40), 'hello');
console.log('encoded:', s);

const list = m.decode(s);
// list[0] => { otype: 1, val: 5 }
// list[1] => { otype: 7, val: -1 }
// list[2] => { otype: 2, val: 40 }
// list[3] => 'hello'
```

### 仅 IDX 二进制层

```javascript
import { Idx, u16, i64 } from '@vanni.fan/idmix';

const idx = Idx.create();
const bin = idx.encode(u16(5), i64(-1));
const out = idx.decode(bin); // 数组，类型与编码时一致
```

### 仅文本混淆层

```javascript
import { encodeBytes, decodeString } from '@vanni.fan/idmix';

const protoBytes = new Uint8Array([0x08, 0x96, 0x01]);
const text = encodeBytes(protoBytes);
const restored = decodeString(text);
```

---

## 支持的输入类型

`Idx.encode` / `IdMix.encode` 接受多个参数，每个元素可为：

| 类型 | 说明 |
|------|------|
| `{ otype, val }` 或 `u8()`/`u16()`/`i32()`/`i64()` 等 | 带宽度整数；解码后还原相同 otype |
| `string` | UTF-8 文本，**1–63 字节**（字节长度，非字符数） |
| `Uint8Array` | 原始字节，**1–63 字节** |
| `number` / `bigint` | 视为 `int64` |

**限制：**

- 单次最多 **255** 个对象（`maxObjects`）
- 每个字符串/字节切片最多 **63** 字节
- 空字符串或空 `Uint8Array` 会被拒绝

---

## API 参考

### 常量

```javascript
import { DEFAULT_ALPHABET, MAX_STRING_LEN, OType } from '@vanni.fan/idmix';
```

### Idx — 二进制层

```javascript
import { Idx } from '@vanni.fan/idmix';

const idx = Idx.create({
  maxObjects: 255,   // 1–255，默认 255
  maxVariants: 32,   // 1–32，默认 32
  checkBits: 2,      // 1 或 2，默认 2
});

idx.encode(...values);                    // variant_id = 0
idx.encodeWithVariant(variantID, ...values);
idx.decode(uint8Array);                   // 返回数组
```

单对象时 header **1 字节**；多对象时 **2 字节**（含 count）。

### Codec — 文本层

```javascript
import { RadixCodec, encodeBytes, decodeString, createBase64Codec } from '@vanni.fan/idmix';

const rc = RadixCodec.create('abcd');
encodeBytes(data, rc);
decodeString(text, rc);

const b64 = createBase64Codec();
encodeBytes(data, b64);
```

### IdMix — 高级封装

```javascript
import { IdMix, Idx } from '@vanni.fan/idmix';

// 默认 Idx + 默认 RadixCodec
const m = IdMix.new();

// 自定义字符表
const m2 = IdMix.new('abcd');
// 或
const m3 = IdMix.create({ alphabet: 'abcd' });

// 自定义 Idx
const idx = Idx.create({ maxObjects: 100, maxVariants: 16 });
const m4 = IdMix.create({ idx });

// 自定义 Codec
const m5 = IdMix.create({ codec: createBase64Codec() });

m.encode(...values);                      // 随机 variant_id
m.encodeWithVariant(0, ...values);        // 确定性编码（测试用）
m.decode(text);
m.getIdx();
m.getCodec();
```

### 类型辅助函数

```javascript
import { u8, u16, u32, u64, i8, i16, i32, i64, typedValuesEqual } from '@vanni.fan/idmix';

u16(5);   // { otype: 1, val: 5 }
i64(-1n); // { otype: 7, val: -1n }
```

---

## 配置示例

### 自定义 Idx 参数

```javascript
const idx = Idx.create({
  maxObjects: 100,
  maxVariants: 16,
  checkBits: 2,
});
const m = IdMix.create({ idx });
```

### 自定义 Codec（异或 + Radix）

```javascript
import { RadixCodec, createFuncCodec, DEFAULT_ALPHABET } from '@vanni.fan/idmix';

const inner = RadixCodec.create(DEFAULT_ALPHABET);
const xorKey = 0x5a;

const xorCodec = createFuncCodec({
  encodeFn(data) {
    const buf = new Uint8Array(data.length);
    for (let i = 0; i < data.length; i++) buf[i] = data[i] ^ xorKey;
    return inner.encode(buf);
  },
  decodeFn(s) {
    const buf = inner.decode(s);
    for (let i = 0; i < buf.length; i++) buf[i] ^= xorKey;
    return buf;
  },
});

const m = IdMix.create({ codec: xorCodec });
```

---

## 测试

```bash
cd javascript
npm test
```

跨语言互操作向量：`../testdata/cross_language_vectors.json`

---

## 文件结构

```
javascript/
├── README.md           # 中文文档（本文档）
├── README_en.md        # English documentation
├── package.json
└── src/
    ├── index.js        # 包入口
    ├── idmix.js        # IdMix 高级封装
    ├── idx_codec.js    # Idx 二进制编解码
    ├── codec.js        # Codec 接口、Base64Codec、encodeBytes/decodeString
    ├── alphabet.js     # RadixCodec
    └── number.js       # 类型转换与辅助函数
```

---

## 相关链接

- [IDX v1.2 协议规范](../arithmetic.md)
- [Go 参考实现](../golang/)
- [项目总览 README](../README.md)
