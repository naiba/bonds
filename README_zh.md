# Bonds

[![Test](https://github.com/naiba/bonds/actions/workflows/test.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/test.yml)
[![Release](https://github.com/naiba/bonds/actions/workflows/release.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/naiba/bonds)](https://github.com/naiba/bonds/releases)

📖 [文档](https://naiba.github.io/bonds/zh/) | [English](README.md)

现代化的社区驱动个人关系管理器 — 受 [Monica](https://github.com/monicahq/monica) 启发，使用 **Go** 和 **React** 全新重写。

## 为什么做 Bonds？

Monica 是一个拥有 24k+ star 的优秀开源个人 CRM。但作为一个由小团队业余维护的项目（[他们自己的话](https://github.com/monicahq/monica/issues/6626)），开发速度已经明显放缓 — 700+ 未关闭 issue，响应能力有限。

**Bonds** 就是下一代版本：

- **快速轻量** — 单个二进制文件，毫秒级启动，内存占用极低
- **部署简单** — 一个二进制 + SQLite 即可运行，无需 PHP/Composer/Node 运行时
- **现代界面** — React 19 + TypeScript，流畅的 SPA 体验
- **测试完善** — 1014 后端测试、1​29 前端测试、174 个 E2E 测试
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

Bonds 采用**混合配置**方式：

- **环境变量** — 基础设施设置（数据库、服务器、安全）
- **管理后台** — 所有运行时设置（SMTP、OAuth、Telegram、WebAuthn 等）

首次启动时，环境变量会自动导入数据库。之后，请在 Web 界面的 **管理员 > 系统设置** 中管理设置。

```bash
cp server/.env.example server/.env
```

### 环境变量（必须）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DEBUG` | `false` | 调试模式：启用 Echo 请求日志、GORM SQL 日志、Swagger UI（默认开启） |
| `JWT_SECRET` | — | **生产环境必填。** 认证令牌签名密钥 |
| `SERVER_PORT` | `8080` | 服务端口 |
| `SERVER_HOST` | `0.0.0.0` | 服务器监听地址 |
| `DB_DSN` | `bonds.db` | 数据库连接字符串。SQLite：文件路径；PostgreSQL：`host=... port=5432 user=... password=... dbname=... sslmode=disable` |
| `DB_DRIVER` | `sqlite` | 数据库驱动（`sqlite` 或 `postgres`） |
| `APP_ENV` | `development` | 生产环境设置为 `production` |
| `STORAGE_UPLOAD_DIR` | `uploads` | 文件上传目录 |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | 全文搜索索引目录 |
| `BACKUP_DIR` | `data/backups` | 数据库备份存储目录 |

### 管理后台设置

以下设置在登录后通过 **管理员 > 系统设置** 页面管理：

- **应用** — 名称、URL、公告横幅
- **认证** — 密码登录开关、用户注册开关
- **JWT** — 令牌过期时间、刷新窗口
- **SMTP** — 主机、端口、用户名、密码、发件人邮箱
- **OAuth / OIDC** — GitHub、Google 和 OIDC/SSO 凭据
- **WebAuthn** — 依赖方 ID、显示名称、允许来源
- **Telegram** — Bot Token（通知推送）
- **地理编码** — 服务商（Nominatim/LocationIQ）、API Key
- **存储** — 最大上传大小
- **备份** — Cron 定时计划、保留天数
 **Swagger** — 启用/禁用 API 文档界面

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

Bonds 提供自动生成的 OpenAPI/Swagger 文档，覆盖全部 345 个 API 端点。

访问 Swagger UI，可通过调试模式或管理后台开关：
```bash
# 方式一：调试模式（Swagger 默认开启）
DEBUG=true ./bonds-server
# 方式二：管理后台开启（无需调试模式）
# 进入 管理员 > 系统设置 > Swagger > 启用
```

然后打开 http://localhost:8080/swagger/index.html

> Swagger UI 默认跟随 `DEBUG` 标志，也可在管理后台设置页面独立控制。

## 与 Monica 的关系

Bonds 是受 [Monica](https://github.com/monicahq/monica)（AGPL-3.0）启发的全新重写项目。它使用完全不同的技术栈（Go + React 替代 PHP/Laravel + Vue）重新实现了 Monica 的数据模型和功能集。不包含原项目的任何代码。

## 许可证

[Business Source License 1.1](LICENSE) (BSL 1.1) — 源码可见许可证，条款如下：

- **个人用户**：非商业使用完全免费
- **组织/企业**：商业使用需向 Licensor 购买许可
- **禁止行为**：转售、再许可、作为托管/管理服务提供
- **转换日期**：2030年2月17日 — 届时自动转为 [AGPL-3.0](LICENSE)（与原版 Monica 相同）

转换日期后，软件将以 AGPL-3.0 完全开源。
