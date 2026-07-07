# vanni-idmix — Python Implementation

> **中文:** [README.md](README.md)

Python reference implementation providing **IDX v1.2 binary encoding** and a **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

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
│  Idx            │                 │  Codec (protocol)   │
│  Binary codec   │                 │  binary ↔ text      │
│  Standalone     │                 │  Standalone         │
└─────────────────┘                 └─────────────────────┘
```

| Component | Typical use |
|-----------|-------------|
| **Idx** | Compact self-describing binary only; output `bytes` |
| **Codec + encode_bytes / decode_string** | Wrap Protobuf/CBOR/any binary as text |
| **IdMix** | access_key, short IDs: integer/string sequences → obfuscated text |

## Installation

```bash
pip install vanni-idmix
```

**Requires:** Python 3.9+

## Quick start

### Full pipeline (IdMix)

```python
from idmix import IdMix, u16, i64, u32

m = IdMix.new()

# Integers + short strings (≤63 bytes)
s = m.encode(u16(5), i64(-1), u32(40), "hello")
print("encoded:", s)

values = m.decode(s)
print(values)  # [5, -1, 40, 'hello']
```

> Decode returns `int` / `str`. Use `u16()`, `i64()`, etc. when you need Go-aligned bit-width encoding.

### IDX binary layer only

```python
from idmix import Idx, u16, i64

idx = Idx.new()
data = idx.encode(u16(5), i64(-1))
out = idx.decode(data)
```

### Text obfuscation layer only

```python
from idmix import encode_bytes, decode_string

proto_bytes = bytes([0x08, 0x96, 0x01])
text = encode_bytes(proto_bytes)
raw = decode_string(text)
```

## Supported input types

`Idx.encode` / `IdMix.encode` accept variadic arguments. Each element may be:

| Python type | Description |
|-------------|-------------|
| `int` | Integer (stored as int64) |
| `TypedInt` / `u8()`…`i64()` | Bit-width–aware integers aligned with Go types |
| `str` | UTF-8 text, **1~63 bytes** (byte count, not character count) |
| `bytes` / `bytearray` | Raw byte string, **1~63 bytes** |

**Limits**:

- Up to **255** objects per encode (`with_max_objects` configurable)
- Max **63** bytes per string/byte segment
- Empty strings and empty `bytes` are rejected

## Module layout

| File | Role |
|------|------|
| `idx_codec.py` | `Idx` binary codec |
| `codec.py` | `Codec` protocol, `Base64Codec`, `FuncCodec`, module functions |
| `alphabet.py` | `RadixCodec` default radix codec |
| `number.py` | Type normalization and `TypedInt` |
| `idmix.py` | `IdMix` high-level API |

## Configuration examples

### Custom Idx options

```python
from idmix import IdMix, Idx, with_idx, with_max_objects, with_max_variants, with_check_bits

idx = Idx.new(
    with_max_objects(100),
    with_max_variants(16),
    with_check_bits(2),
)
m = IdMix.new(with_idx(idx))
```

### Custom alphabet

```python
from idmix import IdMix, with_alphabet

m = IdMix.new(with_alphabet("abcd"))
```

### Base64 as text layer

```python
from idmix import IdMix, Base64Codec, with_codec

m = IdMix.new(with_codec(Base64Codec.new()))
```

### Custom Codec (XOR + Radix)

```python
from idmix import FuncCodec, RadixCodec, encode_bytes, decode_string

inner = RadixCodec.new("abcd")
key = 0x5A

codec = FuncCodec(
    encode_fn=lambda data: inner.encode(bytes(b ^ key for b in data)),
    decode_fn=lambda s: bytes(b ^ key for b in inner.decode(s)),
)
text = encode_bytes(b"\xde\xad", codec)
```

## Tests

```bash
cd python
python -m unittest discover -s tests -v
```

Cross-language vectors live in `../testdata/cross_language_vectors.json`.

## Links

- [Protocol spec](../arithmetic.md)
- [Go reference](../golang/README_en.md)
- [GitHub repository](https://github.com/Vanni-Fan/idmix)
