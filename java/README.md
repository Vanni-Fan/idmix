# idmix — Java 实现

> **English:** [README_en.md](README_en.md)

Java 实现，提供 **IDX 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

```
Maven: io.github.vanni-fan:idmix
```

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（高级封装）                                            │
│  Encode / Decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → ICodec.encode                               │
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

```xml
<dependency>
    <groupId>io.github.vanni-fan</groupId>
    <artifactId>idmix</artifactId>
    <version>0.4.0</version>
</dependency>
```

**要求**：Java 17+

```java
import io.github.vannifan.idmix.*;
```

## 快速开始

### 完整流程（IdMix）

```java
IdMix m = IdMix.create();

// 整数 + 短字符串（≤63 字节）
String s = m.encode(TypedValue.u16(5), TypedValue.i64(-1), TypedValue.u32(40), "hello");
Object[] list = m.decode(s);
```

### 仅 IDX 二进制层

```java
Idx idx = Idx.create();
byte[] bin = idx.encodeWithVariant(0, TypedValue.u16(5), TypedValue.i64(-1));
Object[] out = idx.decode(bin);
```

### 仅文本混淆层

```java
byte[] protoBytes = new byte[] {0x08, (byte) 0x96, 0x01};
String text = Codec.encodeBytes(protoBytes);
byte[] raw = Codec.decodeString(text);
```

## 支持的输入类型

`Idx.encode` / `IdMix.encode` 接受 `Object...`，每个元素可为：

| Java 类型 | 说明 |
|-----------|------|
| `Byte`, `Short`, `Integer`, `Long` | 有符号整数包装类型 |
| `TypedValue` | 带类型整数值（推荐用于无符号类型） |
| `String` | UTF-8 文本，**1~63 字节** |
| `byte[]` | 原始字节串，**1~63 字节** |

**限制**：

- 单次最多 **255** 个对象
- 字符串/字节串单段最长 **63** 字节
- 空字符串、空 `byte[]` 不允许

## API 参考

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

## 测试

```bash
cd java
mvn test -q
```

## 相关链接

- [IDX v1.2 协议规范](../arithmetic.md)
- [Go 参考实现](../golang/README.md)
- [跨语言测试向量](../testdata/cross_language_vectors.json)
