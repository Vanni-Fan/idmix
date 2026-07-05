# vanni-idmix

XID v1.1 Python 实现：将带类型整数序列编码为短字符串。

## 安装

```bash
pip install vanni-idmix
```

## 用法

```python
from idmix import IdMix, u16, i64, u32

m = IdMix.new()
s = m.encode(u16(5), i64(-1), u32(40))
out = m.decode(s)
```

完整文档见 [GitHub 仓库](https://github.com/Vanni-Fan/idmix)。
