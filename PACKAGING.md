# 各语言包发布指南

用户无需 `git clone` 即可通过各语言包管理器安装 idmix。以下说明**包名**、**安装命令**及**发布前需注册的账号**。

> 当前仓库已配置好各语言的包元数据；**首次发布**需你在对应平台注册账号并执行一次发布命令。发布后用户即可直接安装。

## 汇总


| 语言         | 包名                                  | 安装命令                                       | 发布平台                                          | 需注册账号              |
| ---------- | ----------------------------------- | ------------------------------------------ | --------------------------------------------- | ------------------ |
| Go         | `github.com/Vanni-Fan/idmix/golang` | `go get github.com/Vanni-Fan/idmix/golang` | GitHub（module proxy）                          | GitHub             |
| PHP        | `vanni/idmix`                       | `composer require vanni/idmix`             | [Packagist](https://packagist.org)            | Packagist + GitHub |
| Rust       | `idmix`                             | `cargo add idmix`                          | [crates.io](https://crates.io)                | crates.io + GitHub |
| Python     | `vanni-idmix`                       | `pip install vanni-idmix`                  | [PyPI](https://pypi.org)                      | PyPI               |
| JavaScript | `@vanni/idmix`                      | `npm install @vanni/idmix`                 | [npm](https://www.npmjs.com)                  | npm                |
| Java       | `io.github.vanni-fan:idmix`         | Maven 依赖（见下）                               | [Maven Central](https://central.sonatype.com) | Sonatype Central   |
| C#         | `Vanni.Idmix`                       | `dotnet add package Vanni.Idmix`           | [NuGet](https://www.nuget.org)                | NuGet + Microsoft  |
| VB.NET     | `Vanni.Idmix.Vb`                    | `dotnet add package Vanni.Idmix.Vb`        | NuGet                                         | 同上                 |
| C/C++      | `idmix`                             | vcpkg / CMake FetchContent                 | GitHub / vcpkg                                | GitHub             |


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



### Packagist 提交说明（monorepo）

**Packagist 免费版只读取仓库根目录的** `composer.json`，不支持 `php/` 子目录 URL，因此：

- ✅ 正确：提交 `https://github.com/Vanni-Fan/idmix`（根目录已有 `composer.json`，autoload 指向 `php/src/`）
- ❌ 错误：提交 `https://github.com/Vanni-Fan/idmix/php`（会报 *No composer.json was found*）

发布步骤：

1. 注册 [Packagist.org](https://packagist.org)
2. 提交包 URL：`https://github.com/Vanni-Fan/idmix`（**不是** `/php` 子路径）
3. 推送上述根目录 `composer.json` 到 GitHub 的 `main` 分支
4. 在 Packagist 设置中启用 GitHub Webhook 自动同步

本地开发仍可在 `php/` 目录使用 `php/composer.json`（autoload 相对路径为 `src/`）。

---



## Rust ([crates.io](http://crates.io))

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



### 发布前：安装构建工具

`build` 和 `twine` **不是 Python 标准库**，需先安装（只需一次）：

```bash
python -m pip install --upgrade build twine
```

> Windows 上若直接输入 `twine` 报「无法识别」，请用 `python -m twine`（Scripts 目录可能不在 PATH 里）。



### 构建与上传

```bash
cd python
python -m build
python -m twine upload dist/*
```

首次上传会提示 PyPI 凭据；推荐使用 [API Token](https://pypi.org/manage/account/token/)（用户名填 `__token__`，密码填 `pypi-...`）。

1. 注册 [PyPI](https://pypi.org)（可选 [TestPyPI](https://test.pypi.org) 试发）
2. `python -m pip install build twine`
3. `cd python && python -m build`
4. `python -m twine upload dist/*`

---



## JavaScript (npm)

```bash
npm install @vanni.fan/idmix
```

1. 注册 [npmjs.com](https://www.npmjs.com)
2. 在 `javascript/` 目录：`npm publish --access public`

---



## Java (Maven Central)

Namespace：`io.github.vanni-fan`（GitHub 登录 Sonatype 时自动验证）

用户依赖：

```xml
<dependency>
    <groupId>io.github.vanni-fan</groupId>
    <artifactId>idmix</artifactId>
    <version>0.2.0</version>
</dependency>
```

```java
import io.github.vannifan.idmix.IdMix;
import io.github.vannifan.idmix.TypedValue;
```

> **注意**：Maven `groupId` 为 `io.github.vanni-fan`（可有 `-`），Java 包名为 `io.github.vannifan.idmix`（不能有 `-`）。



### 发布步骤（Central Portal，非旧版 OSSRH）

完整说明见 **[java/PUBLISHING.md](java/PUBLISHING.md)**。

1. 登录 [central.sonatype.com](https://central.sonatype.com)，确认 namespace `io.github.vanni-fan` 已验证
2. 生成 **User Token**（Account → Generate User Token）
3. 安装 **Gpg4win**，生成 GPG 密钥并上传到 keyserver，在 Portal 注册 GPG Key
4. 将 Token 写入 `%USERPROFILE%\.m2\settings.xml`（参考 `java/settings.xml.example`）
5. 发布：

```powershell
cd java
mvn test
mvn clean deploy -Prelease
```

1. 在 [Deployments](https://central.sonatype.com/publishing/deployments) 查看校验状态；约 10–30 分钟后可在 [search.maven.org](https://search.maven.org/) 搜索

---



## C# / VB.NET (NuGet)

```bash
dotnet add package Vanni.Idmix
dotnet add package Vanni.Idmix.Vb
```

NuGet 账号：**vanni.fan**（登录名，与包名 `Vanni.Idmix` 无关）

### 方式 A：Trusted Publishing（推荐，无需长期 API Key）

1. 在 GitHub 仓库 **Settings → Secrets → Actions** 添加：
  - `NUGET_USER` = `vanni.fan`（NuGet **用户名**，不是邮箱）
2. 在 [nuget.org → Trusted Publishing → Create](https://www.nuget.org/manage/trustedpublishers) 填写：


| 字段               | 填什么                 |
| ---------------- | ------------------- |
| Policy Name      | `idmix-publish`（任意） |
| Package Owner    | `vanni.fan`         |
| Repository Owner | `Vanni-Fan`         |
| Repository       | `idmix`             |
| Workflow File    | `publish-nuget.yml` |


1. 将 `.github/workflows/publish-nuget.yml` push 到 GitHub
2. 发布方式（二选一）：
  - GitHub → **Actions** → **Publish NuGet** → **Run workflow**（手动）
  - 或创建 GitHub **Release** tag（如 `v0.2.0`）自动触发



### 方式 B：API Key（本机一次性上传，官网标注 Not recommended）

仍可用，适合不想配 CI 时本机 `dotnet nuget push`：

1. [nuget.org → API Keys](https://www.nuget.org/account/apikeys) → Create
2. Glob：`Vanni.Idmix*`，Scope：Push
3. 本机执行：

```powershell
dotnet pack csharp/src/Vanni.Idmix/Vanni.Idmix.csproj -c Release
dotnet nuget push csharp/src/Vanni.Idmix/bin/Release/Vanni.Idmix.0.2.0.nupkg `
  --api-key YOUR_KEY --source https://api.nuget.org/v3/index.json
```

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

