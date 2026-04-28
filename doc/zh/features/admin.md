# 管理面板

Bonds 包含管理面板，用于系统级配置和用户管理。

## 成为管理员

在新 Bonds 实例上**第一个注册的用户**会自动获得实例管理员权限。之后，现有管理员可以在管理面板的**用户管理**页面中提升其他用户为管理员。

## 管理面板

拥有管理员权限的用户可以访问管理面板来配置：

- **系统设置** — 所有存储在数据库中的应用级配置
- **用户管理** — 查看和管理所有注册用户
- **备份** — 配置自动备份和触发手动备份

## 系统设置

从 v0.2.0 开始，大部分配置已从环境变量迁移到数据库存储。管理面板提供 Web UI 来配置：

| 类别 | 设置项 |
|------|--------|
| **常规** | 应用名称、URL、公告横幅 |
| **SMTP** | 邮件服务器地址、端口、凭证、发件人 |
| **OAuth** | GitHub 和 Google OAuth 客户端凭证 |
| **OIDC** | OpenID Connect 提供商（SSO） |
| **WebAuthn** | Relying Party ID、显示名、允许的来源 |
| **Telegram** | Bot Token（用于通知） |
| **地理编码** | 服务提供商和 API Key |
| **备份** | Cron 调度、保留天数 |
| **Swagger** | 启用/禁用 API 文档界面 |

::: tip
首次启动时，这些设置会从环境变量中读取作为初始值。之后所有修改都通过管理面板进行。
:::

### 静态加密

当配置了 `SETTINGS_ENC_KEY` 时（参见[配置 → 加密敏感设置](/zh/guide/configuration#加密敏感设置)），以下字段会以 AES-256-GCM 加密存储在数据库中：

- **system_settings** 中的 `smtp.password`、`geocoding.api_key` 以及任何 `secret.*` 键
- **oauth_providers** 中所有 `client_secret`（GitHub、Google、GitLab、Discord、OIDC）

无论是否启用加密，Admin **GET /admin/settings** 始终把敏感值脱敏为 `***`，管理员浏览器和审计日志看不到明文凭证。提交 `***` 进行更新表示保留原值，UI 可以安全地往返编辑非敏感字段而不会清空密钥。

启用密钥后首次启动会自动将已有明文行迁移为密文，迁移幂等可重复执行。

## 个性化设置 {#个性化设置}

账户所有者可通过个性化设置自定义 Bonds 的多个方面，API 路径为 `/api/settings/personalize/:entity`：

| 实体 | 可自定义内容 |
|------|-------------|
| `genders` | 性别选项 |
| `pronouns` | 代词选项 |
| `address-types` | 地址类型标签 |
| `pet-categories` | 宠物类别 |
| `contact-info-types` | 联系方式类型（邮箱、电话等） |
| `call-reasons` | 通话原因分类 |
| `religions` | 宗教选项 |
| `gift-occasions` | 礼物场合类型 |
| `gift-states` | 礼物追踪状态 |
| `group-types` | 联系人分组类型 |
| `post-templates` | 日记模板 |
| `relationship-types` | 关系类型定义 |
| `templates` | 联系人页面模板 |
| `modules` | 模块配置 |
| `currencies` | 货币偏好 |

部分内置项（如邮箱和电话联系方式类型）不可删除。

## 用户邀请

通过邮件邀请他人加入你的账户：

1. 前往设置 → 邀请
2. 输入收件人邮箱并选择权限级别
3. 系统发送包含邀请链接的邮件（7 天内有效）
4. 收件人创建账户并自动关联到你的账户

权限级别：**管理者**（100）、**编辑者**（200）、**查看者**（300）。

## 备份系统

Bonds 内置自动备份系统：

- **定时备份** — 在管理面板中配置 Cron 调度（6 字段格式含秒，如 `0 0 2 * * *` 表示每天凌晨 2 点）
- **手动备份** — 在管理面板中按需触发备份
- **保留策略** — 旧备份在可配置天数后自动清理（默认 30 天）
- **存储位置** — 备份存储在 `BACKUP_DIR` 配置的目录中（默认 `data/backups`）

## 定时任务（Cron）

提醒发送、CardDAV/CalDAV 同步以及自动备份都通过内置的 cron 调度器运行：

- 单进程 SQLite 部署开箱即用 — 每个任务在每个调度时刻最多执行一次。
- **多副本 PostgreSQL 部署**（如 Kubernetes Deployment `replicas: 2`、Docker Compose `deploy.replicas`、负载均衡的多个实例）同样安全：每个任务通过 `pg_try_advisory_xact_lock` + 对 `crons` 表的原子条件 `UPDATE` 加锁，两个副本即便在同一刻触发同一任务也不会同时执行。
- 副本崩溃不会导致任务卡死 — advisory lock 在持有事务结束时自动释放。

无需任何配置，调度器会根据数据库驱动自动选择正确的策略。
