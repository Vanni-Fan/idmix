# vanni-idmix — PHP 实现

> **English:** [README_en.md](README_en.md)

PHP 参考实现，提供 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

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
| **Idx** | 只需紧凑自描述二进制，输出二进制字符串 |
| **Codec + encodeBytes / decodeString** | 包装 Protobuf/CBOR/任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

## 安装

```bash
composer require vanni/idmix
```

**要求**：PHP 8.1+，`ext-bcmath`，`ext-mbstring`

## 快速开始

### 完整流程（IdMix）

```php
use Vanni\Idmix\IdMix;
use Vanni\Idmix\TypedValue;

$m = IdMix::new();

// 整数 + 短字符串（≤63 字节）
$s = $m->encode(TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40), 'hello');
$values = $m->decode($s);
// [TypedValue, TypedValue, TypedValue, 'hello']
```

> 解码整数返回 `TypedValue`（bcmath 字符串值，支持完整 uint64）；字符串返回 `string`。

### 仅 IDX 二进制层

```php
use Vanni\Idmix\Idx;
use Vanni\Idmix\TypedValue;

$idx = Idx::new();
$data = $idx->encode(TypedValue::u16(5), TypedValue::i64(-1));
$out = $idx->decode($data);
```

### 仅文本混淆层

```php
use function Vanni\Idmix\encodeBytes;
use function Vanni\Idmix\decodeString;

$proto = "\x08\x96\x01";
$text = encodeBytes($proto);
$raw = decodeString($text);
```

## 支持的输入类型

`Idx.encode` / `IdMix.encode` 接受可变参数，每个元素可为：

| PHP 类型 | 说明 |
|----------|------|
| `int` | 整数（按 int64 存储） |
| `TypedValue` / `u8()`…`i64()` | 带位宽的整数，与 Go 类型对齐 |
| `string` | 原始字节串，**1~63 字节**（按字节计） |
| `Bytes` | 原始字节包装类，**1~63 字节** |

**限制**：

- 单次最多 **255** 个对象（`Idx::new($maxObjects)` 可调）
- 字符串/字节串单段最长 **63** 字节
- 空字符串、空 `Bytes` 不允许

## 模块结构

| 文件 | 职责 |
|------|------|
| `Idx.php` | `Idx` 二进制编解码 |
| `Codec.php` | `Codec` 接口、`FuncCodec`、包级 `encodeBytes` / `decodeString` |
| `Base64Codec.php` | 标准 Base64 文本层 |
| `RadixCodec.php` | 默认进制编解码 |
| `Number.php` | 类型规范化 |
| `Idmix.php` | `IdMix` 高级封装 |

## 配置示例

### 自定义 Idx 参数

```php
$idx = Idx::new(maxObjects: 100, maxVariants: 16, checkBits: 2);
$m = IdMix::create($idx);
```

### 自定义字符表

```php
$m = IdMix::withAlphabet('abcd');
```

### 使用 Base64 作为文本层

```php
use Vanni\Idmix\Base64Codec;
use Vanni\Idmix\IdMix;

$m = IdMix::create(null, Base64Codec::new());
```

## 测试

```bash
cd php
php tests/run_tests.php
```

跨语言向量位于 `../testdata/cross_language_vectors.json`。

## 相关链接

- [协议规范](../arithmetic.md)
- [Go 参考实现](../golang/README.md)
- [GitHub 仓库](https://github.com/Vanni-Fan/idmix)
