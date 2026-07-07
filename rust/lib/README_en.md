# idmix — Rust Implementation

> **中文:** [README.md](README.md)

Rust library implementing **IDX v1.2 binary encoding** and a **pluggable idmix text layer**, matching the Go reference implementation. See [arithmetic.md](../../arithmetic.md) in the repository root for the protocol specification.

## Architecture

Three layers, each usable on its own:

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix                                                       │
│  encode / decode: integers / short strings → text             │
│  Internally: Idx encode → Codec.encode                        │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec (trait)      │
│  Binary codec   │                 │  binary ↔ text      │
└─────────────────┘                 └─────────────────────┘
```

| Component | Typical use |
|-----------|-------------|
| **Idx** | Compact self-describing binary; output `Vec<u8>` |
| **Codec + encode_bytes / decode_string** | Wrap any binary as text |
| **IdMix** | access_key, short IDs: integer/string sequences → obfuscated text |

## Installation

Add to `Cargo.toml` (path or crates.io when published):

```toml
[dependencies]
idmix = { path = "../rust/lib" }
```

## Quick start

### Full pipeline (IdMix)

```rust
use idmix::{IdMix, Value};

fn main() -> Result<(), idmix::IdMixError> {
    let m = IdMix::new()?;

    let values = [
        Value::U16(5),
        Value::I64(-1),
        Value::U32(40),
        Value::String("hello".into()),
    ];
    let s = m.encode(&values)?;
    let out = m.decode(&s)?;
    assert_eq!(out, values);
    Ok(())
}
```

### IDX binary layer only

```rust
use idmix::{Idx, Value};

let idx = Idx::new()?;
let bin = idx.encode(&[Value::U16(5), Value::I64(-1)])?;
let out = idx.decode(&bin)?;
```

### Text layer only

```rust
use idmix::{encode_bytes, decode_string};

let raw = [0x08u8, 0x96, 0x01];
let text = encode_bytes(&raw, None)?;
let back = decode_string(&text, None)?;
```

## Supported Value types

| Variant | Notes |
|---------|-------|
| `Value::U8` … `Value::U64` | Unsigned integers; original width preserved |
| `Value::I8` … `Value::I64` | Signed integers |
| `Value::String` | UTF-8 text, **1–63 bytes** (byte length) |
| `Value::Bytes` | Raw bytes, **1–63 bytes** |

**Limits:**

- At most **255** objects per encode (`IdxBuilder::max_objects`)
- Each string/byte slice at most **63** bytes
- Empty string or empty byte slice is rejected

## Main API

### Idx

```rust
let idx = Idx::builder()
    .max_objects(255)   // 1–255, default 255
    .max_variants(32)   // 1–32, default 32
    .check_bits(2)      // 1 or 2, default 2
    .build()?;

idx.encode(&values)?;
idx.encode_with_variant(0, &values)?;  // deterministic encoding
idx.decode(&data)?;
```

Single-object blocks use a **1-byte** header; multi-object blocks use **2 bytes** (including count).

### IdMix

```rust
use std::sync::Arc;
use idmix::{Base64Codec, IdMix, IdMixBuilder};

let m = IdMix::builder()
    .alphabet("abcd")                    // custom RadixCodec
    .codec(Arc::new(Base64Codec::new())) // or custom Codec
    .idx(idx)
    .build()?;

m.encode(&values)?;
m.encode_with_variant(0, &values)?;
m.decode(&s)?;
```

### Codec trait

- `RadixCodec` — default base-62 (`DEFAULT_ALPHABET`)
- `Base64Codec` — standard Base64
- `FuncCodec` — closure-based custom logic

```rust
use idmix::{Codec, encode_bytes, decode_string};

let s = encode_bytes(&data, Some(&radix_codec))?;
let raw = decode_string(&s, Some(&radix_codec))?;
```

## Spec binary example

With `variant_id=0`, `[uint16(5), int64(-1), uint32(40)]` encodes to:

```
80 03 22 47 B5 1F
```

## Tests

```bash
cd rust/lib
cargo test
```

Includes unit tests, Idx boundary tests (`tests/idx_tests.rs`), and cross-language vectors (`tests/extreme_cross_language.rs`).

## Version

Current version **0.4.0** (IDX v1.2). Breaking changes from 0.3.x (XID v1.1): header format and string support. See [arithmetic.md](../../arithmetic.md).
