# idmix — Go Reference Implementation

> **中文:** [README.md](README.md)

Go reference implementation providing **IDX binary encoding** and a **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

```
github.com/Vanni-Fan/idmix/golang
```

## Architecture

The package has three layers, each **usable on its own**:

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (high-level API)                                     │
│  Encode / Decode: integers / short strings → text             │
│  Internally: Idx encode → Codec.Encode                        │
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
| **Idx** | Compact self-describing binary only; output `[]byte` |
| **Codec + EncodeBytes / DecodeString** | Wrap Protobuf/CBOR/any binary as text |
| **IdMix** | access_key, short IDs: integer/string sequences → obfuscated text |

---

## Installation

```bash
go get github.com/Vanni-Fan/idmix/golang
```

**Requires:** Go 1.23+

```go
import idmix "github.com/Vanni-Fan/idmix/golang"
```

---

## Quick start

### Full pipeline (IdMix)

```go
package main

import (
    "fmt"
    idmix "github.com/Vanni-Fan/idmix/golang"
)

func main() {
    m, err := idmix.New()
    if err != nil {
        panic(err)
    }

    // Integers + short strings (≤63 bytes)
    s, err := m.Encode(uint16(5), int64(-1), uint32(40), "hello")
    if err != nil {
        panic(err)
    }
    fmt.Println("encoded:", s)

    list, err := m.Decode(s)
    if err != nil {
        panic(err)
    }
    fmt.Println(list[0].(uint16), list[1].(int64), list[2].(uint32), list[3].(string))
}
```

### IDX binary layer only

```go
idx, _ := idmix.NewIdx()
bin, _ := idx.Encode(uint16(5), int64(-1))
out, _ := idx.Decode(bin) // []any; types match encoding input
_ = bin
_ = out
```

### Text obfuscation layer only

```go
// Wrap any binary (e.g. Protobuf output)
protoBytes := []byte{0x08, 0x96, 0x01}
text, _ := idmix.EncodeBytes(protoBytes)
raw, _ := idmix.DecodeString(text)
```

---

## Supported input types

`Idx.Encode` / `IdMix.Encode` accept `...any`. Each element may be:

| Go type | Notes |
|---------|-------|
| `uint8`, `uint16`, `uint32`, `uint64`, `uint` | Unsigned integers; original width preserved |
| `int8`, `int16`, `int32`, `int64`, `int` | Signed integers (`int` stored as `int64`) |
| `string` | UTF-8 text, **1–63 bytes** (byte length, not rune count) |
| `[]byte` | Raw bytes, **1–63 bytes** |

Decode returns `[]any`; use type assertions, e.g. `list[0].(uint16)`.

**Limits:**

- At most **255** objects per encode (`WithMaxObjects`, max 255)
- Each string/byte slice at most **63** bytes
- Empty string or empty `[]byte` is rejected

---

## API reference

### Constants

#### `DefaultAlphabet`

```go
const DefaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
```

Default 62-character alphabet for `RadixCodec`.

---

### Idx — binary layer

#### `type Idx struct`

IDX binary encoder/decoder. Options are applied via `IdxOption` at creation time — **not on IdMix**.

#### `type IdxOption func(*Idx) error`

Idx configuration option type.

#### `func NewIdx(opts ...IdxOption) (*Idx, error)`

Creates an Idx instance. Defaults:

| Option | Default | Range |
|--------|---------|-------|
| `maxObjects` | 255 | 1–255 |
| `maxVariants` | 32 | 1–32 |
| `checkBits` | 2 | 1–2 |

#### `func WithMaxObjects(n int) IdxOption`

Maximum number of objects allowed per encode.

#### `func WithMaxVariants(n int) IdxOption`

Number of variants. `Encode` picks randomly from `[0, maxVariants)`; same input with different variants yields different binary (XOR obfuscation).

#### `func WithCheckBits(n int) IdxOption`

Width of XOR checksum bits in the header (1 or 2).

#### `func (idx *Idx) Encode(values ...any) ([]byte, error)`

Encodes values into an IDX binary block. Uses `variant_id = 0`.

- At least one value required
- Object count must not exceed `maxObjects`
- **1-byte header** for a single object; **2-byte header** for multiple objects (includes count)

#### `func (idx *Idx) EncodeWithVariant(variantID int, values ...any) ([]byte, error)`

Same as `Encode` but with an explicit `variant_id` (0 .. maxVariants-1). For tests and deterministic encoding.

#### `func (idx *Idx) Decode(data []byte) ([]any, error)`

Decodes an IDX binary block into `[]any`. Failures include: data too short, checksum mismatch, invalid variant, malformed objects, etc.

---

### Codec — text layer interface

#### `type Codec interface`

```go
type Codec interface {
    Encode(data []byte) (string, error)
    Decode(s string) ([]byte, error)
}
```

Pluggable binary↔text codec. Implementations may include:

- Custom radix alphabet (`RadixCodec`)
- Standard Base64 (`Base64Codec`)
- AES + Base64, XOR + Base64, etc. (`FuncCodec` or custom types)

#### `func EncodeBytes(data []byte, codec ...Codec) (string, error)`

Package-level: encodes arbitrary binary to text.

- Omits `codec` → default `RadixCodec` (`DefaultAlphabet`)
- Passing `nil` is ignored; default is used

#### `func DecodeString(s string, codec ...Codec) ([]byte, error)`

Package-level: decodes text to binary. Same `codec` rules as `EncodeBytes`.

---

### RadixCodec — default radix codec

#### `type RadixCodec struct`

Base-N codec using a custom alphabet (default idmix text layer).

Strategy: prepend a 2-byte big-endian length prefix, then encode the payload as a big integer in the custom base.

#### `func NewRadixCodec(alphabet string) (*RadixCodec, error)`

Creates a RadixCodec.

- Alphabet must have at least 2 characters
- Characters must be unique

#### `func (rc *RadixCodec) Alphabet() string`

Returns the alphabet string.

#### `func (rc *RadixCodec) Base() int`

Returns the radix (alphabet length).

#### `func (rc *RadixCodec) Encode(data []byte) (string, error)`

Implements `Codec`.

#### `func (rc *RadixCodec) Decode(s string) ([]byte, error)`

Implements `Codec`.

---

### Base64Codec

#### `type Base64Codec struct{}`

Standard Base64 codec implementing `Codec`.

#### `func NewBase64Codec() Base64Codec`

Creates a Base64Codec instance.

---

### FuncCodec — function-based Codec

#### `type FuncCodec struct`

```go
type FuncCodec struct {
    EncodeFn func(data []byte) (string, error)
    DecodeFn func(s string) ([]byte, error)
}
```

`Codec` implemented via functions; useful for composing encryption, XOR, etc.

---

### IdMix — high-level API

#### `type IdMix struct`

Combines `Idx` and `Codec`. Configuration:

- **Idx** (`WithIdx`)
- **Codec** (`WithCodec` or convenience `WithAlphabet`)

#### `type Option func(*IdMix) error`

IdMix configuration option type.

#### `func New(opts ...Option) (*IdMix, error)`

Creates IdMix. Defaults: `NewIdx()` + default `RadixCodec`.

#### `func WithIdx(idx *Idx) Option`

Injects a preconfigured Idx instance (`maxObjects`, etc. set on Idx).

#### `func WithCodec(codec Codec) Option`

Sets the text codec. `codec` must not be `nil`.

#### `func WithAlphabet(alphabet string) Option`

Convenience: creates `RadixCodec` from `alphabet` and sets it as the Codec.

#### `func (m *IdMix) Idx() *Idx`

Returns the embedded Idx.

#### `func (m *IdMix) Codec() Codec`

Returns the current Codec.

#### `func (m *IdMix) Encode(values ...any) (string, error)`

Encoding steps:

1. Pick random `variant_id ∈ [0, maxVariants)`
2. Idx encodes to binary
3. `Codec.Encode` produces text

At least one value; same type/length rules as Idx.

#### `func (m *IdMix) Decode(s string) ([]any, error)`

Decoding steps:

1. `Codec.Decode` → binary
2. `Idx.Decode` → `[]any`

#### `func (m *IdMix) EncodeWithVariant(variantID int, values ...any) (string, error)`

Deterministic encode (fixed `variant_id`). Mainly for tests and cross-language vector generation.

---

## Configuration examples

### Custom Idx options

```go
idx, err := idmix.NewIdx(
    idmix.WithMaxObjects(100),
    idmix.WithMaxVariants(16),
    idmix.WithCheckBits(2),
)
if err != nil {
    panic(err)
}

m, err := idmix.New(idmix.WithIdx(idx))
```

### Custom alphabet

```go
m, err := idmix.New(idmix.WithAlphabet("abcd"))
// Equivalent to:
radix, _ := idmix.NewRadixCodec("abcd")
m, err := idmix.New(idmix.WithCodec(radix))
```

### Base64 as text layer

```go
m, err := idmix.New(idmix.WithCodec(idmix.NewBase64Codec()))
s, _ := m.Encode(uint32(1001), uint8(3))
```

### Custom Codec (XOR + Radix)

```go
inner, _ := idmix.NewRadixCodec(idmix.DefaultAlphabet)
const xorKey byte = 0x5A

xorCodec := idmix.FuncCodec{
    EncodeFn: func(data []byte) (string, error) {
        buf := make([]byte, len(data))
        for i, b := range data {
            buf[i] = b ^ xorKey
        }
        return inner.Encode(buf)
    },
    DecodeFn: func(s string) ([]byte, error) {
        buf, err := inner.Decode(s)
        if err != nil {
            return nil, err
        }
        for i := range buf {
            buf[i] ^= xorKey
        }
        return buf, nil
    },
}

m, _ := idmix.New(idmix.WithCodec(xorCodec))
```

### Protobuf + idmix text layer (without Idx)

```go
import "google.golang.org/protobuf/encoding/protowire"

// Assume protoBytes is already serialized
var protoBytes []byte

text, _ := idmix.EncodeBytes(protoBytes)           // default Radix
text, _ = idmix.EncodeBytes(protoBytes, idmix.NewBase64Codec())

restored, _ := idmix.DecodeString(text)
```

---

## Type assertions and uint64

After decode, assert to the types used at encode time:

```go
list, _ := m.Decode(s)

u16 := list[0].(uint16)
i64 := list[1].(int64)
str := list[2].(string)

// uint64 values > MaxInt64 are stored as int64 bit patterns; decode as uint64
u64 := list[3].(uint64)
```

---

## Error handling

Common errors:

| Scenario | Typical message |
|----------|-----------------|
| Empty input | `at least one value is required` |
| Unsupported type | `unsupported type %T` |
| String too long | `string length 64 exceeds max 63` |
| Empty string | `empty string is not allowed` |
| Too many objects | `too many objects: N (max M)` |
| Invalid config | `maxObjects must be between 1 and 255`, etc. |
| Checksum failure | `checksum mismatch` |
| Invalid variant | `invalid variant_id N (max M)` |
| Invalid alphabet | `alphabet contains duplicate character` |
| Nil codec | `codec cannot be nil` |

Always check `err != nil` in production code; tests can assert specific messages for boundary cases.

---

## Testing

```bash
cd golang

# All tests
go test ./...

# Verbose
go test -v ./...

# Single-object 1-byte header
go test -v -run TestSingleObject

# String 63/64-byte limits and Idx options
go test -v -run 'TestStringLengthBoundaries|TestIdxMax'

# Cross-language vectors
go test -v -run CrossLanguage

# Regenerate testdata (after algorithm changes)
# Windows PowerShell:
$env:GENERATE_VECTORS=1; go test -run TestGenerateCrossLanguageVectors

# Linux/macOS:
GENERATE_VECTORS=1 go test -run TestGenerateCrossLanguageVectors

# Binary size vs MsgPack/CBOR/Protobuf
go test -v -run TestCompareSerializationFormats
```

---

## File layout

```
golang/
├── README.md           # Chinese documentation
├── README_en.md        # English documentation (this file)
├── go.mod
├── idmix.go            # IdMix entry; package EncodeBytes/DecodeString
├── idx_codec.go        # Idx binary codec
├── codec.go            # Codec interface, Base64Codec, FuncCodec
├── alphabet.go         # RadixCodec
├── number.go           # any → internal representation
├── idmix_test.go       # End-to-end and demo tests
├── idx_test.go         # Idx options and string boundary tests
├── cross_language_test.go
├── vectors_test.go
└── benchmark_*.go      # Benchmarks vs sqids, MsgPack, etc.
```

---

## Other language implementations

Go is the **reference implementation** in this repository. Cross-language test vectors:

`../testdata/cross_language_vectors.json`

After changing the IDX algorithm or constants, run `GENERATE_VECTORS=1 go test -run TestGenerateCrossLanguageVectors` in `golang/`, then sync other language ports.

---

## Related links

- [IDX v1.2 specification](../arithmetic.md)
- [Project overview README](../README.md)
- [Packaging guide](../PACKAGING.md)
- [pkg.go.dev](https://pkg.go.dev/github.com/Vanni-Fan/idmix/golang)
