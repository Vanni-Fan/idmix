# idmix — C++ 实现

> **English:** [README_en.md](README_en.md)

C++17 库实现 **IDX v1.2 二进制编码**与 **idmix 可插拔文本层**，与 Go 参考实现行为一致。协议规范见仓库根目录 [arithmetic.md](../arithmetic.md)。

## 架构

三层可独立使用：

```
┌─────────────────────────────────────────────────────────────┐
│  IdMix（idmix.hpp）                                           │
│  encode / decode：整数/短字符串 → 文本                        │
│  内部：Idx 编码 → Codec.encode                                │
└───────────────────────────┬─────────────────────────────────┘
                            │
         ┌──────────────────┴──────────────────┐
         ▼                                      ▼
┌─────────────────┐                 ┌─────────────────────┐
│  Idx（idx.hpp）  │                 │  Codec（codec.hpp）  │
│  二进制编解码    │                 │  二进制 ↔ 文本       │
└─────────────────┘                 └─────────────────────┘
```

| 组件 | 典型用途 |
|------|----------|
| **Idx** | 紧凑自描述二进制，输出 `std::vector<uint8_t>` |
| **encodeBytes / decodeString** | 包装任意二进制为文本 |
| **IdMix** | access_key、短 ID：整数/字符串序列 → 混淆文本 |

## 构建

```bash
cd cpp
cmake -B build
cmake --build build
ctest --test-dir build
```

## 快速开始

### 完整流程（IdMix）

```cpp
#include "idmix/idmix.hpp"
#include <iostream>

int main() {
    idmix::IdMix m = idmix::IdMix::newDefault();

    std::vector<idmix::Value> values = {
        idmix::TypedValue::u16(5),
        idmix::TypedValue::i64(-1),
        idmix::TypedValue::u32(40),
        std::string("hello"),
    };

    auto s = m.encode(values);
    auto out = m.decode(s);
    // out == values
}
```

### 仅 IDX 二进制层

```cpp
#include "idmix/idx.hpp"

idmix::Idx idx;
auto bin = idx.encode({idmix::TypedValue::u16(5), idmix::TypedValue::i64(-1)});
auto out = idx.decode(bin);
```

### 仅文本层

```cpp
#include "idmix/codec.hpp"

auto text = idmix::encodeBytes({0x08, 0x96, 0x01});
auto raw = idmix::decodeString(text);
```

## Value 类型

`Value = std::variant<TypedValue, std::string>`

| 类型 | 说明 |
|------|------|
| `TypedValue` | 带位宽整数（`u8()`…`i64()` 工厂） |
| `std::string` | UTF-8 文本，**1~63 字节** |

**限制：**

- 单次最多 **255** 个对象（`Idx(255, 32, 2)` 默认）
- 字符串最长 **63** 字节
- 空字符串不允许

## 主要 API

### Idx

```cpp
idmix::Idx idx(255, 32, 2);
idx.encode(values);
idx.encodeWithVariant(0, values);  // 确定性编码
idx.decode(data);
```

单对象时 header **1 字节**；多对象时 **2 字节**（含 count）。

### IdMix

```cpp
idmix::IdMix m("abcd");  // 自定义字符表
m.encode(values);
m.encodeWithVariant(0, values);
m.decode(s);
m.idx();    // 内嵌 Idx
m.codec();  // 内嵌 ICodec
```

### Codec

- `RadixCodec` — 默认 62 进制（`DEFAULT_ALPHABET`）
- `ICodec` — 可扩展接口
- `encodeBytes` / `decodeString` — 包级函数（默认 RadixCodec）

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
