# idmix — C++ Implementation

> **中文:** [README.md](README.md)

C++17 library for **IDX v1.2 binary encoding** and the **idmix pluggable text layer**, matching the Go reference. See [arithmetic.md](../arithmetic.md) for the protocol.

## Architecture

Three layers, usable independently:

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (idmix.hpp)                                           │
│  encode / decode: integers/short strings → text              │
│  Internally: Idx encode → Codec.encode                       │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx (idx.hpp)  │                 │  Codec (codec.hpp)  │
│  Binary codec   │                 │  Binary ↔ text      │
└─────────────────┘                 └─────────────────────┘
```

| Component | Typical use |
|-----------|-------------|
| **Idx** | Compact self-describing binary as `std::vector<uint8_t>` |
| **encodeBytes / decodeString** | Wrap arbitrary binary as text |
| **IdMix** | access_key, short IDs: int/string sequences → obfuscated text |

## Build

```bash
cd cpp
cmake -B build
cmake --build build
ctest --test-dir build
```

## Quick start

### Full pipeline (IdMix)

```cpp
#include "idmix/idmix.hpp"

idmix::IdMix m = idmix::IdMix::newDefault();

std::vector<idmix::Value> values = {
    idmix::TypedValue::u16(5),
    idmix::TypedValue::i64(-1),
    idmix::TypedValue::u32(40),
    std::string("hello"),
};

auto s = m.encode(values);
auto out = m.decode(s);
```

### IDX binary layer only

```cpp
#include "idmix/idx.hpp"

idmix::Idx idx;
auto bin = idx.encode({idmix::TypedValue::u16(5), idmix::TypedValue::i64(-1)});
auto out = idx.decode(bin);
```

### Text layer only

```cpp
#include "idmix/codec.hpp"

auto text = idmix::encodeBytes({0x08, 0x96, 0x01});
auto raw = idmix::decodeString(text);
```

## Value type

`Value = std::variant<TypedValue, std::string>`

| Type | Description |
|------|-------------|
| `TypedValue` | Width-preserving integers (`u8()`…`i64()` factories) |
| `std::string` | UTF-8 text, **1–63 bytes** |

**Limits:**

- Up to **255** objects per encode (default `Idx(255, 32, 2)`)
- Strings up to **63** bytes
- Empty strings are rejected

## Main API

### Idx

```cpp
idmix::Idx idx(255, 32, 2);
idx.encode(values);
idx.encodeWithVariant(0, values);  // deterministic
idx.decode(data);
```

Single object: **1-byte** header; multiple objects: **2-byte** header with count.

### IdMix

```cpp
idmix::IdMix m("abcd");
m.encode(values);
m.encodeWithVariant(0, values);
m.decode(s);
m.idx();
m.codec();
```

### Codec

- `RadixCodec` — default base-62 (`DEFAULT_ALPHABET`)
- `ICodec` — extensible interface
- `encodeBytes` / `decodeString` — package-level helpers

## Canonical binary example

With `variant_id=0`, `[uint16(5), int64(-1), uint32(40)]` encodes as:

```
80 03 22 47 B5 1F
```

## Tests

Cross-language vectors: `../testdata/cross_language_vectors.json`.

## Version

Current version **0.4.0** (IDX v1.2). Breaking changes from 0.3.x (XID v1.1): header format and string support. See [arithmetic.md](../arithmetic.md).

## Links

- [Go reference](../golang/README_en.md)
- [GitHub repository](https://github.com/Vanni-Fan/idmix)
