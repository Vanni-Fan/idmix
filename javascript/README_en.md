# idmix — JavaScript Implementation

> **中文:** [README.md](README.md)

JavaScript implementation of **IDX v1.2 binary encoding** and the **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

```
npm install @vanni.fan/idmix
```

**Requires:** Node.js 18+ (ESM, `node:test`)

## Architecture

The package has three layers, each **usable on its own**:

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (high-level API)                                     │
│  encode / decode: integers / short strings → text             │
│  Internally: Idx encode → Codec.encode                        │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec (interface)  │
│  Binary codec   │                 │  binary ↔ text      │
│  Standalone     │                 │  Standalone         │
└─────────────────┘                 └─────────────────────┘
```

| Component | Typical use |
|-----------|-------------|
| **Idx** | Compact self-describing binary only; output `Uint8Array` |
| **encodeBytes / decodeString** | Wrap Protobuf/CBOR/any binary as text |
| **IdMix** | access_key, short IDs: integer/string sequences → obfuscated text |

---

## Quick start

### Full pipeline (IdMix)

```javascript
import { IdMix, u16, i64, u32 } from '@vanni.fan/idmix';

const m = IdMix.new();

// Integers + short strings (≤63 bytes)
const s = m.encode(u16(5), i64(-1), u32(40), 'hello');
console.log('encoded:', s);

const list = m.decode(s);
// list[0] => { otype: 1, val: 5 }
// list[1] => { otype: 7, val: -1 }
// list[2] => { otype: 2, val: 40 }
// list[3] => 'hello'
```

### IDX binary layer only

```javascript
import { Idx, u16, i64 } from '@vanni.fan/idmix';

const idx = Idx.create();
const bin = idx.encode(u16(5), i64(-1));
const out = idx.decode(bin); // array; types match encoding input
```

### Text obfuscation layer only

```javascript
import { encodeBytes, decodeString } from '@vanni.fan/idmix';

const protoBytes = new Uint8Array([0x08, 0x96, 0x01]);
const text = encodeBytes(protoBytes);
const restored = decodeString(text);
```

---

## Supported input types

`Idx.encode` / `IdMix.encode` accept multiple arguments. Each element may be:

| Type | Notes |
|------|-------|
| `{ otype, val }` or `u8()`/`u16()`/`i32()`/`i64()` etc. | Width-tagged integers; decoded with same otype |
| `string` | UTF-8 text, **1–63 bytes** (byte length, not character count) |
| `Uint8Array` | Raw bytes, **1–63 bytes** |
| `number` / `bigint` | Treated as `int64` |

**Limits:**

- At most **255** objects per encode (`maxObjects`)
- Each string/byte slice at most **63** bytes
- Empty string or empty `Uint8Array` is rejected

---

## API reference

### Constants

```javascript
import { DEFAULT_ALPHABET, MAX_STRING_LEN, OType } from '@vanni.fan/idmix';
```

### Idx — binary layer

```javascript
import { Idx } from '@vanni.fan/idmix';

const idx = Idx.create({
  maxObjects: 255,   // 1–255, default 255
  maxVariants: 32,   // 1–32, default 32
  checkBits: 2,      // 1 or 2, default 2
});

idx.encode(...values);                    // variant_id = 0
idx.encodeWithVariant(variantID, ...values);
idx.decode(uint8Array);                   // returns array
```

**1-byte header** for a single object; **2-byte header** for multiple objects (includes count).

### Codec — text layer

```javascript
import { RadixCodec, encodeBytes, decodeString, createBase64Codec } from '@vanni.fan/idmix';

const rc = RadixCodec.create('abcd');
encodeBytes(data, rc);
decodeString(text, rc);

const b64 = createBase64Codec();
encodeBytes(data, b64);
```

### IdMix — high-level API

```javascript
import { IdMix, Idx } from '@vanni.fan/idmix';

// Default Idx + default RadixCodec
const m = IdMix.new();

// Custom alphabet
const m2 = IdMix.new('abcd');
// or
const m3 = IdMix.create({ alphabet: 'abcd' });

// Custom Idx
const idx = Idx.create({ maxObjects: 100, maxVariants: 16 });
const m4 = IdMix.create({ idx });

// Custom Codec
const m5 = IdMix.create({ codec: createBase64Codec() });

m.encode(...values);                      // random variant_id
m.encodeWithVariant(0, ...values);        // deterministic (for tests)
m.decode(text);
m.getIdx();
m.getCodec();
```

### Type helpers

```javascript
import { u8, u16, u32, u64, i8, i16, i32, i64, typedValuesEqual } from '@vanni.fan/idmix';

u16(5);   // { otype: 1, val: 5 }
i64(-1n); // { otype: 7, val: -1n }
```

---

## Configuration examples

### Custom Idx options

```javascript
const idx = Idx.create({
  maxObjects: 100,
  maxVariants: 16,
  checkBits: 2,
});
const m = IdMix.create({ idx });
```

### Custom Codec (XOR + Radix)

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

## Tests

```bash
cd javascript
npm test
```

Cross-language interoperability vectors: `../testdata/cross_language_vectors.json`

---

## File layout

```
javascript/
├── README.md           # Chinese documentation
├── README_en.md        # This file
├── package.json
└── src/
    ├── index.js        # Package entry
    ├── idmix.js        # IdMix high-level API
    ├── idx_codec.js    # Idx binary codec
    ├── codec.js        # Codec interface, Base64Codec, encodeBytes/decodeString
    ├── alphabet.js     # RadixCodec
    └── number.js       # Type conversion and helpers
```

---

## Related links

- [IDX v1.2 protocol spec](../arithmetic.md)
- [Go reference implementation](../golang/)
- [Project README](../README.md)
