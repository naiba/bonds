# 配置

Bonds 采用混合配置模式：**基础设施设置**通过环境变量配置，**应用设置**通过 Web 管理面板管理。

## 环境变量

复制示例文件开始：

```bash
cp server/.env.example server/.env
```

### 核心设置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DEBUG` | `false` | 调试模式：启用请求日志、SQL 日志、Swagger UI（默认开启） |
| `JWT_SECRET` | — | **生产环境必填。** 认证令牌签名密钥 |
| `SERVER_PORT` | `8080` | 服务端口 |
| `SERVER_HOST` | `0.0.0.0` | 监听地址 |
| `DB_DRIVER` | `sqlite` | 数据库驱动：`sqlite` 或 `postgres` |
| `DB_DSN` | `bonds.db` | 数据库连接字符串 |
| `APP_ENV` | `development` | 生产环境设置为 `production` |

### 存储与搜索

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `STORAGE_UPLOAD_DIR` | `uploads` | 文件上传目录 |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | 全文搜索索引目录 |
| `BACKUP_DIR` | `data/backups` | 自动备份目录 |

### 数据库连接

**SQLite**（默认 — 零配置）：
```bash
DB_DRIVER=sqlite
DB_DSN=bonds.db
```

**PostgreSQL**：
```bash
DB_DRIVER=postgres
DB_DSN="host=localhost port=5432 user=bonds password=secret dbname=bonds sslmode=disable"
```

## 管理面板设置（Web UI）

大部分应用设置通过 **管理面板** 配置，仅管理员可访问：

- **SMTP** — 邮件服务器设置，用于发送通知和邀请
- **OAuth** — GitHub 和 Google OAuth 客户端凭证
- **OIDC** — OpenID Connect 提供商，用于企业 SSO（Authentik、Keycloak 等）
- **WebAuthn** — 通行密钥认证的 Relying Party 配置
- **Telegram** — Bot Token，用于 Telegram 通知
- **地理编码** — 服务提供商和 API Key
- **公告** — 全局公告横幅文字
- **备份** — Cron 调度、保留天数
 **Swagger** — 独立于调试模式启用/禁用 API 文档界面

::: tip 从环境变量迁移
首次启动时，Bonds 会从环境变量中读取这些设置作为初始值写入数据库。之后所有修改都通过管理面板进行。环境变量仅作为初始种子值使用。
:::

## 生产环境清单

1. **设置 `JWT_SECRET`** — 使用强随机字符串（32+ 字符）
2. **设置 `APP_ENV=production`** — 禁用调试功能
3. **设置 `APP_URL`** — 你的公开 URL（用于邮件链接和 OAuth 回调）
4. **配置 SMTP** — 邮件通知和邀请功能必需
5. **使用 HTTPS** — WebAuthn 必需，所有部署均推荐
6. **配置备份** — 在管理面板中设置自动备份

## Docker 环境示例

```yaml
services:
  bonds:
    image: ghcr.io/naiba/bonds:latest
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=修改为随机字符串
      - APP_ENV=production
      - APP_URL=https://bonds.example.com
      - DB_DSN=/data/bonds.db
      - STORAGE_UPLOAD_DIR=/data/uploads
      - BLEVE_INDEX_PATH=/data/bonds.bleve
      - BACKUP_DIR=/data/backups
    volumes:
      - bonds-data:/data

volumes:
  bonds-data:
```
