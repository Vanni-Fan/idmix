# idmix — C# Implementation

> **中文:** [README.md](README.md)

C# implementation providing **IDX binary encoding** and a **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

```
NuGet: Vanni.Idmix
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (high-level API)                                     │
│  Encode / Decode: integers / short strings → text           │
│  Internally: Idx encode → ICodec.Encode                     │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  ICodec (interface) │
│  Binary codec   │                 │  binary ↔ text      │
└─────────────────┘                 └─────────────────────┘
```

## Installation

```bash
dotnet add package Vanni.Idmix
```

**Requires:** .NET 8.0+

```csharp
using Vanni.Idmix;
```

## Quick start

### Full pipeline (IdMix)

```csharp
var m = IdMix.Create();

// Integers + short strings (≤63 bytes)
var s = m.Encode((ushort)5, -1L, 40u, "hello");
var list = m.Decode(s);
// list[0] = (ushort)5, list[1] = -1L, list[2] = 40u, list[3] = "hello"
```

### IDX binary layer only

```csharp
var idx = Idx.Create();
var bin = idx.EncodeWithVariant(0, (ushort)5, -1L);
var out_ = idx.Decode(bin);
```

### Text obfuscation layer only

```csharp
var protoBytes = new byte[] { 0x08, 0x96, 0x01 };
var text = Codec.EncodeBytes(protoBytes);
var raw = Codec.DecodeString(text);
```

## Supported input types

`Idx.Encode` / `IdMix.Encode` accept `params object[]`. Each element may be:

| C# type | Notes |
|---------|-------|
| `byte`, `ushort`, `uint`, `ulong` | Unsigned integers; original width preserved |
| `sbyte`, `short`, `int`, `long` | Signed integers |
| `string` | UTF-8 text, **1–63 bytes** (byte length, not char count) |
| `byte[]` | Raw bytes, **1–63 bytes** |
| `TypedValue` | Typed integer wrapper (convenience) |

**Limits:**

- At most **255** objects per encode
- Each string/byte slice at most **63** bytes
- Empty string or empty `byte[]` is rejected

## API reference

### Idx

```csharp
var idx = Idx.Create();
var idx2 = Idx.Create(b => {
    b.MaxObjects = 100;
    b.MaxVariants = 16;
    b.CheckBits = 2;
});

byte[] bin = idx.Encode((ushort)5, "hello");
byte[] bin2 = idx.EncodeWithVariant(0, (ushort)5);
object[] out_ = idx.Decode(bin);
```

Defaults: `maxObjects=255`, `maxVariants=32`, `checkBits=2`.

### ICodec / Codec

```csharp
// Default RadixCodec
Codec.EncodeBytes(data);
Codec.DecodeString(text);

// Base64
var b64 = Base64Codec.Instance;
Codec.EncodeBytes(data, b64);

// Custom RadixCodec
var radix = new RadixCodec(IdMix.DefaultAlphabet);
```

### IdMix

```csharp
var m = IdMix.Create();
var m2 = IdMix.Create(b => b.WithAlphabet("abcd"));
var m3 = IdMix.Create(b => b.WithCodec(Base64Codec.Instance));
var m4 = IdMix.Create(b => b.WithIdx(Idx.Create()));

string s = m.Encode((ushort)5, "hello");
string s2 = m.EncodeWithVariant(0, (ushort)5);  // deterministic encode
object[] list = m.Decode(s);

Idx idx = m.Idx;
ICodec codec = m.Codec;
```

## Testing

```bash
cd csharp/tests/Vanni.Idmix.Tests
dotnet test
```

## Related links

- [IDX v1.2 specification](../arithmetic.md)
- [Go reference implementation](../golang/README_en.md)
- [Cross-language test vectors](../testdata/cross_language_vectors.json)
