# vanni-idmix — PHP Implementation

> **中文:** [README.md](README.md)

PHP reference implementation providing **IDX v1.2 binary encoding** and a **pluggable idmix text layer**. See [arithmetic.md](../arithmetic.md) in the repository root for the protocol specification.

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
| **Idx** | Compact self-describing binary only |
| **Codec + encodeBytes / decodeString** | Wrap Protobuf/CBOR/any binary as text |
| **IdMix** | access_key, short IDs: integer/string sequences → obfuscated text |

## Installation

```bash
composer require vanni/idmix
```

**Requires:** PHP 8.1+, `ext-bcmath`, `ext-mbstring`

## Quick start

### Full pipeline (IdMix)

```php
use Vanni\Idmix\IdMix;
use Vanni\Idmix\TypedValue;

$m = IdMix::new();

// Integers + short strings (≤63 bytes)
$s = $m->encode(TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40), 'hello');
$values = $m->decode($s);
```

> Decoded integers are `TypedValue` (bcmath decimal strings, full uint64 range); strings are `string`.

### IDX binary layer only

```php
use Vanni\Idmix\Idx;
use Vanni\Idmix\TypedValue;

$idx = Idx::new();
$data = $idx->encode(TypedValue::u16(5), TypedValue::i64(-1));
$out = $idx->decode($data);
```

### Text obfuscation layer only

```php
use function Vanni\Idmix\encodeBytes;
use function Vanni\Idmix\decodeString;

$proto = "\x08\x96\x01";
$text = encodeBytes($proto);
$raw = decodeString($text);
```

## Supported input types

`Idx.encode` / `IdMix.encode` accept variadic arguments. Each element may be:

| PHP type | Description |
|----------|-------------|
| `int` | Integer (stored as int64) |
| `TypedValue` | Bit-width–aware integers aligned with Go types |
| `string` | Raw byte string, **1~63 bytes** (byte count) |
| `Bytes` | Raw bytes wrapper, **1~63 bytes** |

**Limits:**

- Up to **255** objects per encode (`Idx::new($maxObjects)` configurable)
- Max **63** bytes per string/byte segment
- Empty string or empty `Bytes` is rejected

## Module layout

| File | Role |
|------|------|
| `Idx.php` | `Idx` binary codec |
| `Codec.php` | `Codec` interface, `FuncCodec`, `encodeBytes` / `decodeString` |
| `Base64Codec.php` | Standard Base64 text layer |
| `RadixCodec.php` | Default radix codec |
| `Number.php` | Type normalization |
| `Idmix.php` | `IdMix` high-level API |

## Configuration examples

### Custom Idx options

```php
$idx = Idx::new(maxObjects: 100, maxVariants: 16, checkBits: 2);
$m = IdMix::create($idx);
```

### Custom alphabet

```php
$m = IdMix::withAlphabet('abcd');
```

### Base64 as text layer

```php
use Vanni\Idmix\Base64Codec;
use Vanni\Idmix\IdMix;

$m = IdMix::create(null, Base64Codec::new());
```

## Tests

```bash
cd php
php tests/run_tests.php
```

Cross-language vectors live in `../testdata/cross_language_vectors.json`.

## Links

- [Protocol spec](../arithmetic.md)
- [Go reference](../golang/README_en.md)
- [GitHub repository](https://github.com/Vanni-Fan/idmix)
