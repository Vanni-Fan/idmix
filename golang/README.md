# idmix — Go 参考实现

> **English:** [README_en.md](README_en.md)

Go 语言参考实现，提供 **IDX 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

```
github.com/Vanni-Fan/idmix/golang
```

## 架构概览

本包分三层，**可独立使用**：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（高级封装）                                            │
│  Encode / Decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → Codec.Encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  Codec（接口）       │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
│  可单独使用      │                 │  可单独使用          │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 只需紧凑自描述二进制，输出 `[]byte` |
| **Codec + EncodeBytes / DecodeString** | 包装 Protobuf/CBOR/任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

---

## 安装

```bash
go get github.com/Vanni-Fan/idmix/golang
```

**要求**：Go 1.23+

```go
import idmix "github.com/Vanni-Fan/idmix/golang"
```

---

## 快速开始

### 完整流程（IdMix）

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

    // 整数 + 短字符串（≤63 字节）
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

### 仅 IDX 二进制层

```go
idx, _ := idmix.NewIdx()
bin, _ := idx.Encode(uint16(5), int64(-1))
out, _ := idx.Decode(bin) // []any，类型与编码时一致
_ = bin
_ = out
```

### 仅文本混淆层

```go
// 包装任意二进制（如 Protobuf 输出）
protoBytes := []byte{0x08, 0x96, 0x01}
text, _ := idmix.EncodeBytes(protoBytes)
raw, _ := idmix.DecodeString(text)
```

---

## 支持的输入类型

`Idx.Encode` / `IdMix.Encode` 接受 `...any`，每个元素可为：

| Go 类型 | 说明 |
|---------|------|
| `uint8`, `uint16`, `uint32`, `uint64`, `uint` | 无符号整数，保留位宽 |
| `int8`, `int16`, `int32`, `int64`, `int` | 有符号整数（`int` 按 `int64` 存储） |
| `string` | UTF-8 文本，**1~63 字节**（按字节计，非字符数） |
| `[]byte` | 原始字节串，**1~63 字节** |

解码返回 `[]any`，需自行类型断言，例如 `list[0].(uint16)`。

**限制**：

- 单次最多 **255** 个对象（`WithMaxObjects` 可调，上限 255）
- 字符串/字节串单段最长 **63** 字节
- 空字符串、空 `[]byte` 不允许

---

## API 参考

### 常量

#### `DefaultAlphabet`

```go
const DefaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
```

默认 `RadixCodec` 使用的 62 进制字符表。

---

### Idx — 二进制层

#### `type Idx struct`

IDX 二进制编解码器。配置项在创建时通过 `IdxOption` 注入，**不在 IdMix 上配置**。

#### `type IdxOption func(*Idx) error`

Idx 配置项函数类型。

#### `func NewIdx(opts ...IdxOption) (*Idx, error)`

创建 Idx 实例。默认值：

| 选项 | 默认值 | 范围 |
|------|--------|------|
| `maxObjects` | 255 | 1~255 |
| `maxVariants` | 32 | 1~32 |
| `checkBits` | 2 | 1~2 |

#### `func WithMaxObjects(n int) IdxOption`

设置单次编码允许的最大对象个数。

#### `func WithMaxVariants(n int) IdxOption`

设置变体数。`Encode` 随机选取 `[0, maxVariants)`；同一输入不同变体产生不同二进制（异或混淆）。

#### `func WithCheckBits(n int) IdxOption`

设置 header 中 XOR 校验位宽度（1 或 2 位）。

#### `func (idx *Idx) Encode(values ...any) ([]byte, error)`

将多个值编码为 IDX 二进制块。使用 `variant_id = 0`。

- 至少 1 个值
- 对象数不超过 `maxObjects`
- 单对象时 header **1 字节**；多对象时 **2 字节**（含 count）

#### `func (idx *Idx) EncodeWithVariant(variantID int, values ...any) ([]byte, error)`

与 `Encode` 相同，但指定 `variant_id`（0 ~ maxVariants-1）。用于测试或确定性编码。

#### `func (idx *Idx) Decode(data []byte) ([]any, error)`

将 IDX 二进制块解码为 `[]any`。失败原因包括：数据过短、校验和不匹配、变体越界、对象格式错误等。

---

### Codec — 文本层接口

#### `type Codec interface`

```go
type Codec interface {
    Encode(data []byte) (string, error)
    Decode(s string) ([]byte, error)
}
```

二进制与文本之间的可插拔编解码器。可实现：

- 自定义字符表进制（`RadixCodec`）
- 标准 Base64（`Base64Codec`）
- AES + Base64、异或 + Base64 等（`FuncCodec` 或自定义 struct）

#### `func EncodeBytes(data []byte, codec ...Codec) (string, error)`

包级函数：将任意二进制编码为文本。

- `codec` 不传时使用默认 `RadixCodec`（`DefaultAlphabet`）
- 传入 `nil` 会被忽略，仍用默认

#### `func DecodeString(s string, codec ...Codec) ([]byte, error)`

包级函数：将文本还原为二进制。`codec` 规则同 `EncodeBytes`。

---

### RadixCodec — 默认进制编解码器

#### `type RadixCodec struct`

基于自定义字符表的 Base-N 编解码（默认 idmix 文本层）。

编码策略：在数据前加 2 字节大端长度前缀，整体按字符表做大整数进制转换。

#### `func NewRadixCodec(alphabet string) (*RadixCodec, error)`

创建 RadixCodec。

- 字符表至少 2 个字符
- 字符不可重复

#### `func (rc *RadixCodec) Alphabet() string`

返回字符表字符串。

#### `func (rc *RadixCodec) Base() int`

返回进制基数（字符表长度）。

#### `func (rc *RadixCodec) Encode(data []byte) (string, error)`

实现 `Codec` 接口。

#### `func (rc *RadixCodec) Decode(s string) ([]byte, error)`

实现 `Codec` 接口。

---

### Base64Codec

#### `type Base64Codec struct{}`

标准 Base64 编解码器，实现 `Codec`。

#### `func NewBase64Codec() Base64Codec`

创建 Base64Codec 实例。

---

### FuncCodec — 函数式 Codec

#### `type FuncCodec struct`

```go
type FuncCodec struct {
    EncodeFn func(data []byte) (string, error)
    DecodeFn func(s string) ([]byte, error)
}
```

由函数实现的 `Codec`，便于组合加密/异或等逻辑。

---

### IdMix — 高级封装

#### `type IdMix struct`

组合 `Idx` 与 `Codec`。仅配置：

- **Idx**（`WithIdx`）
- **Codec**（`WithCodec` 或便捷 `WithAlphabet`）

#### `type Option func(*IdMix) error`

IdMix 配置项函数类型。

#### `func New(opts ...Option) (*IdMix, error)`

创建 IdMix。默认：`NewIdx()` + 默认 `RadixCodec`。

#### `func WithIdx(idx *Idx) Option`

注入已配置好的 Idx 实例（`maxObjects` 等在其上设置）。

#### `func WithCodec(codec Codec) Option`

设置文本编解码器。`codec` 不可为 `nil`。

#### `func WithAlphabet(alphabet string) Option`

便捷方法：用 `alphabet` 创建 `RadixCodec` 并设为 Codec。

#### `func (m *IdMix) Idx() *Idx`

返回内嵌 Idx。

#### `func (m *IdMix) Codec() Codec`

返回当前 Codec。

#### `func (m *IdMix) Encode(values ...any) (string, error)`

编码流程：

1. 随机选取 `variant_id ∈ [0, maxVariants)`
2. `Idx` 编码为二进制
3. `Codec.Encode` 转为文本

至少 1 个值；类型与长度限制同 Idx。

#### `func (m *IdMix) Decode(s string) ([]any, error)`

解码流程：

1. `Codec.Decode` 得到二进制
2. `Idx.Decode` 得到 `[]any`

#### `func (m *IdMix) EncodeWithVariant(variantID int, values ...any) (string, error)`

确定性编码（固定 `variant_id`），主要用于测试与跨语言向量生成。

---

## 配置示例

### 自定义 Idx 参数

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

### 自定义字符表

```go
m, err := idmix.New(idmix.WithAlphabet("abcd"))
// 等价于：
radix, _ := idmix.NewRadixCodec("abcd")
m, err := idmix.New(idmix.WithCodec(radix))
```

### 使用 Base64 作为文本层

```go
m, err := idmix.New(idmix.WithCodec(idmix.NewBase64Codec()))
s, _ := m.Encode(uint32(1001), uint8(3))
```

### 自定义 Codec（异或 + Radix）

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

### Protobuf + idmix 文本层（不用 Idx）

```go
import "google.golang.org/protobuf/encoding/protowire"

// 假设已有 proto 序列化结果
var protoBytes []byte

text, _ := idmix.EncodeBytes(protoBytes)           // 默认 Radix
text, _ = idmix.EncodeBytes(protoBytes, idmix.NewBase64Codec())

restored, _ := idmix.DecodeString(text)
```

---

## 类型断言与 uint64

解码后需断言为编码时的具体类型：

```go
list, _ := m.Decode(s)

u16 := list[0].(uint16)
i64 := list[1].(int64)
str := list[2].(string)

// uint64 大值（> MaxInt64）以位模式经 int64 存储，解码为 uint64
u64 := list[3].(uint64)
```

---

## 错误处理

常见错误：

| 场景 | 典型错误信息 |
|------|----------------|
| 空参数 | `at least one value is required` |
| 非支持类型 | `unsupported type %T` |
| 字符串过长 | `string length 64 exceeds max 63` |
| 空字符串 | `empty string is not allowed` |
| 对象过多 | `too many objects: N (max M)` |
| 非法配置 | `maxObjects must be between 1 and 255` 等 |
| 校验失败 | `checksum mismatch` |
| 变体越界 | `invalid variant_id N (max M)` |
| 非法字符表 | `alphabet contains duplicate character` |
| Codec 为 nil | `codec cannot be nil` |

建议对业务路径检查 `err != nil`，测试时可对边界用例断言具体错误。

---

## 测试

```bash
cd golang

# 全部测试
go test ./...

# 详细日志
go test -v ./...

# 单元素 1 字节 header
go test -v -run TestSingleObject

# 字符串 63/64 字节边界与 Idx 配置项
go test -v -run 'TestStringLengthBoundaries|TestIdxMax'

# 跨语言向量
go test -v -run CrossLanguage

# 重新生成 testdata（修改算法后）
# Windows PowerShell:
$env:GENERATE_VECTORS=1; go test -run TestGenerateCrossLanguageVectors

# 与 MsgPack/CBOR/Protobuf 二进制长度对比
go test -v -run TestCompareSerializationFormats
```

---

## 文件结构

```
golang/
├── README.md           # 中文文档（本文档）
├── README_en.md        # English documentation
├── go.mod
├── idmix.go            # IdMix 入口、包级 EncodeBytes/DecodeString
├── idx_codec.go        # Idx 二进制编解码
├── codec.go            # Codec 接口、Base64Codec、FuncCodec
├── alphabet.go         # RadixCodec
├── number.go           # any → 内部类型转换
├── idmix_test.go       # 端到端与演示测试
├── idx_test.go         # Idx 配置项与字符串边界测试
├── cross_language_test.go
├── vectors_test.go
└── benchmark_*.go      # 与 sqids、MsgPack 等对比基准
```

---

## 与其他语言实现

Go 为本仓库**参考实现**。跨语言互操作向量：

`../testdata/cross_language_vectors.json`

修改 IDX 算法或常量后，在 `golang` 目录执行 `GENERATE_VECTORS=1 go test -run TestGenerateCrossLanguageVectors` 重新生成，再同步其他语言实现。

---

## 相关链接

- [IDX v1.2 协议规范](../arithmetic.md)
- [项目总览 README](../README.md)
- [各语言打包说明](../PACKAGING.md)
- [pkg.go.dev](https://pkg.go.dev/github.com/Vanni-Fan/idmix/golang)
