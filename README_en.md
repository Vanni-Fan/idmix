# idmix — IDX v1.2 Self-Describing Serialization + Text Obfuscation

> **中文:** [README.md](README.md)

Encodes multiple **typed integers** or **short strings** (≤63 bytes) into compact text strings, suitable for access keys, short IDs, tokens, and similar use cases. **C, C++, C#, Go, Java, JavaScript, PHP, Python, and Rust** implementations are interoperable and follow the [IDX v1.2 specification](arithmetic.md).

## Language Documentation

| Language | 中文 | English |
| --- | --- | --- |
| Go (reference) | [golang/README.md](golang/README.md) | [golang/README_en.md](golang/README_en.md) |
| Rust | [rust/lib/README.md](rust/lib/README.md) | [rust/lib/README_en.md](rust/lib/README_en.md) |
| Python | [python/README.md](python/README.md) | [python/README_en.md](python/README_en.md) |
| JavaScript | [javascript/README.md](javascript/README.md) | [javascript/README_en.md](javascript/README_en.md) |
| Java | [java/README.md](java/README.md) | [java/README_en.md](java/README_en.md) |
| C# | [csharp/README.md](csharp/README.md) | [csharp/README_en.md](csharp/README_en.md) |
| PHP | [php/README.md](php/README.md) | [php/README_en.md](php/README_en.md) |
| C++ | [cpp/README.md](cpp/README.md) | [cpp/README_en.md](cpp/README_en.md) |
| C | [c/README.md](c/README.md) | [c/README_en.md](c/README_en.md) |

**Two layers usable independently**:

| Layer | API (Go) | Role |
| --- | --- | --- |
| **IDX** | `NewIdx()` / `Idx.Encode` / `Idx.Decode` | Integers / short strings ↔ self-describing binary |
| **Codec** | `EncodeBytes` / `DecodeString` + `Codec` interface | Arbitrary binary ↔ text (pluggable) |
| **Combined** | `IdMix.Encode` / `Decode` | IDX + Codec |

The text layer is **pluggable** via the **`Codec` interface**: default `RadixCodec` (custom alphabet), or `Base64Codec`, or `FuncCodec` wrapping AES/XOR and similar transforms.

You can also pass **Protobuf / CBOR / MessagePack** binary directly through `EncodeBytes` to produce obfuscated text, as an alternative to the traditional `Protobuf + AES + Base64` pipeline.

## Use Cases

- **Type self-description**: Each integer carries its original type (`uint8`–`uint64` / `int8`–`int64`, 8 otypes total); decoding does not depend on an external schema
- **Short strings**: Extended mode supports UTF-8 / byte strings up to 63 bytes
- **Extreme compression**: Positive values 0–15 and negative values -1 to -15 use only 1 byte; **single-object headers are only 1 byte**
- **32-variant polymorphism**: The same data can produce up to 32 different encodings (variant XOR obfuscation)
- **Lightweight self-check**: 2-bit global XOR checksum; roughly 75% of random tampering is rejected immediately
- **Text obfuscation (idmix layer)**: Default 62-character alphabet; can wrap arbitrary binary on its own

## Algorithm Overview (IDX v1.2 + idmix)

```
integers / short strings → [IDX binary layer] → [idmix text layer] → string
arbitrary binary         → [idmix text layer] → string          (skip IDX)
```

### 1. IDX Binary Layer

Binary block = **1- or 2-byte header** + **data object sequence**.

**Header (single object, 1 byte)**:

| Field | Width | Meaning |
| --- | --- | --- |
| bit[1:0] | 2 | `check` checksum bits |
| bit[6:2] | 5 | `variant_id` (0–31) |
| bit7 | 1 | `0` = single object |

**Header (multiple objects, 2 bytes)**: byte 0 bit7=1; byte 1 is count (2–255).

**Data objects**:

- **Embedded mode (1 byte)**: small integers in [-15, 15]
- **Extended number (1+ bytes)**: bit6=0; signed types use two's-complement little-endian payload (no separate sign bit)
- **Extended string (1+ bytes)**: bit6=1; bit5–0 is length (1–63), followed by raw bytes

**Variant obfuscation**: `mask = (variant_id × 0x9D + 0x37) & 0xFF`, XOR applied byte-by-byte over the object region (header excluded).

Spec example `uint16(5), int64(-1), uint32(40)` (variant=0, three objects) binary block:

```
80 03 22 47 B5 1F
```

See [arithmetic.md](arithmetic.md) for the full protocol.

### 2. idmix Text Layer

A 2-byte big-endian length prefix is prepended to the binary block; the whole payload is converted to a string using a **custom radix** (default 62-character alphabet). Can be used standalone on arbitrary binary, not limited to IDX output.

## Comparison with [Sqids](https://sqids.org/)

[Sqids](https://sqids.org/) is a popular short-ID library that supports only **non-negative integer sequences**. idmix extends this model for structured access_key scenarios.


| Feature | Sqids | idmix |
| --- | --- | --- |
| Input types | Non-negative integer sequence (`uint64`) | Typed integer sequence (`uint8`–`uint64` / `int8`–`int64`) |
| Negative numbers | Not supported | Supported (embedded / extended mode) |
| Type self-description | No; decoding needs external schema | Yes; each value carries its original type |
| Polymorphic encoding | No; same input → same output | Yes; 32 variant strings by default |
| Self-check | No | Yes; 2-bit global XOR (~75% rejection of random strings) |
| Blocklist | Supported; filters sensitive words | **Not supported** (see note below) |
| Minimum length | Supports `min_length` | Not supported (aims for shortest natural encoding) |
| Custom alphabet | Supported (default 64 chars) | Supported (default 62 chars) |
| Max elements per encode | No hard limit | Default 255 (configurable) |


**About blocklists**: idmix does not provide Sqids-style blocklist filtering. idmix has built-in **32-variant polymorphism**—each encode randomly picks a `variant_id`, so string forms are naturally dispersed and the chance of a specific sensitive word is very low; if unsatisfied, call `Encode` again for another variant. Combined with the 2-bit checksum, random guessing is also less effective.

Below is an **encoding length and performance** comparison of **Go** and **Rust** implementations vs Sqids (single-threaded on local machine, 20,000 samples; idmix lengths include 32 variants, reported as min–max). For extreme-value cases Sqids only supports non-negative integers, so `int64_max` / `uint64_max` are passed as `uint64`; negative cases `int32_min` / `int64_min` / `mixed_extremes` are idmix-only capabilities (see end of section).

### Encoding Length


| Scenario | sqids | idmix |
| --- | --- | --- |
| [1, 2, 3] | 6 | 8 |
| Single value [42] | 2 | 6 |
| Single value uint32 [2000000000] | 7 | 10 |
| AccessKey [1001, 1690000000, 3] | 12 | 16 |
| Small integers [0..9] | 20 | 17 |
| **uint32_max** | 7 | 10 |
| **int64_max** | 12 | 16 |
| **uint64_max** | 12 | 16 |
| **Extreme triple [u32max, i64max, u64max]** | 31 | 35 |


Sqids produces shorter strings for pure non-negative small integers; idmix is slightly longer due to header, type tags, and variant overhead, but can be shorter when small integers are dense (embedded mode: 1 byte/value). At single-field extremes idmix is still a bit longer than Sqids (e.g. `uint32_max`: 10 vs 7), but the gap narrows for extreme triple sequences (35 vs 31).

### Encoding Performance (idmix / sqids ratio; >1 means idmix is faster)


| Scenario | Go encode | Rust encode | Go decode | Rust decode |
| --- | --- | --- | --- | --- |
| [1, 2, 3] | **47.2×** | 12.1× | **8.0×** | 5.3× |
| AccessKey [1001, 1690000000, 3] | **20.6×** | 16.2× | **2.4×** | 4.2× |
| uint32 [2000000000] | **16.3×** | 12.7× | **1.9×** | 4.0× |
| **uint32_max** | **14.3×** | 12.8× | **2.6×** | 4.0× |
| **int64_max** | **10.8×** | 17.3× | **1.3×** | 3.0× |
| **uint64_max** | **16.5×** | 16.8× | **1.4×** | 2.9× |
| **Extreme triple [u32max, i64max, u64max]** | **7.8×** | 10.2× | **2.3×** | 2.7× |


idmix encode/decode is dominated by bit operations and is significantly faster than Sqids; Go is overall faster than Rust (Rust text layer uses `num-bigint`). At extreme values idmix encoding advantage remains clear (7.8×–47×); decode advantage narrows as values grow (large-integer radix decode ~1.3×–2.6×, still faster than or close to Sqids).

### idmix-Only Capabilities (not supported by Sqids)

- Typed: `uint16(5), int64(-1), uint32(40)` → 9 characters
- Negative and signed extremes: `int32_min` → 10 characters, `int64_min` → 16 characters
- `mixed_extremes` with negatives (five-field extremes) → 54 characters
- Random variants: multiple encodes of the same input produce different strings

Run comparison tests:

```bash
# Go — length
cd golang && go test -v -run TestCompareSqids

# Go — performance (includes extremes)
cd golang && go test -v -run TestCompareSqidsPerformance

# Rust
cd rust/lib && cargo test --test benchmark_sqids -- --nocapture
```

## Comparison with MessagePack / CBOR / Protobuf (IDX Binary Layer)

The following compares **IDX binary byte counts** with MessagePack, CBOR, and Protobuf (no schema, per-field `otype`+`val`), **excluding base64 or idmix text layer**, for a fair comparison with MsgPack/CBOR/Protobuf.

### Encoding Length (bytes)

| Scenario | IDX | MsgPack | CBOR | Protobuf |
| --- | --- | --- | --- | --- |
| spec_example | 6 | 46 | 23 | 27 |
| uint32_max | 6 | 16 | 12 | 10 |
| int32_min | 6 | 16 | 12 | 15 |
| int64_min | 10 | 16 | 16 | 15 |
| int64_max | 10 | 16 | 16 | 14 |
| uint64_max | 10 | 16 | 8 | 15 |
| mixed_extremes | 39 | 76 | 60 | 69 |
| access_key | 11 | 46 | 28 | 23 |
| embedded_small | 6 | 61 | 29 | 42 |
| string_example | 16 | 16 | 8 | 6 |

`mixed_extremes` contains five single-field extreme values; `string_example` = `"hello"` + `uint16(5)` + `"世界"` (variant=0, measured in Go reference implementation).

IDX binary size is typically **much smaller** than MsgPack for typed integers; inline short strings are close to CBOR.

### Encode/Decode Performance (IDX binary layer, relative ratios)

Go reference implementation, single-threaded, 20,000 samples per item. Ratio = **IDX ops/s ÷ other ops/s**; **>1 means IDX is faster**.

**Encode**

| Scenario | vs MsgPack | vs CBOR | vs Protobuf |
| --- | --- | --- | --- |
| spec_example | 2.2× | 1.3× | 0.8× |
| access_key | 3.2× | 2.2× | 1.1× |
| embedded_small | 3.7× | 1.9× | 1.8× |
| mixed_extremes | 1.7× | 0.7× | 0.9× |

**Decode**

| Scenario | vs MsgPack | vs CBOR | vs Protobuf |
| --- | --- | --- | --- |
| spec_example | 5.9× | 4.3× | 0.8× |
| access_key | 4.9× | 3.1× | 0.6× |
| embedded_small | 7.1× | 5.7× | 0.8× |
| mixed_extremes | 3.0× | 2.7× | 0.5× |

**Summary**: IDX binary encode/decode is significantly faster than MsgPack/CBOR for small-to-medium integers; add the idmix text layer when URL-readable strings are needed.

Run comparison tests:

```bash
cd golang && go test -v -run TestCompareSerializationFormats
cd golang && go test -v -run TestCompareSerializationFormatsPerformance
```

## Language Implementations and uint64 Support

All implementations cover **uint8–uint64** (and corresponding signed types int8–int64, 8 otypes total); extended-mode payloads are unsigned little-endian bytes with `otype` indicating type; signed types strip redundant sign-extension bytes when encoding to save space.


| Language | uint64 / large integers | Special requirements |
| --- | --- | --- |
| **Go** | Native `uint64`; internally passed as bit patterns via `int64`/`uint64` | None |
| **Rust** | Native `u64` / `i64` | Text-layer radix encoding depends on `num-bigint` crate |
| **Python** | Native arbitrary-precision `int` | None |
| **JavaScript** | `BigInt` (`typed_value.js`) | Node.js 10.4+; int64/uint64 in cross-language JSON vectors use **strings** to avoid `JSON.parse` precision loss |
| **Java** | `long` API + unsigned bit patterns (`Long.compareUnsigned`) | `TypedValue.u64()` accepts decimal or `0x` hex strings |
| **C#** | `ulong` / `long` + unsigned bit patterns | `TypedValue.U64(ulong)` |
| **VB.NET** | Same as C# | Same as C# |
| **PHP** | **ext-bcmath** decimal strings (`IntMath`) | **Must enable** `bcmath` extension (`composer.json` declares `ext-bcmath`); 32- and 64-bit PHP both handle full uint64 |
| **C / C++** | `uint64_t` / `int64_t` | In C API, `idmix_value_t.val` is `int64_t`; large uint64 values passed as bit patterns |


### Extended Mode Notes

Core ideas from [arithmetic.md](arithmetic.md) section 2.2:

1. **Extended number** (bit6=0): signed types use two's-complement little-endian payload; unsigned types use unsigned little-endian; **bit6 is no longer a sign bit**.
2. **Extended string** (bit6=1): bit5–0 is length (1–63), followed by raw bytes.
3. Storage width `sw` is determined only by value magnitude, not otype bit width.
4. **Embedded mode** (bit7=0): covers **[-15, 15]** (+16 uses 2-byte extended form).
5. **Single-object header** is only 1 byte; multi-object adds a count byte.

### Cross-Language Interoperability Tests

Shared vector file: [testdata/cross_language_vectors.json](testdata/cross_language_vectors.json) (generated by the Go reference implementation).

```bash
# Regenerate vectors (after changing constants or algorithm)
cd golang && GENERATE_VECTORS=1 go test -run TestGenerateCrossLanguageVectors

# Per-language verification
cd golang && go test -run CrossLanguage
cd php && php tests/run_tests.php
cd python && python -m unittest discover -s tests -v
cd javascript && npm test
cd rust/lib && cargo test --test extreme_cross_language
cd java && mvn test
cd csharp/tests/Vanni.Idmix.Tests && dotnet test
```

## Quick Start

### Go

```bash
go get github.com/Vanni-Fan/idmix/golang
```

```go
package main

import (
    "fmt"
    idmix "github.com/Vanni-Fan/idmix/golang"
)

func main() {
    // Combined: IDX + Codec
    m, _ := idmix.New()
    str, _ := m.Encode(uint16(5), int64(-1), uint32(40), "hello")
    list, _ := m.Decode(str)
    fmt.Println(str, list[0].(uint16), list[1].(int64), list[2].(uint32), list[3].(string))

    // IDX binary layer only (configure maxObjects here)
    idx, _ := idmix.NewIdx(idmix.WithMaxObjects(100))
    bin, _ := idx.Encode(uint16(5), int64(-1))
    out, _ := idx.Decode(bin)

    // Text layer only (package-level functions; Codec optional)
    protoBytes := []byte{0x08, 0x96, 0x01}
    obfuscated, _ := idmix.EncodeBytes(protoBytes)
    obfuscated, _ = idmix.EncodeBytes(protoBytes, idmix.NewBase64Codec())
    raw, _ := idmix.DecodeString(obfuscated, idmix.NewBase64Codec())

    // Custom Codec: WithCodec(radix) or WithAlphabet("abcd")
    radix, _ := idmix.NewRadixCodec(idmix.DefaultAlphabet)
    custom, _ := idmix.New(idmix.WithIdx(idx), idmix.WithCodec(radix))
    _, _ = custom, out
    _ = raw
}
```

### PHP

Requires PHP 8.1+ and **ext-bcmath**, **ext-mbstring** extensions (`composer.json` declares them). Integers are handled as decimal strings via bcmath; both 32- and 64-bit PHP handle the full uint64 range.

```bash
composer require vanni/idmix
```

```php
<?php
use Vanni\Idmix\IdMix;
use Vanni\Idmix\TypedValue;

$m = IdMix::new();
$str = $m->encode(TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40));
$out = $m->decode($str);
// $out[0]->val === 5, $out[1]->val === -1, $out[2]->val === 40
```

### Rust

```bash
cargo add idmix@0.4.0
```

```rust
use idmix::{IdMix, Value};

fn main() {
    let m = IdMix::new().unwrap();
    let values = [Value::U16(5), Value::I64(-1), Value::U32(40)];
    let encoded = m.encode(&values).unwrap();
    let decoded = m.decode(&encoded).unwrap();
    println!("{encoded:?} => {decoded:?}");
}
```

### Python

```bash
pip install vanni-idmix
```

```python
from idmix import IdMix, u16, i64, u32

m = IdMix.new()
s = m.encode(u16(5), i64(-1), u32(40))
out = m.decode(s)
# out[0].val == 5, out[1].val == -1, out[2].val == 40
```

### JavaScript (Node.js)

```bash
npm install @vanni.fan/idmix
```

```javascript
import { IdMix } from '@vanni.fan/idmix';

const m = IdMix.new();
// otype: 1=uint16, 7=int64, 2=uint32
const s = m.encode({ otype: 1, val: 5 }, { otype: 7, val: -1 }, { otype: 2, val: 40 });
const out = m.decode(s);
```

### Java

Maven dependency ([Maven Central](https://central.sonatype.com/artifact/io.github.vanni-fan/idmix/0.4.0)):

```xml
<dependency>
    <groupId>io.github.vanni-fan</groupId>
    <artifactId>idmix</artifactId>
    <version>0.4.0</version>
</dependency>
```

```java
import io.github.vannifan.idmix.IdMix;
import io.github.vannifan.idmix.TypedValue;

IdMix m = IdMix.newDefault();
String s = m.encode(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40));
var out = m.decode(s);
```

### C#

```bash
dotnet add package Vanni.Idmix
```

```csharp
using Vanni.Idmix;

var m = IdMix.NewDefault();
var s = m.Encode(TypedValue.U16(5), TypedValue.I64(-1), TypedValue.U32(40));
var outList = m.Decode(s);
```

### [VB.NET](http://VB.NET)

```bash
dotnet add package Vanni.Idmix.Vb
```

```vb
Imports Vanni.Idmix

Dim m = IdMix.NewDefault()
Dim s = m.Encode(TypedValue.U16(5), TypedValue.I64(-1), TypedValue.U32(40))
Dim outList = m.Decode(s)
```

### C++

C/C++ have no unified official package registry like npm or crates.io; no separate "publish" step is needed. Users integrate via **FetchContent** in their own CMake project (the snippet below goes in **your project**, not maintained in this repo):

```cmake
include(FetchContent)
FetchContent_Declare(idmix
  GIT_REPOSITORY https://github.com/Vanni-Fan/idmix.git
  GIT_TAG v0.4.0
  SOURCE_SUBDIR cpp
)
FetchContent_MakeAvailable(idmix)
target_link_libraries(your_app PRIVATE idmix::idmix)
```

Or clone and build locally:

```bash
git clone https://github.com/Vanni-Fan/idmix.git
cd idmix/cpp && cmake -B build && cmake --build build
```

```cpp
#include "idmix/idmix.hpp"
using namespace idmix;

IdMix m = IdMix::newDefault();
auto s = m.encode({TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)});
auto out = m.decode(s);
```

### C

Same FetchContent integration (`SOURCE_SUBDIR` set to `c`, link `idmix::c`):

```cmake
include(FetchContent)
FetchContent_Declare(idmix_c
  GIT_REPOSITORY https://github.com/Vanni-Fan/idmix.git
  GIT_TAG v0.4.0
  SOURCE_SUBDIR c
)
FetchContent_MakeAvailable(idmix_c)
target_link_libraries(your_app PRIVATE idmix::c)
```

Or build locally:

```bash
git clone https://github.com/Vanni-Fan/idmix.git
cd idmix/c && cmake -B build && cmake --build build
```

```c
#include "idmix.h"

idmix_ctx_t* ctx = idmix_create(NULL);
idmix_value_t vals[] = {
    {IDMIX_OTYPE_UINT16, 5},
    {IDMIX_OTYPE_INT64, -1},
    {IDMIX_OTYPE_UINT32, 40},
};
char* s = NULL;
idmix_encode(ctx, vals, 3, &s);
idmix_value_t* out = NULL;
size_t n = 0;
idmix_decode(ctx, s, &out, &n);
idmix_free_string(s);
idmix_free_values(out);
idmix_destroy(ctx);
```

## Custom Alphabet

```go
// Go
m, _ := idmix.New(idmix.WithAlphabet("abcd"))
```

```php
// PHP
$m = IdMix::withAlphabet('一二三四五六七八九十');
```

```rust
// Rust
let m = IdMix::builder().alphabet("abcd").build().unwrap();
```

```python
# Python
m = IdMix.new("abcd")
```

```javascript
// JavaScript
const m = IdMix.new('abcd');
```

```java
// Java
IdMix m = new IdMix("abcd");
```

```csharp
// C#
var m = new IdMix("abcd");
```

```vb
' VB.NET
Dim m As New IdMix("abcd")
```

```cpp
// C++
IdMix m("abcd");
```

## Project Structure

```
idmix/
├── README.md        # Project overview (Chinese)
├── README_en.md     # Project overview (English)
├── arithmetic.md    # IDX v1.2 full spec + idmix text layer
├── PACKAGING.md     # Per-language packaging and install guide
├── golang/          # Go reference implementation (README / README_en)
├── rust/lib/        # Rust crate
├── php/             # PHP (Composer: vanni/idmix)
├── python/          # Python (pip: vanni-idmix)
├── javascript/      # JavaScript (npm: @vanni.fan/idmix)
├── java/            # Java (Maven: io.github.vanni-fan:idmix)
├── csharp/          # C# (NuGet: Vanni.Idmix)
├── vb/              # VB.NET (NuGet: Vanni.Idmix.Vb)
├── cpp/             # C++ library
├── c/               # C library
└── testdata/        # Cross-language test vectors cross_language_vectors.json
```

## Running Tests (verbose output)

```bash
# Go
cd golang && go test -v ./...

# Rust
cd rust/lib && cargo test -- --nocapture

# Python
cd python && python -m unittest discover -s tests -v

# JavaScript
cd javascript && npm test

# Java
cd java && mvn test

# C#
cd csharp/tests/Vanni.Idmix.Tests && dotnet test

# VB.NET
dotnet build vb/src/Vanni.Idmix.Vb/Vanni.Idmix.Vb.vbproj

# C++
cd cpp && cmake -B build && cmake --build build && build/Debug/idmix_test

# C
cd c && cmake -B build && cmake --build build && build/Debug/idmix_c_test
```

## Package Install (no git clone required)

Current version **0.4.0** (IDX v1.2). Install per language via package managers; see [PACKAGING.md](PACKAGING.md) for release details.

| Language | Install command | Registry |
| --- | --- | --- |
| Go | `go get github.com/Vanni-Fan/idmix/golang` | [pkg.go.dev](https://pkg.go.dev/github.com/Vanni-Fan/idmix/golang) |
| PHP | `composer require vanni/idmix` | [Packagist](https://packagist.org/packages/vanni/idmix) |
| Rust | `cargo add idmix@0.4.0` | [crates.io](https://crates.io/crates/idmix) |
| Python | `pip install vanni-idmix` | [PyPI](https://pypi.org/project/vanni-idmix/) |
| JavaScript | `npm install @vanni.fan/idmix` | [npm](https://www.npmjs.com/package/@vanni.fan/idmix) |
| Java | Maven `io.github.vanni-fan:idmix:0.4.0` | [Maven Central](https://central.sonatype.com/artifact/io.github.vanni-fan/idmix) |
| C# | `dotnet add package Vanni.Idmix` | [NuGet](https://www.nuget.org/packages/Vanni.Idmix) |
| VB.NET | `dotnet add package Vanni.Idmix.Vb` | [NuGet](https://www.nuget.org/packages/Vanni.Idmix.Vb) |
| C/C++ | See **FetchContent** above (user CMake integration) | [GitHub](https://github.com/Vanni-Fan/idmix) |


**C/C++ note**: There is no unified official registry like PyPI or NuGet, and no separate "publish" step on your side. The `cpp/` and `c/` directories in the repo are the source libraries; users add `FetchContent_Declare(...)` in **their own** CMake project to pull from GitHub. An optional [vcpkg](https://vcpkg.io) port has not been submitted upstream; this does not affect usage.

## Limits

- Maximum **255** objects per encode
- Maximum **63** bytes per string segment
- Best suited for small-to-medium integers; compression ratio drops for very large values
- Variant is chosen randomly; repeated `Encode` of the same input yields different strings, all decodable correctly
- **PHP** requires `ext-bcmath`; a clear runtime error is thrown if it is not installed

## License

Apache-2.0
