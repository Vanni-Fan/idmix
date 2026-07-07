# idmix — Rust 实现

> **English:** [README_en.md](README_en.md)

Rust 库实现 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**，与 Go 参考实现行为一致。协议规范见仓库根目录 [arithmetic.md](../../arithmetic.md)。

## 架构

三层可独立使用：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix                                                       │
│  encode / decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → Codec.encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec（trait）      │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 紧凑自描述二进制，输出 `Vec<u8>` |
| **Codec + encode_bytes / decode_string** | 包装任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

## 安装

在 `Cargo.toml` 中添加（路径或 crates.io 发布后）：

```toml
[dependencies]
idmix = { path = "../rust/lib" }
```

## 快速开始

### 完整流程（IdMix）

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

### 仅 IDX 二进制层

```rust
use idmix::{Idx, Value};

let idx = Idx::new()?;
let bin = idx.encode(&[Value::U16(5), Value::I64(-1)])?;
let out = idx.decode(&bin)?;
```

### 仅文本层

```rust
use idmix::{encode_bytes, decode_string};

let raw = [0x08u8, 0x96, 0x01];
let text = encode_bytes(&raw, None)?;
let back = decode_string(&text, None)?;
```

## 支持的 Value 类型

| 变体 | 说明 |
|------|------|
| `Value::U8` … `Value::U64` | 无符号整数，保留位宽 |
| `Value::I8` … `Value::I64` | 有符号整数 |
| `Value::String` | UTF-8 文本，**1~63 字节** |
| `Value::Bytes` | 原始字节，**1~63 字节** |

**限制：**

- 单次最多 **255** 个对象（`IdxBuilder::max_objects`）
- 字符串/字节串最长 **63** 字节
- 空字符串、空字节串不允许

## 主要 API

### Idx

```rust
let idx = Idx::builder()
    .max_objects(255)   // 1~255，默认 255
    .max_variants(32)   // 1~32，默认 32
    .check_bits(2)      // 1 或 2，默认 2
    .build()?;

idx.encode(&values)?;
idx.encode_with_variant(0, &values)?;  // 确定性编码
idx.decode(&data)?;
```

单对象时 header **1 字节**；多对象时 **2 字节**（含 count）。

### IdMix

```rust
use std::sync::Arc;
use idmix::{Base64Codec, IdMix, IdMixBuilder};

let m = IdMix::builder()
    .alphabet("abcd")                    // 自定义 RadixCodec
    .codec(Arc::new(Base64Codec::new())) // 或自定义 Codec
    .idx(idx)
    .build()?;

m.encode(&values)?;
m.encode_with_variant(0, &values)?;
m.decode(&s)?;
```

### Codec trait

- `RadixCodec` — 默认 62 进制（`DEFAULT_ALPHABET`）
- `Base64Codec` — 标准 Base64
- `FuncCodec` — 闭包包装自定义逻辑

```rust
use idmix::{Codec, encode_bytes, decode_string};

let s = encode_bytes(&data, Some(&radix_codec))?;
let raw = decode_string(&s, Some(&radix_codec))?;
```

## 规范二进制样例

`variant_id=0` 时，`[uint16(5), int64(-1), uint32(40)]` 编码为：

```
80 03 22 47 B5 1F
```

## 测试

```bash
cd rust/lib
cargo test
```

包含单元测试、Idx 边界测试（`tests/idx_tests.rs`）及跨语言向量（`tests/extreme_cross_language.rs`）。

## 版本

当前版本 **0.4.0**（IDX v1.2）。自 0.3.x（XID v1.1）起 header 格式与字符串支持有破坏性变更，详见 [arithmetic.md](../../arithmetic.md)。
