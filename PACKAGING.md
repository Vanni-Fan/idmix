# 各语言包发布指南

用户无需 `git clone` 即可通过各语言包管理器安装 idmix。以下说明**包名**、**安装命令**及**发布前需注册的账号**。

> 当前仓库已配置好各语言的包元数据；**首次发布**需你在对应平台注册账号并执行一次发布命令。发布后用户即可直接安装。

## 汇总

| 语言 | 包名 | 安装命令 | 发布平台 | 需注册账号 |
|------|------|----------|----------|------------|
| Go | `github.com/Vanni-Fan/idmix/golang` | `go get github.com/Vanni-Fan/idmix/golang` | GitHub（module proxy） | GitHub |
| PHP | `vanni/idmix` | `composer require vanni/idmix` | [Packagist](https://packagist.org) | Packagist + GitHub |
| Rust | `idmix` | `cargo add idmix` | [crates.io](https://crates.io) | crates.io + GitHub |
| Python | `vanni-idmix` | `pip install vanni-idmix` | [PyPI](https://pypi.org) | PyPI |
| JavaScript | `@vanni/idmix` | `npm install @vanni/idmix` | [npm](https://www.npmjs.com) | npm |
| Java | `fan.vanni:idmix` | Maven 依赖（见下） | [Maven Central](https://central.sonatype.com) | Sonatype OSS |
| C# | `Vanni.Idmix` | `dotnet add package Vanni.Idmix` | [NuGet](https://www.nuget.org) | NuGet + Microsoft |
| VB.NET | `Vanni.Idmix.Vb` | `dotnet add package Vanni.Idmix.Vb` | NuGet | 同上 |
| C/C++ | `idmix` | vcpkg / CMake FetchContent | GitHub / vcpkg | GitHub |

---

## Go

```bash
go get github.com/Vanni-Fan/idmix/golang
```

将代码推送到 GitHub 后，Go module proxy 会自动索引。**无需额外账号**。

---

## PHP (Composer)

```bash
composer require vanni/idmix
```

1. 注册 [Packagist.org](https://packagist.org)
2. 提交包 URL：`https://github.com/Vanni-Fan/idmix`
3. 启用 GitHub Webhook 自动同步

---

## Rust (crates.io)

```bash
cargo add idmix
```

1. 注册 [crates.io](https://crates.io)
2. `cargo login` 后，在 `rust/lib/` 执行 `cargo publish`

---

## Python (pip)

```bash
pip install vanni-idmix
```

1. 注册 [PyPI](https://pypi.org)
2. 在 `python/` 目录：`python -m build && twine upload dist/*`

---

## JavaScript (npm)

```bash
npm install @vanni/idmix
```

1. 注册 [npmjs.com](https://www.npmjs.com)
2. 在 `javascript/` 目录：`npm publish --access public`

---

## Java (Maven Central)

```xml
<dependency>
  <groupId>fan.vanni</groupId>
  <artifactId>idmix</artifactId>
  <version>0.2.0</version>
</dependency>
```

1. 注册 [central.sonatype.com](https://central.sonatype.com)
2. 验证 namespace `fan.vanni`

---

## C# / VB.NET (NuGet)

```bash
dotnet add package Vanni.Idmix
dotnet add package Vanni.Idmix.Vb
```

1. 注册 [nuget.org](https://www.nuget.org)
2. `dotnet pack` + `dotnet nuget push`

---

## C / C++

### CMake FetchContent

```cmake
include(FetchContent)
FetchContent_Declare(idmix
  GIT_REPOSITORY https://github.com/Vanni-Fan/idmix.git
  GIT_TAG v0.2.0
  SOURCE_SUBDIR cpp
)
FetchContent_MakeAvailable(idmix)
target_link_libraries(your_app PRIVATE idmix::idmix)
```

### vcpkg

```bash
vcpkg install idmix
```

---

## 需注册账号 checklist

- [ ] **GitHub** — Go 自动可用
- [ ] **Packagist** — PHP
- [ ] **crates.io** — Rust
- [ ] **PyPI** — Python
- [ ] **npm** — JavaScript
- [ ] **Sonatype Central** — Java
- [ ] **NuGet** — C# / VB.NET
- [ ] **vcpkg**（可选）— C/C++
