# vanni-idmix — Python 实现

> **English:** [README_en.md](README_en.md)

Python 参考实现，提供 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

## 架构概览

本包分三层，**可独立使用**：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（高级封装）                                            │
│  encode / decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → Codec.encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec（协议）       │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
│  可单独使用      │                 │  可单独使用          │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 只需紧凑自描述二进制，输出 `bytes` |
| **Codec + encode_bytes / decode_string** | 包装 Protobuf/CBOR/任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

## 安装

```bash
pip install vanni-idmix
```

**要求**：Python 3.9+

## 快速开始

### 完整流程（IdMix）

```python
from idmix import IdMix, u16, i64, u32

m = IdMix.new()

# 整数 + 短字符串（≤63 字节）
s = m.encode(u16(5), i64(-1), u32(40), "hello")
print("encoded:", s)

values = m.decode(s)
print(values)  # [5, -1, 40, 'hello']
```

> 解码返回 `int` / `str`。若需与 Go 一致的位宽编码，可使用 `u16()`、`i64()` 等工厂函数。

### 仅 IDX 二进制层

```python
from idmix import Idx, u16, i64

idx = Idx.new()
data = idx.encode(u16(5), i64(-1))
out = idx.decode(data)
```

### 仅文本混淆层

```python
from idmix import encode_bytes, decode_string

proto_bytes = bytes([0x08, 0x96, 0x01])
text = encode_bytes(proto_bytes)
raw = decode_string(text)
```

## 支持的输入类型

`Idx.encode` / `IdMix.encode` 接受可变参数，每个元素可为：

| Python 类型 | 说明 |
|-------------|------|
| `int` | 整数（按 int64 存储） |
| `TypedInt` / `u8()`…`i64()` | 带位宽的整数，与 Go 类型对齐 |
| `str` | UTF-8 文本，**1~63 字节**（按字节计） |
| `bytes` / `bytearray` | 原始字节串，**1~63 字节** |

**限制**：

- 单次最多 **255** 个对象（`with_max_objects` 可调）
- 字符串/字节串单段最长 **63** 字节
- 空字符串、空 `bytes` 不允许

## 模块结构

| 文件 | 职责 |
|------|------|
| `idx_codec.py` | `Idx` 二进制编解码 |
| `codec.py` | `Codec` 协议、`Base64Codec`、`FuncCodec`、包级函数 |
| `alphabet.py` | `RadixCodec` 默认进制编解码 |
| `number.py` | 类型规范化与 `TypedInt` |
| `idmix.py` | `IdMix` 高级封装 |

## 配置示例

### 自定义 Idx 参数

```python
from idmix import IdMix, Idx, with_idx, with_max_objects, with_max_variants, with_check_bits

idx = Idx.new(
    with_max_objects(100),
    with_max_variants(16),
    with_check_bits(2),
)
m = IdMix.new(with_idx(idx))
```

### 自定义字符表

```python
from idmix import IdMix, with_alphabet

m = IdMix.new(with_alphabet("abcd"))
```

### 使用 Base64 作为文本层

```python
from idmix import IdMix, Base64Codec, with_codec

m = IdMix.new(with_codec(Base64Codec.new()))
```

### 自定义 Codec（异或 + Radix）

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

## 测试

```bash
cd python
python -m unittest discover -s tests -v
```

跨语言向量位于 `../testdata/cross_language_vectors.json`。

## 相关链接

- [协议规范](../arithmetic.md)
- [Go 参考实现](../golang/README.md)
- [GitHub 仓库](https://github.com/Vanni-Fan/idmix)
