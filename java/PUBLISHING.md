# Maven Central 发布指南（Sonatype Central Portal）

Namespace：`io.github.vanni-fan`（GitHub 登录自动验证）

Maven 坐标：

| 字段 | 值 |
|------|-----|
| groupId | `io.github.vanni-fan` |
| artifactId | `idmix` |
| Java 包名 | `io.github.vannifan.idmix`（包名不能含 `-`） |

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

---

## 第一步：生成 User Token（Central Portal）

1. 登录 [central.sonatype.com](https://central.sonatype.com)
2. 右上角头像 → **View Account** → **Generate User Token**
3. 复制 **Username** 和 **Password**（Password 只显示一次）

---

## 第二步：配置 GPG 签名（必须）

Maven Central 要求所有构件 GPG 签名。

### Windows 安装 Gpg4win

1. 下载安装 [Gpg4win](https://www.gpg4win.org/)
2. 生成密钥：

```powershell
gpg --full-generate-key
# 选 RSA and RSA, 4096 bits, 填姓名和邮箱 web.cn@msn.com
```

3. 查看密钥 ID：

```powershell
gpg --list-secret-keys --keyid-format LONG
# sec   rsa4096/ABCD1234EFGH5678 ...
#                              ^^^^^^^^^^^^^^^^ 这是 KEY_ID
```

4. **上传公钥到 keyserver**（Central 会从公网 keyserver 验证签名，**无需**在 Portal 手动登记）：

```powershell
gpg --keyserver keyserver.ubuntu.com --send-keys YOUR_KEY_ID
```

> 旧版 OSSRH 有 “Register GPG Key” 页面；**新版 Central Portal 通常不需要**，只要公钥已在 keyserver 上、且 `mvn deploy -Prelease` 能成功签名即可。

---

## 第三步：配置 Maven settings.xml（**当前报错原因**）

错误 `server is null` = Maven **找不到** `<server id="central">` 凭据。

1. 打开 [central.sonatype.com/usertoken](https://central.sonatype.com/usertoken)
2. 点击 **Generate User Token**，复制 **Username** 和 **Password**（关闭弹窗后 Password 不可再查看）
3. 创建或编辑 **`C:\Users\webcn\.m2\settings.xml`**（不是项目里的 example 文件）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0 https://maven.apache.org/xsd/settings-1.2.0.xsd">
    <servers>
        <server>
            <id>central</id>
            <username>这里填Token的Username</username>
            <password>这里填Token的Password</password>
        </server>
    </servers>
</settings>
```

**自检**（PowerShell）：

```powershell
Test-Path $env:USERPROFILE\.m2\settings.xml
Select-String -Path $env:USERPROFILE\.m2\settings.xml -Pattern '<id>central</id>'
```

必须能看到 `<id>central</id>`，且 username/password 是 **User Token**，不是 npm/PyPI 的密码，也不是 GitHub 密码。

---

## 第四步：构建并发布

```powershell
cd D:\vanni\idmix\java

# 先跑测试
mvn test

# 发布（会提示输入 GPG 密钥口令）
mvn clean deploy -Prelease
```

成功后会：

1. 上传 bundle 到 Central Portal
2. 自动校验（`autoPublish=true`）
3. 在 [Deployments](https://central.sonatype.com/publishing/deployments) 查看状态

若未开 `autoPublish`，校验通过后需手动点 **Publish**。

---

## 常见问题

| 问题 | 处理 |
|------|------|
| `server is null` / `Unable to get publisher server properties` | 未配置 `%USERPROFILE%\.m2\settings.xml` 中的 `<server id="central">` + User Token |
| `gpg: signing failed` | 确认 Gpg4win 已安装，`gpg --list-secret-keys` 有密钥 |
| namespace 不匹配 | groupId 必须是 `io.github.vanni-fan` |
| 版本已存在 | 修改 `pom.xml` 中 `<version>` 后重新 deploy |

---

## 同步到 Maven Central 索引

发布后约 **10–30 分钟** 可在 [search.maven.org](https://search.maven.org/) 搜索 `io.github.vanni-fan idmix`。
