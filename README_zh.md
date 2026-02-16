# Bonds

[![Test](https://github.com/naiba/bonds/actions/workflows/test.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/test.yml)
[![Release](https://github.com/naiba/bonds/actions/workflows/release.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/naiba/bonds)](https://github.com/naiba/bonds/releases)

> [English](README.md)

现代化的社区驱动个人关系管理器 — 受 [Monica](https://github.com/monicahq/monica) 启发，使用 **Go** 和 **React** 全新重写。

## 为什么做 Bonds？

Monica 是一个拥有 24k+ star 的优秀开源个人 CRM。但作为一个由小团队业余维护的项目（[他们自己的话](https://github.com/monicahq/monica/issues/6626)），开发速度已经明显放缓 — 700+ 未关闭 issue，响应能力有限。

**Bonds** 就是下一代版本：

- **快速轻量** — 单个二进制文件，毫秒级启动，内存占用极低
- **部署简单** — 一个二进制 + SQLite 即可运行，无需 PHP/Composer/Node 运行时
- **现代界面** — React 19 + TypeScript，流畅的 SPA 体验
- **测试完善** — 347 后端测试、54 前端测试、6 个 E2E 测试文件
- **社区优先** — 为接受贡献和快速迭代而生

> **致谢**：Bonds 站在 [@djaiss](https://github.com/djaiss)、[@asbiin](https://github.com/asbiin) 以及整个 Monica 社区的肩膀上。原版 Monica 仍以 AGPL-3.0 许可证在 [monicahq/monica](https://github.com/monicahq/monica) 持续提供。

## 功能特性

- **联系人管理** — 笔记、任务、提醒、礼物、债务、活动、人生事件、宠物等完整生命周期管理
- **多 Vault** — 数据隔离 + 基于角色的权限控制（管理者 / 编辑者 / 查看者）
- **提醒系统** — 一次性和周期性（每周/每月/每年），支持邮件和 Telegram 通知
- **全文搜索** — 基于 Bleve 的中英文混合搜索，覆盖联系人和笔记
- **CardDAV / CalDAV** — 与 Apple 通讯录、Thunderbird 等 DAV 客户端同步联系人和日历
- **vCard 导入导出** — 批量导入 `.vcf` 文件，导出单个或全部联系人
- **文件上传** — 照片和文档附加到联系人，自动生成首字母头像
- **两步验证 (TOTP)** — 基于 TOTP 的双因素认证 + 恢复码
- **WebAuthn / FIDO2** — 通行密钥登录（硬件密钥、生物识别）
- **OAuth 登录** — GitHub 和 Google 单点登录
- **用户邀请** — 通过邮件邀请他人加入账户，支持权限级别
- **审计日志** — 联系人所有变更的操作记录
- **地理编码** — 通过 Nominatim（免费）或 LocationIQ 获取地址坐标
- **Telegram 通知** — 通过 Telegram Bot 发送提醒
- **国际化** — 英文和中文，前后端全覆盖

## 快速开始

### 方式一：Docker（推荐）

```bash
# 下载 docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# 启动
docker compose up -d
```

打开 **http://localhost:8080**，注册账号即可使用。

自定义配置，编辑 `docker-compose.yml`：

```yaml
environment:
  - JWT_SECRET=你的密钥         # ⚠️ Change this!
```

### 方式二：下载预编译版本

从 [GitHub Releases](https://github.com/naiba/bonds/releases) 下载最新版本，然后：

```bash
# 设置 JWT 密钥并运行
export JWT_SECRET=你的密钥
./bonds-server
```

服务器会在 **http://localhost:8080** 启动，自带前端界面和 SQLite 数据库。

### 方式三：从源码构建

**环境要求**：Go 1.25+、[Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# 安装依赖
make setup

# 构建单二进制文件（内嵌前端）
make build-all

# 运行
export JWT_SECRET=你的密钥
./server/bin/bonds-server
```

## 配置

Bonds 通过环境变量配置。复制示例文件开始：

```bash
cp server/.env.example server/.env
```

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DEBUG` | `false` | 调试模式：启用 Echo 请求日志、GORM SQL 日志、Swagger UI |
| `JWT_SECRET` | — | **生产环境必填。** 认证令牌签名密钥 |
| `SERVER_PORT` | `8080` | 服务端口 |
| `SERVER_HOST` | `0.0.0.0` | 服务器监听地址 |
| `DB_DSN` | `bonds.db` | SQLite 数据库文件路径 |
| `DB_DRIVER` | `sqlite` | 数据库驱动 |
| `APP_NAME` | `Bonds` | 应用名称（用于邮件、WebAuthn 等） |
| `APP_ENV` | `development` | 生产环境设置为 `production` |
| `APP_URL` | `http://localhost:8080` | 公开 URL（用于邮件链接和 OAuth 回调） |
| `JWT_EXPIRY_HRS` | `24` | JWT 令牌过期时间（小时） |
| `JWT_REFRESH_HRS` | `168` | JWT 刷新窗口（小时，默认 7 天） |
| `SMTP_HOST` | — | SMTP 邮件服务器 |
| `SMTP_PORT` | `587` | SMTP 端口 |
| `SMTP_USERNAME` | — | SMTP 用户名 |
| `SMTP_PASSWORD` | — | SMTP 密码 |
| `SMTP_FROM` | — | 发件人邮箱 |
| `STORAGE_UPLOAD_DIR` | `uploads` | 文件上传目录 |
| `STORAGE_MAX_SIZE` | `10485760` | 最大上传大小（字节，默认 10 MB） |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token（用于发送通知） |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | 全文搜索索引目录 |
| `OAUTH_GITHUB_KEY` | — | GitHub OAuth App Client ID |
| `OAUTH_GITHUB_SECRET` | — | GitHub OAuth App Client Secret |
| `OAUTH_GOOGLE_KEY` | — | Google OAuth Client ID |
| `OAUTH_GOOGLE_SECRET` | — | Google OAuth Client Secret |
| `GEOCODING_PROVIDER` | `nominatim` | 地理编码服务（`nominatim` 或 `locationiq`） |
| `GEOCODING_API_KEY` | — | LocationIQ API Key |
| `WEBAUTHN_RP_ID` | — | WebAuthn Relying Party ID（如 `bonds.example.com`） |
| `WEBAUTHN_RP_DISPLAY_NAME` | `Bonds` | WebAuthn 显示名称 |
| `WEBAUTHN_RP_ORIGINS` | — | WebAuthn 允许的来源（逗号分隔） |
| `ANNOUNCEMENT` | — | 公告横幅内容（显示给所有用户） |

## 开发

```bash
# 安装依赖
make setup

# 生成 API 客户端（首次构建前必须执行）
make gen-api

# 同时启动前后端开发模式
make dev
```

Go 后端运行在 `:8080`，Vite 开发服务器运行在 `:5173`。前端自动将 API 请求代理到后端。

### 代码生成管线

前端 TypeScript API 客户端从后端 OpenAPI/Swagger 规范**自动生成**。生成的文件不纳入 git 版本控制 — 在 CI 和开发过程中按需重新生成。

```
Go handlers（swag 注解）
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api（或 bun run gen:api）
web/src/api/generated/   ← gitignored，按需重新生成
    ↓
web/src/api/index.ts     ← 入口文件，导入生成的模块 + 类型别名
```

修改后端 API（handlers、DTOs、routes）后，运行：

```bash
make gen-api       # 重新生成 swagger.json + TypeScript API 客户端
```

### 常用命令

```bash
make dev           # 同时启动前后端开发模式
make build         # 分别构建后端和前端
make build-all     # 构建内嵌前端的单二进制文件
make test          # 运行所有测试（后端 + 前端）
make test-e2e      # 运行端到端测试（Playwright）
make lint          # 运行代码检查（go vet + eslint）
make swagger       # 仅重新生成 Swagger/OpenAPI 文档
make gen-api       # 重新生成 Swagger 文档 + TypeScript API 客户端
make clean         # 清理所有构建产物 + 生成文件
make setup         # 安装所有依赖
```

### API 文档

Bonds 提供自动生成的 OpenAPI/Swagger 文档，覆盖全部 286 个 API 端点。

启用调试模式后访问 Swagger UI：

```bash
DEBUG=true ./bonds-server
# 打开 http://localhost:8080/swagger/index.html
```

> Swagger UI 仅在 `DEBUG=true` 时可用，生产环境不暴露。

## 与 Monica 的关系

Bonds 是受 [Monica](https://github.com/monicahq/monica)（AGPL-3.0）启发的全新重写项目。它使用完全不同的技术栈（Go + React 替代 PHP/Laravel + Vue）重新实现了 Monica 的数据模型和功能集。不包含原项目的任何代码。

## 许可证

[AGPL-3.0](LICENSE) — 与原版 Monica 项目相同的许可证。
