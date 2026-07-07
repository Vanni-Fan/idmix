# idmix — IDX v1.2 自描述序列 + 文本混淆

> **English:** [README_en.md](README_en.md)

将多个**带类型的整数**或**短字符串**（≤63 字节）编码为短字符串，适用于 access_key、短 ID、令牌等场景。**C、C++、C#、Go、Java、JavaScript、PHP、Python、Rust** 多语言实现互通，遵循 [IDX v1.2 规范](arithmetic.md)。

如果这个项目对你有帮助，欢迎在 GitHub 上点个 **⭐ Star** 支持一下：[github.com/Vanni-Fan/idmix](https://github.com/Vanni-Fan/idmix)

## 各语言文档


| 语言         | 中文                                           | English                                            |
| ---------- | -------------------------------------------- | -------------------------------------------------- |
| Go（参考实现）   | [golang/README.md](golang/README.md)         | [golang/README_en.md](golang/README_en.md)         |
| Rust       | [rust/lib/README.md](rust/lib/README.md)     | [rust/lib/README_en.md](rust/lib/README_en.md)     |
| Python     | [python/README.md](python/README.md)         | [python/README_en.md](python/README_en.md)         |
| JavaScript | [javascript/README.md](javascript/README.md) | [javascript/README_en.md](javascript/README_en.md) |
| Java       | [java/README.md](java/README.md)             | [java/README_en.md](java/README_en.md)             |
| C#         | [csharp/README.md](csharp/README.md)         | [csharp/README_en.md](csharp/README_en.md)         |
| PHP        | [php/README.md](php/README.md)               | [php/README_en.md](php/README_en.md)               |
| C++        | [cpp/README.md](cpp/README.md)               | [cpp/README_en.md](cpp/README_en.md)               |
| C          | [c/README.md](c/README.md)                   | [c/README_en.md](c/README_en.md)                   |


**两层可独立使用**：


| 层         | API（Go）                                     | 作用               |
| --------- | ------------------------------------------- | ---------------- |
| **IDX**   | `NewIdx()` / `Idx.Encode` / `Idx.Decode`    | 整数/短字符串 ↔ 自描述二进制 |
| **Codec** | `EncodeBytes` / `DecodeString` + `Codec` 接口 | 任意二进制 ↔ 文本（可插拔）  |
| **组合**    | `IdMix.Encode` / `Decode`                   | IDX + Codec      |


文本层通过 `Codec` **接口**可插拔：默认 `RadixCodec`（自定义字符表），也可换 `Base64Codec`、或 `FuncCodec` 包装 AES/XOR 等。

也可将 **Protobuf / CBOR / MessagePack** 二进制直接经 `EncodeBytes` 输出混淆文本，替代传统的 `Protobuf + AES + Base64` 路径。

## 用途

- **类型自描述**：每个整数携带原始类型（`uint8 ~ uint64` / `int8 ~ int64` 共 8 种 otype），解码时不依赖外部 schema
- **短字符串**：扩展模式支持 ≤63 字节的 UTF-8/字节串
- **极致压缩**：[0, 15] 的正数、[-1, -15] 的负数仅占 1 字节；**单对象时 header 仅 1 字节**
- **32 态多态**：同一组数据可生成最多 32 种不同编码（variant 异或混淆）
- **轻量自校验**：2-bit 全局 XOR 校验，约 75% 的随机篡改可被即时拒绝
- **文本混淆（idmix 层）**：默认 62 进制字符表，可单独包装任意二进制



## 算法概要（IDX v1.2 + idmix）

```
整数/短字符串 → [IDX 二进制层] → [idmix 文本层] → 字符串
任意二进制    → [idmix 文本层] → 字符串          （跳过 IDX）
```



### 1. IDX 二进制层

二进制块 = **1 或 2 字节 header** + **数据对象序列**。

**Header（单对象，1 字节）**：


| 位域       | 宽度  | 含义                 |
| -------- | --- | ------------------ |
| bit[1:0] | 2   | `check` 校验位        |
| bit[6:2] | 5   | `variant_id`（0~31） |
| bit7     | 1   | `0` = 单对象          |


**Header（多对象，2 字节）**：字节 0 的 bit7=1，字节 1 为 count（2~255）。

**数据对象**：

- **内嵌模式（1 字节）**：小整数 [-15,15]
- **扩展数字（1+ 字节）**：bit6=0，有符号类型用二补码小端负载（无独立符号位）
- **扩展字符串（1+ 字节）**：bit6=1，bit5-0 为长度（1~63），后跟原始字节

**变体混淆**：`mask = (variant_id × 0x9D + 0x37) & 0xFF`，对对象区逐字节 XOR（header 不参与）。

规范示例 `uint16(5), int64(-1), uint32(40)`（variant=0，三对象）的二进制块：

```
80 03 22 47 B5 1F
```

完整协议见 [arithmetic.md](arithmetic.md)。

### 2. idmix 文本层

在二进制块前附加 2 字节大端长度前缀，整体按**自定义进制**（默认 62 字符表）转为字符串。可单独用于任意二进制，不限于 IDX 输出。

## 与 [Sqids](https://sqids.org/) 功能对比

[Sqids](https://sqids.org/) 是流行的短 ID 库，仅支持**非负整数序列**。idmix 在此基础上面向结构化 access_key 场景做了扩展。


| 功能              | Sqids            | idmix                                    |
| --------------- | ---------------- | ---------------------------------------- |
| 输入类型            | 非负整数序列（`uint64`） | 任何整数数值（`uint8 - uint64` / `int8 - int64`），以小于64字节的字符 |
| 负数支持            | 不支持              | 支持                           |
| 负数字符串          | 不支持              | 支持，但长度最大63              |
| 类型自描述           | 否，解码需外部 schema   | 是，每个值携带原始类型                        |
| 多态编码            | 否，同输入同输出         | 是，默认 32 种变体字符串（可配置）            |
| 自校验             | 否                | 是，2-bit 全局 XOR（约 75% 拒随机串）（可配置）       |
| 阻止列表（Blocklist） | 支持，过滤敏感词         | **不支持**（见下说明）                      |
| 最小长度约束          | 支持 `min_length`  | 不支持（追求最短自然编码）                       |
| 自定义字母表          | 支持（默认 64 字符）     | 支持（默认 62 字符）                       |
| 单次最大元素数         | 无硬性上限            | 最大255，默认 255（可配置）                  |


**关于阻止列表**：idmix 不提供 Sqids 式的 blocklist 过滤。原因是 idmix 内置 **32 态变体多态**，减小最大元素个数，还可以进一步增多变体数量——同一组数据每次编码会随机选取不同 `variant_id`，字符串形态天然分散，出现特定敏感词的概率极低；若偶发不满意，重新调用 `Encode` 即可得到另一变体。配合 2-bit 校验，随机猜测的有效率也更低。

以下为 **Go** 与 **Rust** 实现相对 sqids 的**编码长度与性能**对比（本机单线程，20000 次采样；idmix 长度含 32 态变体，报告 min~max）。极值对比中 sqids 仅支持非负整数，`int64_max` / `uint64_max` 以 `uint64` 传入；带负数的 `int32_min` / `int64_min` / `mixed_extremes` 为 idmix 独有能力（见文末）。

### 编码长度


| 场景                                 | sqids | idmix |
| ---------------------------------- | ----- | ----- |
| [1, 2, 3]                          | 6     | 8     |
| 单值 [42]                            | 2     | 6     |
| 单值 uint32 [2000000000]             | 7     | 10    |
| AccessKey [1001, 1690000000, 3]    | 12    | 16    |
| 小整数 [0..9]                         | 20    | 17    |
| **uint32_max**                     | 7     | 10    |
| **int64_max**                      | 12    | 16    |
| **uint64_max**                     | 12    | 16    |
| **极值三元组 [u32max, i64max, u64max]** | 31    | 35    |


sqids 在纯非负小整数场景下字符串更短；idmix 因 header、类型标记和变体开销略长，但小整数密集时反而更短（内嵌模式 1 字节/值）。极值单字段时 idmix 仍略长于 sqids（如 `uint32_max`：10 vs 7），但三元组极值序列差距缩小（35 vs 31）。

### 编码性能（idmix / sqids 倍数，>1 表示 idmix 更快）


| 场景                                 | Go 编码     | Rust 编码 | Go 解码    | Rust 解码 |
| ---------------------------------- | --------- | ------- | -------- | ------- |
| [1, 2, 3]                          | **47.2×** | 12.1×   | **8.0×** | 5.3×    |
| AccessKey [1001, 1690000000, 3]    | **20.6×** | 16.2×   | **2.4×** | 4.2×    |
| uint32 [2000000000]                | **16.3×** | 12.7×   | **1.9×** | 4.0×    |
| **uint32_max**                     | **14.3×** | 12.8×   | **2.6×** | 4.0×    |
| **int64_max**                      | **10.8×** | 17.3×   | **1.3×** | 3.0×    |
| **uint64_max**                     | **16.5×** | 16.8×   | **1.4×** | 2.9×    |
| **极值三元组 [u32max, i64max, u64max]** | **7.8×**  | 10.2×   | **2.3×** | 2.7×    |


idmix 编解码以位运算为主，显著快于 sqids；Go 实现整体快于 Rust（Rust 文本层使用 `num-bigint`）。
极值场景下 idmix 编码优势仍明显（7.8× - 47×），解码优势随数值增大而收窄（大整数 radix 解码约 1.3× - 2.6×，仍快于或接近 sqids）。

### idmix 额外能力（sqids 不支持）

- 带类型：`uint16(5), int64(-1), uint32(40)` → 9 字符
- 负数与有符号极值：`int32_min` → 10 字符，`int64_min` → 16 字符
- 含负数的 `mixed_extremes`（五字段极值）→ 54 字符
- 随机变体：同一输入多次编码产生不同字符串

运行对比测试：

```bash
# Go — 长度
cd golang && go test -v -run TestCompareSqids

# Go — 性能（含极值）
cd golang && go test -v -run TestCompareSqidsPerformance

# Rust
cd rust/lib && cargo test --test benchmark_sqids -- --nocapture
```



## 与 MessagePack / CBOR / Protobuf 对比（IDX 二进制层）

以下对比 **IDX 二进制字节数**与 MessagePack、CBOR、Protobuf（无 schema、逐字段带 `otype`+`val`），**不含 base64 或 idmix 文本层**，与 MsgPack/CBOR/Protobuf 公平对比。

### 编码长度（字节）


| 场景             | IDX | MsgPack | CBOR | Protobuf |
| -------------- | --- | ------- | ---- | -------- |
| spec_example   | 6   | 46      | 23   | 27       |
| uint32_max     | 6   | 16      | 12   | 10       |
| int32_min      | 6   | 16      | 12   | 15       |
| int64_min      | 10  | 16      | 16   | 15       |
| int64_max      | 10  | 16      | 16   | 14       |
| uint64_max     | 10  | 16      | 8    | 15       |
| mixed_extremes | 39  | 76      | 60   | 69       |
| access_key     | 11  | 46      | 28   | 23       |
| embedded_small | 6   | 61      | 29   | 42       |
| string_example | 16  | 16      | 8    | 6        |


`mixed_extremes` 含五个极值单字段；`string_example` = `"hello"` + `uint16(5)` + `"世界"`（variant=0，Go 参考实现实测）。

IDX 在 typed 整数场景下二进制体积通常**远小于** MsgPack；短字符串内联时与 CBOR 接近。

### 编解码性能（IDX 二进制层，相对倍数）

Go 参考实现，单线程，每项 20000 次采样。倍数为 **IDX ops/s ÷ 对方 ops/s**；**>1 表示 IDX 更快**。

**编码**


| 场景             | vs MsgPack | vs CBOR | vs Protobuf |
| -------------- | ---------- | ------- | ----------- |
| spec_example   | 2.2×       | 1.3×    | 0.8×        |
| access_key     | 3.2×       | 2.2×    | 1.1×        |
| embedded_small | 3.7×       | 1.9×    | 1.8×        |
| mixed_extremes | 1.7×       | 0.7×    | 0.9×        |


**解码**


| 场景             | vs MsgPack | vs CBOR | vs Protobuf |
| -------------- | ---------- | ------- | ----------- |
| spec_example   | 5.9×       | 4.3×    | 0.8×        |
| access_key     | 4.9×       | 3.1×    | 0.6×        |
| embedded_small | 7.1×       | 5.7×    | 0.8×        |
| mixed_extremes | 3.0×       | 2.7×    | 0.5×        |


**小结**：IDX 二进制编解码在中小整数场景显著快于 MsgPack/CBOR；若需 URL 可读字符串，再叠加 idmix 文本层。

运行对比测试：

```bash
cd golang && go test -v -run TestCompareSerializationFormats
cd golang && go test -v -run TestCompareSerializationFormatsPerformance
```



## 各语言实现与 uint64 支持

所有实现均覆盖 **uint8~uint64**（及对应有符号类型 int8~int64，共 8 种 otype）；扩展模式负载为无符号小端字节，`otype` 标明类型，有符号类型编码时去掉冗余符号扩展字节以压缩体积。


| 语言             | uint64 / 大整数                            | 解码后的值类型                                                                               | 特殊要求                                                                                 |
| -------------- | ------------------------------------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| **Go**         | 原生 `uint64` / `int64`                     | 原生数值类型                                                                                 | 无                                                                                    |
| **Rust**       | 原生 `u64` / `i64`                          | 原生数值类型                                                                                 | 文本层 radix 编码依赖 `num-bigint` crate                                                    |
| **Python**     | 原生任意精度 `int`                           | `int`                                                                                  | 无                                                                                    |
| **JavaScript** | `number` / `bigint`                         | 在 `±(2^53-1)` 内为 `number`，超出后为 **`bigint`**（如 `18446744073709551615n`）                  | Node.js 10.4+；跨语言 JSON 向量中 int64/uint64 用**字符串**避免 `JSON.parse` 精度丢失                 |
| **Java**       | `long` API + 无符号位模式（`Long.compareUnsigned`） | `long`（uint64 以无符号位模式存入 `long`）                                                       | 编码时 `TypedValue.u64()` 可接受十进制或 `0x` 十六进制**字符串**                                      |
| **C#**         | `ulong` / `long` + 无符号位模式                   | `ulong` / `long`                                                                       | `TypedValue.U64(ulong)`                                                              |
| **VB.NET**     | 同 C#                                        | 同 C#                                                                                   | 同 C#                                                                                 |
| **PHP**        | **ext-bcmath** 十进制字符串（`IntMath`）            | **所有整数均为十进制字符串**（`TypedValue::$val`），如 `"18446744073709551615"`                        | **必须启用** `bcmath` **扩展**（`composer.json` 已声明 `ext-bcmath`）；32/64 位 PHP 均可处理完整 uint64 |
| **C / C++**    | `uint64_t` / `int64_t`                      | 原生整数类型                                                                                 | C API 中 `idmix_value_t.val` 为 `int64_t`，uint64 大值以位模式传入                              |




### 不支持原生 uint64 时的解码表示

各语言在**二进制协议层完全互通**，但解码后返回给调用方的数值类型因语言而异。跨语言解码超大整数时，请按目标语言的表示方式处理：

| 场景 | PHP | JavaScript | 说明 |
| --- | --- | --- | --- |
| 解码 Go 编码的 `uint64_max` | `$out[0]->val === '18446744073709551615'` | `out[0].val === 18446744073709551615n` | PHP 一律为**十进制字符串**；JS 超出安全整数后为 **`bigint`** |
| 编码超大 uint64 | `TypedValue::u64('18446744073709551615')` | `u64(18446744073709551615n)` 或 `u64('18446744073709551615')` | 输入均可接受字符串；PHP 内部始终存为字符串 |
| 跨语言 JSON 向量 | 字段值为字符串 `"18446744073709551615"` | 字段值为字符串（测试文件避免 `JSON.parse` 丢精度） | 见 `testdata/cross_language_vectors.json` |

**PHP 示例**（解码其它语言编码的 uint64 超大数）：

```php
$out = $m->decode($str);
// $out[0] 为 TypedValue，otype=3（uint64）
echo $out[0]->val; // "18446744073709551615"（字符串，非 int）
```

**JavaScript 示例**：

```javascript
const out = m.decode(str);
// out[0].otype === 3
typeof out[0].val === 'bigint'  // true（超大 uint64）
out[0].val === 18446744073709551615n
```

> **注意**：JavaScript 用 `bigint` 而非字符串表示超大整数；只有 PHP（以及跨语言 JSON 向量文件）使用十进制字符串。比较时请用 `typedValuesEqual()` 或各语言提供的等价工具，不要直接 `===` 跨语言比较。

### 扩展模式说明

[arithmetic.md](arithmetic.md) 第 2.2 节核心思路：

1. **扩展数字**（bit6=0）：有符号类型用二补码小端负载，无符号类型用无符号小端；**不再使用 bit6 符号位**。
2. **扩展字符串**（bit6=1）：bit5-0 为长度（1~63），后跟原始字节。
3. **存储宽度** `sw` 仅由数值大小决定，与 otype 位宽无关。
4. **内嵌模式**（bit7=0）：覆盖 **[-15, 15]**（+16 占 2 字节扩展）。
5. **单对象 header** 仅 1 字节，多对象时追加 count 字节。



### 跨语言互操作测试

共享向量文件：`[testdata/cross_language_vectors.json](testdata/cross_language_vectors.json)`（由 Go 参考实现生成）。

```bash
# 重新生成向量（修改常量或算法后）
cd golang && GENERATE_VECTORS=1 go test -run TestGenerateCrossLanguageVectors

# 各语言校验
cd golang && go test -run CrossLanguage
cd php && php tests/run_tests.php
cd python && python -m unittest discover -s tests -v
cd javascript && npm test
cd rust/lib && cargo test --test extreme_cross_language
cd java && mvn test
cd csharp/tests/Vanni.Idmix.Tests && dotnet test
```



## 快速开始



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
    // 组合：IDX + Codec
    m, _ := idmix.New()
    str, _ := m.Encode(uint16(5), int64(-1), uint32(40), "hello")
    list, _ := m.Decode(str)
    fmt.Println(str, list[0].(uint16), list[1].(int64), list[2].(uint32), list[3].(string))

    // 仅 IDX 二进制层（maxObjects 等在此配置）
    idx, _ := idmix.NewIdx(idmix.WithMaxObjects(100))
    bin, _ := idx.Encode(uint16(5), int64(-1))
    out, _ := idx.Decode(bin)

    // 仅文本层（包级函数，Codec 可选）
    protoBytes := []byte{0x08, 0x96, 0x01}
    obfuscated, _ := idmix.EncodeBytes(protoBytes)
    obfuscated, _ = idmix.EncodeBytes(protoBytes, idmix.NewBase64Codec())
    raw, _ := idmix.DecodeString(obfuscated, idmix.NewBase64Codec())

    // 自定义 Codec：WithCodec(radix) 或 WithAlphabet("abcd")
    radix, _ := idmix.NewRadixCodec(idmix.DefaultAlphabet)
    custom, _ := idmix.New(idmix.WithIdx(idx), idmix.WithCodec(radix))
    _, _ = custom, out
    _ = raw
}
```



### PHP

需要 PHP 8.1+ 及 **ext-bcmath**、**ext-mbstring** 扩展（`composer.json` 已声明）。整数经 bcmath 以十进制字符串运算，32/64 位 PHP 均可处理完整 uint64 范围。

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
cargo add idmix@0.3.0
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

Maven 依赖（[Maven Central](https://central.sonatype.com/artifact/io.github.vanni-fan/idmix/0.3.0)）：

```xml
<dependency>
    <groupId>io.github.vanni-fan</groupId>
    <artifactId>idmix</artifactId>
    <version>0.3.0</version>
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

C/C++ **没有**类似 npm / crates.io 的统一官方包仓库，无需额外「发布」步骤。用户在自己的 CMake 工程里通过 **FetchContent** 从 GitHub 拉取源码即可（下面这段 CMake 是**用户项目**里写的，不是本仓库维护项）：

```cmake
include(FetchContent)
FetchContent_Declare(idmix
  GIT_REPOSITORY https://github.com/Vanni-Fan/idmix.git
  GIT_TAG v0.3.0
  SOURCE_SUBDIR cpp
)
FetchContent_MakeAvailable(idmix)
target_link_libraries(your_app PRIVATE idmix::idmix)
```

也可 clone 后本地构建：

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

同样通过 FetchContent 集成（`SOURCE_SUBDIR` 改为 `c`，链接 `idmix::c`）：

```cmake
include(FetchContent)
FetchContent_Declare(idmix_c
  GIT_REPOSITORY https://github.com/Vanni-Fan/idmix.git
  GIT_TAG v0.3.0
  SOURCE_SUBDIR c
)
FetchContent_MakeAvailable(idmix_c)
target_link_libraries(your_app PRIVATE idmix::c)
```

或本地构建：

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



## 自定义字符表

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



## 项目结构

```
idmix/
├── README.md        # 项目总览（中文）
├── README_en.md     # Project overview (English)
├── arithmetic.md    # IDX v1.2 完整规范 + idmix 文本层
├── PACKAGING.md     # 各语言包发布与安装指南
├── golang/          # Go 参考实现（含 README / README_en）
├── rust/lib/        # Rust crate
├── php/             # PHP (Composer: vanni/idmix)
├── python/          # Python (pip: vanni-idmix)
├── javascript/      # JavaScript (npm: @vanni.fan/idmix)
├── java/            # Java (Maven: io.github.vanni-fan:idmix)
├── csharp/          # C# (NuGet: Vanni.Idmix)
├── vb/              # VB.NET (NuGet: Vanni.Idmix.Vb)
├── cpp/             # C++ 库
├── c/               # C 库
└── testdata/        # 跨语言测试向量 cross_language_vectors.json
```



## 运行测试（详细输出）

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



## 包安装（无需 git clone）

当前版本 **0.4.0**（IDX v1.2）。各语言可通过包管理器直接安装；发布细节见 [PACKAGING.md](PACKAGING.md)。


| 语言         | 安装命令                                       | 包仓库                                                                              |
| ---------- | ------------------------------------------ | -------------------------------------------------------------------------------- |
| Go         | `go get github.com/Vanni-Fan/idmix/golang` | [pkg.go.dev](https://pkg.go.dev/github.com/Vanni-Fan/idmix/golang)               |
| PHP        | `composer require vanni/idmix`             | [Packagist](https://packagist.org/packages/vanni/idmix)                          |
| Rust       | `cargo add idmix@0.4.0`                    | [crates.io](https://crates.io/crates/idmix)                                      |
| Python     | `pip install vanni-idmix`                  | [PyPI](https://pypi.org/project/vanni-idmix/)                                    |
| JavaScript | `npm install @vanni.fan/idmix`             | [npm](https://www.npmjs.com/package/@vanni.fan/idmix)                            |
| Java       | Maven `io.github.vanni-fan:idmix:0.4.0`    | [Maven Central](https://central.sonatype.com/artifact/io.github.vanni-fan/idmix) |
| C#         | `dotnet add package Vanni.Idmix`           | [NuGet](https://www.nuget.org/packages/Vanni.Idmix)                              |
| VB.NET     | `dotnet add package Vanni.Idmix.Vb`        | [NuGet](https://www.nuget.org/packages/Vanni.Idmix.Vb)                           |
| C/C++      | 见上文 **FetchContent**（用户 CMake 集成）          | [GitHub](https://github.com/Vanni-Fan/idmix)                                     |


**C/C++ 说明**：没有像 PyPI / NuGet 那样的统一官方包仓库，也**不需要**你再去某个官网「发布」。仓库里的 `cpp/`、`c/` 即为源码库；用户在**自己的** CMake 工程里写 `FetchContent_Declare(...)` 从 GitHub 拉取即可。可选的 [vcpkg](https://vcpkg.io) 端口尚未提交上游，不影响使用。

## 限制

- 单次编码最多 **255** 个对象
- 字符串单段最长 **63** 字节
- 推荐用于中小整数；过大数值压缩率下降
- 变体随机选取，同一输入多次 Encode 字符串不同，但均可正确解码
- **PHP** 依赖 `ext-bcmath`；未安装时运行时将抛出明确错误



## 许可证

Apache-2.0

---

觉得有用的话，欢迎给项目点个 **⭐**：[github.com/Vanni-Fan/idmix](https://github.com/Vanni-Fan/idmix)