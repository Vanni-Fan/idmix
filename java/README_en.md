# idmix — Java Implementation

> **中文:** [README.md](README.md)

Java implementation providing **IDX binary encoding** and a **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

```
Maven: io.github.vanni-fan:idmix
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (high-level API)                                     │
│  Encode / Decode: integers / short strings → text           │
│  Internally: Idx encode → ICodec.encode                     │
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

```xml
<dependency>
    <groupId>io.github.vanni-fan</groupId>
    <artifactId>idmix</artifactId>
    <version>0.4.0</version>
</dependency>
```

**Requires:** Java 17+

```java
import io.github.vannifan.idmix.*;
```

## Quick start

### Full pipeline (IdMix)

```java
IdMix m = IdMix.create();

// Integers + short strings (≤63 bytes)
String s = m.encode(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40), "hello");
Object[] list = m.decode(s);
```

### IDX binary layer only

```java
Idx idx = Idx.create();
byte[] bin = idx.encodeWithVariant(0, TypedValue.u16(5), TypedValue.i64(-1));
Object[] out = idx.decode(bin);
```

### Text obfuscation layer only

```java
byte[] protoBytes = new byte[] {0x08, (byte) 0x96, 0x01};
String text = Codec.encodeBytes(protoBytes);
byte[] raw = Codec.decodeString(text);
```

## Supported input types

`Idx.encode` / `IdMix.encode` accept `Object...`. Each element may be:

| Java type | Notes |
|-----------|-------|
| `Byte`, `Short`, `Integer`, `Long` | Signed integer wrapper types |
| `TypedValue` | Typed integer (recommended for unsigned types) |
| `String` | UTF-8 text, **1–63 bytes** (byte length, not char count) |
| `byte[]` | Raw bytes, **1–63 bytes** |

**Limits:**

- At most **255** objects per encode
- Each string/byte slice at most **63** bytes
- Empty string or empty `byte[]` is rejected

## API reference

### Idx

```java
Idx idx = Idx.create();
Idx idx2 = Idx.create(new Idx.IdxBuilder() {{
    maxObjects = 100;
    maxVariants = 16;
    checkBits = 2;
}});

byte[] bin = idx.encode(TypedValue.u16(5), "hello");
byte[] bin2 = idx.encodeWithVariant(0, TypedValue.u16(5));
Object[] out = idx.decode(bin);
```

### ICodec / Codec

```java
Codec.encodeBytes(data);
Codec.decodeString(text);
Codec.encodeBytes(data, Codec.Base64Codec.INSTANCE);
```

### IdMix

```java
IdMix m = IdMix.create();
IdMix m2 = IdMix.create(new IdMix.IdMixBuilder().withAlphabet("abcd"));
IdMix m3 = IdMix.create(new IdMix.IdMixBuilder().withCodec(Codec.Base64Codec.INSTANCE));

String s = m.encode(TypedValue.u32(1001), TypedValue.u8(3));
String s2 = m.encodeWithVariant(0, TypedValue.u32(1001));
Object[] list = m.decode(s);
```

## Testing

```bash
cd java
mvn test -q
```

## Related links

- [IDX v1.2 specification](../arithmetic.md)
- [Go reference implementation](../golang/README_en.md)
- [Cross-language test vectors](../testdata/cross_language_vectors.json)
