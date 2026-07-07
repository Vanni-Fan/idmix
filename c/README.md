# idmix — C 实现

> **English:** [README_en.md](README_en.md)

C 库实现 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**，与 Go 参考实现行为一致。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

## 架构

三层可独立使用：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（idmix.h）                                             │
│  idmix_encode / idmix_decode：整数/短字符串 → 文本            │
│  内部：Idx 编码 → Codec.encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx（idx.h）    │                 │  Codec（codec.h）    │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 紧凑自描述二进制，输出 `uint8_t*` |
| **Codec + idmix_encode_bytes / idmix_decode_string** | 包装任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

## 构建

```bash
cd c
cmake -B build
cmake --build build
ctest --test-dir build
```

## 快速开始

### 完整流程（IdMix）

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

### 仅 IDX 二进制层

```c
#include "idx.h"

idmix_idx_t* idx = idmix_idx_create(255, 32, 2);
idmix_value_t values[] = {IDMIX_INT(IDMIX_OTYPE_UINT16, 5)};

uint8_t* data = NULL;
size_t len = 0;
idmix_idx_encode(idx, values, 1, &data, &len);
/* ... */
idmix_idx_free_bytes(data);
idmix_idx_destroy(idx);
```

### 仅文本层

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

| 字段 | 说明 |
|------|------|
| `kind` | `IDMIX_KIND_INT` 或 `IDMIX_KIND_STRING` |
| `otype` / `val` | 整数类型与值（`kind == INT`） |
| `str` / `str_len` | 字符串指针与字节长度（`kind == STRING`，1~63 字节） |

解码后字符串由库 `malloc`，须通过 `idmix_free_values(values, count)` 释放。

## 规范二进制样例

`variant_id=0` 时，`[uint16(5), int64(-1), uint32(40)]` 编码为：

```
80 03 22 47 B5 1F
```

## 测试

跨语言向量位于 `../testdata/cross_language_vectors.json`。

## 版本

当前版本 **0.4.0**（IDX v1.2）。自 0.3.x（XID v1.1）起 header 格式与字符串支持有破坏性变更，详见 [arithmetic.md](../arithmetic.md)。

## 相关链接

- [Go 参考实现](../golang/README.md)
- [GitHub 仓库](https://github.com/Vanni-Fan/idmix)
