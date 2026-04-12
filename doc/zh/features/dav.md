# CardDAV / CalDAV

Bonds 内置 CardDAV 和 CalDAV 服务器，可以与 Apple 通讯录、Thunderbird、GNOME Contacts 等 DAV 兼容客户端同步联系人和日历。

## 端点

| 协议 | URL | 说明 |
|------|-----|------|
| CardDAV | `/dav/` | 联系人同步 |
| CalDAV | `/dav/` | 日历同步（重要日期、任务） |
| 发现 | `/.well-known/carddav` | 自动发现重定向 |
| 发现 | `/.well-known/caldav` | 自动发现重定向 |

## 认证

DAV 客户端使用 **HTTP Basic Auth**（非 JWT），因为大多数 DAV 客户端不支持 Token 认证。

| 场景 | 凭据 |
|------|------|
| 两步验证**未启用** | 邮箱 + 密码，或邮箱 + [个人访问令牌](/zh/features/more#个人访问令牌) |
| 两步验证**已启用** | 邮箱 + 个人访问令牌 **仅此方式**（密码被禁止） |

::: warning
启用两步验证后，使用密码的 DAV 客户端将停止同步。请前往 **设置 → API 令牌** 创建令牌后更新客户端配置。
:::

## 同步内容

### CardDAV（联系人 → vCard 4.0）

| Bonds 字段 | vCard 属性 |
|-----------|-----------|
| 名 + 姓 | `FN`、`N` |
| 电话号码 | `TEL` |
| 邮箱地址 | `EMAIL` |
| 地址 | `ADR` |

### CalDAV

| Bonds 实体 | iCal 类型 | 备注 |
|-----------|----------|------|
| 重要日期 | `VEVENT` | 带 `RRULE=YEARLY` 的周期事件 |
| 任务 | `VTODO` | 任务截止日期和状态 |

## 客户端配置

### Apple 通讯录 / 日历（macOS / iOS）

1. 前往 **设置 → 账户 → 添加账户 → 其他**
2. 选择 **添加 CardDAV 账户** 或 **添加 CalDAV 账户**
3. 输入：
   - 服务器：`https://your-bonds-domain.com`
   - 用户名：你的邮箱
   - 密码：你的密码（若启用了两步验证，则使用个人访问令牌）

Well-known URL（`/.well-known/carddav`、`/.well-known/caldav`）支持自动发现。

### Thunderbird

1. 打开 **通讯录 → 新建 → CardDAV 通讯录**
2. 输入 URL：`https://your-bonds-domain.com/dav/`
3. 使用 Bonds 账户认证

## ETag

Bonds 使用 `UpdatedAt` 时间戳（Unix 纪元）作为所有 DAV 资源的 ETag。客户端通过 ETag 检测变更并高效同步。
