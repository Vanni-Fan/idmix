# idmix — C# 实现

> **English:** [README_en.md](README_en.md)

C# 实现，提供 **IDX 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

```
NuGet: Vanni.Idmix
```

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（高级封装）                                            │
│  Encode / Decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → ICodec.Encode                               │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx            │                 │  ICodec（接口）      │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
└─────────────────┘                 └─────────────────────┘
```

## 安装

```bash
dotnet add package Vanni.Idmix
```

**要求**：.NET 8.0+

```csharp
using Vanni.Idmix;
```

## 快速开始

### 完整流程（IdMix）

```csharp
var m = IdMix.Create();

// 整数 + 短字符串（≤63 字节）
var s = m.Encode((ushort)5, -1L, 40u, "hello");
var list = m.Decode(s);
// list[0] = (ushort)5, list[1] = -1L, list[2] = 40u, list[3] = "hello"
```

### 仅 IDX 二进制层

```csharp
var idx = Idx.Create();
var bin = idx.EncodeWithVariant(0, (ushort)5, -1L);
var out_ = idx.Decode(bin);
```

### 仅文本混淆层

```csharp
var protoBytes = new byte[] { 0x08, 0x96, 0x01 };
var text = Codec.EncodeBytes(protoBytes);
var raw = Codec.DecodeString(text);
```

## 支持的输入类型

`Idx.Encode` / `IdMix.Encode` 接受 `params object[]`，每个元素可为：

| C# 类型 | 说明 |
|---------|------|
| `byte`, `ushort`, `uint`, `ulong` | 无符号整数，保留位宽 |
| `sbyte`, `short`, `int`, `long` | 有符号整数 |
| `string` | UTF-8 文本，**1~63 字节** |
| `byte[]` | 原始字节串，**1~63 字节** |
| `TypedValue` | 带类型整数值（便捷包装） |

**限制**：

- 单次最多 **255** 个对象
- 字符串/字节串单段最长 **63** 字节
- 空字符串、空 `byte[]` 不允许

## API 参考

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

默认值：`maxObjects=255`, `maxVariants=32`, `checkBits=2`。

### ICodec / Codec

```csharp
// 默认 RadixCodec
Codec.EncodeBytes(data);
Codec.DecodeString(text);

// Base64
var b64 = Base64Codec.Instance;
Codec.EncodeBytes(data, b64);

// 自定义 RadixCodec
var radix = new RadixCodec(IdMix.DefaultAlphabet);
```

### IdMix

```csharp
var m = IdMix.Create();
var m2 = IdMix.Create(b => b.WithAlphabet("abcd"));
var m3 = IdMix.Create(b => b.WithCodec(Base64Codec.Instance));
var m4 = IdMix.Create(b => b.WithIdx(Idx.Create()));

string s = m.Encode((ushort)5, "hello");
string s2 = m.EncodeWithVariant(0, (ushort)5);  // 确定性编码
object[] list = m.Decode(s);

Idx idx = m.Idx;
ICodec codec = m.Codec;
```

## 测试

```bash
cd csharp/tests/Vanni.Idmix.Tests
dotnet test
```

## 相关链接

- [IDX v1.2 协议规范](../arithmetic.md)
- [Go 参考实现](../golang/README.md)
- [跨语言测试向量](../testdata/cross_language_vectors.json)
