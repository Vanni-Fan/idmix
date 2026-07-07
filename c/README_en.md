# idmix — C Implementation

> **中文:** [README.md](README.md)

C library for **IDX v1.2 binary encoding** and the **idmix pluggable text layer**, matching the Go reference. See [arithmetic.md](../arithmetic.md) for the protocol.

## Architecture

Three layers, usable independently:

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix (idmix.h)                                             │
│  idmix_encode / idmix_decode: integers/short strings → text │
│  Internally: Idx encode → Codec.encode                       │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx (idx.h)    │                 │  Codec (codec.h)    │
│  Binary codec   │                 │  Binary ↔ text      │
└─────────────────┘                 └─────────────────────┘
```

| Component | Typical use |
|-----------|-------------|
| **Idx** | Compact self-describing binary as `uint8_t*` |
| **Codec + idmix_encode_bytes / idmix_decode_string** | Wrap arbitrary binary as text |
| **IdMix** | access_key, short IDs: int/string sequences → obfuscated text |

## Build

```bash
cd c
cmake -B build
cmake --build build
ctest --test-dir build
```

## Quick start

### Full pipeline (IdMix)

```c
#include "idmix.h"
#include <stdio.h>

int main(void) {
    idmix_ctx_t* ctx = idmix_create(NULL);
    idmix_value_t values[] = {
        IDMIX_INT(IDMIX_OTYPE_UINT16, 5),
        IDMIX_INT(IDMIX_OTYPE_INT64, -1),
        IDMIX_INT(IDMIX_OTYPE_UINT32, 40),
        {IDMIX_KIND_STRING, 0, 0, (char*)"hello", 5},
    };

    char* encoded = NULL;
    idmix_encode(ctx, values, 4, &encoded);
    printf("%s\n", encoded);

    idmix_value_t* decoded = NULL;
    size_t count = 0;
    idmix_decode(ctx, encoded, &decoded, &count);

    idmix_free_string(encoded);
    idmix_free_values(decoded, count);
    idmix_destroy(ctx);
    return 0;
}
```

### IDX binary layer only

```c
#include "idx.h"

idmix_idx_t* idx = idmix_idx_create(255, 32, 2);
idmix_value_t values[] = {IDMIX_INT(IDMIX_OTYPE_UINT16, 5)};

uint8_t* data = NULL;
size_t len = 0;
idmix_idx_encode(idx, values, 1, &data, &len);
idmix_idx_free_bytes(data);
idmix_idx_destroy(idx);
```

### Text layer only

```c
#include "codec.h"

uint8_t raw[] = {0x08, 0x96, 0x01};
char* text = NULL;
idmix_encode_bytes(raw, sizeof(raw), &text);

uint8_t* back = NULL;
size_t back_len = 0;
idmix_decode_string(text, &back, &back_len);

idmix_free_string(text);
idmix_codec_free_bytes(back);
```

## idmix_value_t

| Field | Description |
|-------|-------------|
| `kind` | `IDMIX_KIND_INT` or `IDMIX_KIND_STRING` |
| `otype` / `val` | Integer type and value when `kind == INT` |
| `str` / `str_len` | String pointer and byte length when `kind == STRING` (1–63 bytes) |

Decoded strings are heap-allocated; free with `idmix_free_values(values, count)`.

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
