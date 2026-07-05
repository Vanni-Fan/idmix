# idmix — XID v1.1 自描述整数序列短标识符

将多个**带类型的整数**编码为短字符串，适用于 access_key、短 ID、令牌等场景。Go、PHP、Rust、Python、JavaScript、Java、C#、VB.NET、C/C++ 多语言实现互通，遵循同一套 [XID v1.1 规范](arithmetic.md)。

## 用途

- **类型自描述**：每个整数携带原始类型（`uint8`~`int64`），解码时不依赖外部 schema
- **极致压缩**：0~15 的正数、-1~-16 的负数仅占 1 字节；其余按最小补码宽度存储
- **32 态多态**：同一组数据可生成最多 32 种不同字符串，防猜、规避敏感词
- **轻量自校验**：2-bit 全局 XOR 校验，约 75% 的随机篡改可被即时拒绝
- **自定义字符表**：默认 62 进制（`a-zA-Z0-9`），字符顺序可自定义

## 算法概要（XID v1.1）

编码分两层：

```
整数序列 → [二进制层] → [文本层] → 字符串
```

### 1. 二进制层

二进制块 = **2 字节 header（小端）** + **数据对象序列**。

**Header 位域（默认配置）**：

| 位域 | 宽度 | 含义 |
|------|------|------|
| bit[1:0] | 2 | `check` 校验位（全局 XOR 低 2 位） |
| bit[10:2] | 9 | `count` 对象个数（0~511） |
| bit[15:11] | 5 | `variant_id` 变体 ID（0~31） |

**数据对象**（每个整数一个）：

- **内嵌模式（1 字节）**：无符号 0~15，或有符号 -1~-16
- **扩展模式（1+ 字节）**：其余值，最小补码 + 类型标记（`otype` 0~7）

**变体混淆**：`mask = (variant_id × 0x9D + 0x37) & 0xFF`，对对象区逐字节 XOR（header 不参与）。

**校验**：对整个二进制块逐字节 XOR，低 2 位写入 header。

规范示例 `uint16(5), int64(-1), uint32(40)`（variant=0）的二进制块：

```
0F 00 22 47 B5 1F
```

完整协议见 [arithmetic.md](arithmetic.md)。

### 2. 文本层

在二进制块前附加 2 字节大端长度前缀，整体按**自定义进制**（默认 62 字符表）转为字符串。

## 与 [Sqids](https://sqids.org/) 功能对比

[Sqids](https://sqids.org/) 是流行的短 ID 库，仅支持**非负整数序列**。idmix 在此基础上面向结构化 access_key 场景做了扩展。

| 功能 | Sqids | idmix |
|------|-------|-------|
| 输入类型 | 非负整数序列（`uint64`） | 带类型整数序列（`uint8`~`int64`） |
| 负数支持 | 不支持 | 支持（内嵌 / 扩展模式） |
| 类型自描述 | 否，解码需外部 schema | 是，每个值携带原始类型 |
| 多态编码 | 否，同输入同输出 | 是，默认 32 种变体字符串 |
| 自校验 | 否 | 是，2-bit 全局 XOR（约 75% 拒随机串） |
| 阻止列表（Blocklist） | 支持，过滤敏感词 | **不支持**（见下说明） |
| 最小长度约束 | 支持 `min_length` | 不支持（追求最短自然编码） |
| 自定义字母表 | 支持（默认 64 字符） | 支持（默认 62 字符） |
| 单次最大元素数 | 无硬性上限 | 默认 511（可配置） |

**关于阻止列表**：idmix 不提供 Sqids 式的 blocklist 过滤。原因是 idmix 内置 **32 态变体多态**——同一组数据每次编码会随机选取不同 `variant_id`，字符串形态天然分散，出现特定敏感词的概率极低；若偶发不满意，重新调用 `Encode` 即可得到另一变体。配合 2-bit 校验，随机猜测的有效率也更低。

以下为 **Go** 与 **Rust** 实现相对 sqids 的**编码长度与性能**对比（本机单线程，20000 次采样；idmix 长度含 32 态变体，报告 min~max）。

### 编码长度

| 场景 | sqids | idmix (Go) | idmix (Rust) |
|------|-------|------------|--------------|
| [1, 2, 3] | 6 | 8~8 | 8~8 |
| 单值 [42] | 2 | 6~6 | 6~6 |
| 单值 uint32 [2000000000] | 7 | 10~10 | 10~10 |
| AccessKey [1001, 1690000000, 3] | 12 | 16~16 | 16~16 |
| 小整数 [0..9] | 20 | 17~17 | 17~17 |

sqids 在纯非负小整数场景下字符串更短；idmix 因 header、类型标记和变体开销略长，但小整数密集时反而更短（内嵌模式 1 字节/值）。

### 编码性能（相对 sqids 倍数）

| 场景 | Go 编码 | Rust 编码 | Go 解码 | Rust 解码 |
|------|---------|-----------|---------|-----------|
| [1, 2, 3] | **20.6×** | 12.4× | **2.9×** | 5.8× |
| uint32 [2000000000] | **12.1×** | 13.7× | **2.2×** | 3.9× |

idmix 编解码以位运算为主，显著快于 sqids；Go 实现整体快于 Rust（Rust 文本层使用 `num-bigint`）。

### idmix 额外能力（sqids 不支持）

- 带类型：`uint16(5), int64(-1), uint32(40)` → 9 字符
- 负数、混合类型序列
- 随机变体：同一输入多次编码产生不同字符串

运行对比测试：

```bash
# Go
cd golang && go test -v -run TestCompareSqids

# Rust
cd rust/lib && cargo test --test benchmark_sqids -- --nocapture
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
    m, _ := idmix.New()
    str, _ := m.Encode(uint16(5), int64(-1), uint32(40))
    list, _ := m.Decode(str)
    fmt.Println(str, list[0].(uint16), list[1].(int64), list[2].(uint32))
}
```

### PHP

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

```toml
[dependencies]
idmix = { path = "../rust/lib" }  # 或 crates.io 发布后 cargo add idmix
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
npm install @vanni/idmix
```

```javascript
import { IdMix } from '@vanni/idmix';
import { u16, i64, u32 } from '@vanni/idmix/src/typed_value.js';

const m = IdMix.new();
const s = m.encode(u16(5), i64(-1), u32(40));
const out = m.decode(s);
```

### Java

```bash
# Maven 依赖（发布后）：
# groupId: io.github.vanni-fan, artifactId: idmix
cd java && mvn test
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

### VB.NET

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

CMake FetchContent 或本地构建：

```bash
cd cpp && cmake -B build && cmake --build build
```

```cpp
#include "idmix/idmix.hpp"
using namespace idmix;

IdMix m = IdMix::newDefault();
auto s = m.encode({TypedValue::u16(5), TypedValue::i64(-1), TypedValue::u32(40)});
auto out = m.decode(s);
```

### C

```bash
cd c && cmake -B build && cmake --build build
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
├── arithmetic.md    # XID v1.1 完整规范
├── PACKAGING.md     # 各语言包发布与安装指南
├── golang/          # Go 参考实现
├── rust/lib/        # Rust crate
├── php/             # PHP (Composer: vanni/idmix)
├── python/          # Python (pip: vanni-idmix)
├── javascript/      # JavaScript (npm: @vanni/idmix)
├── java/            # Java (Maven: io.github.vanni-fan:idmix)
├── csharp/          # C# (NuGet: Vanni.Idmix)
├── vb/              # VB.NET (NuGet: Vanni.Idmix.Vb)
├── cpp/             # C++ 库
└── c/               # C 库
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

各语言可通过包管理器直接安装，详见 [PACKAGING.md](PACKAGING.md)。

| 语言 | 安装命令 |
|------|----------|
| Go | `go get github.com/Vanni-Fan/idmix/golang` |
| PHP | `composer require vanni/idmix` |
| Rust | `cargo add idmix` |
| Python | `pip install vanni-idmix` |
| JavaScript | `npm install @vanni/idmix` |
| Java | Maven `io.github.vanni-fan:idmix:0.2.0` |
| C# | `dotnet add package Vanni.Idmix` |
| VB.NET | `dotnet add package Vanni.Idmix.Vb` |
| C/C++ | CMake FetchContent 或 vcpkg（见 PACKAGING.md） |

> **首次发布**：仓库已配置好包元数据，你需在 Packagist、PyPI、npm、crates.io、NuGet、Sonatype 等平台注册账号并执行一次发布。详见 [PACKAGING.md](PACKAGING.md) 中的账号 checklist。

## 限制

- 单次编码最多 **511** 个对象（可配置）
- 推荐用于中小整数；过大数值压缩率下降
- 变体随机选取，同一输入多次编码字符串不同，但均可正确解码

## 许可证

Apache-2.0
