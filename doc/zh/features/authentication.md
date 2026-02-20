# 认证

Bonds 支持多种认证方式，从传统密码登录到现代通行密钥。

## 密码 + JWT

默认认证流程：

1. 使用邮箱和密码注册
2. 登录返回 JWT 令牌
3. 令牌通过 `Authorization: Bearer <token>` 请求头发送
4. 令牌 24 小时后过期（可通过 `JWT_EXPIRY_HRS` 配置）
5. 7 天内可刷新令牌（可通过 `JWT_REFRESH_HRS` 配置）

## 两步验证（TOTP）

使用基于 TOTP 的两步验证增强安全性：

1. **启用** — 前往设置 → 安全 → 启用两步验证
2. **扫描二维码** — 使用任意认证器应用（Google Authenticator、Authy、1Password 等）
3. **保存恢复码** — 生成 8 个一次性恢复码，请妥善保管
4. **确认** — 输入一个 TOTP 码完成激活

### 启用两步验证后的登录

启用两步验证后，登录分为两步：

1. 输入邮箱 + 密码 → 服务器返回 `requires_two_factor: true` + 临时令牌
2. 输入 TOTP 码（或恢复码）→ 服务器返回正式 JWT

### 恢复码

启用两步验证时生成 8 个随机 8 字符码。每个码只能使用一次。当你无法访问认证器应用时使用。

## WebAuthn / FIDO2

Bonds 支持通过 WebAuthn 实现无密码登录：

- **硬件密钥** — YubiKey、Titan Security Key 等
- **生物识别** — Touch ID、Face ID、Windows Hello
- **通行密钥** — iCloud 钥匙串、Android 通行密钥

### 设置

1. 前往设置 → 安全 → 注册新的通行密钥
2. 按照浏览器提示创建凭证
3. 通行密钥已关联到你的账户

### 要求

- **必须使用 HTTPS**（开发环境的 `localhost` 除外）
- 在管理面板中配置 WebAuthn 设置：
  - **RP ID** — 你的域名（如 `bonds.example.com`）
  - **RP 显示名** — 认证时向用户展示的名称
  - **RP 来源** — 允许的来源（如 `https://bonds.example.com`）

## OAuth 登录

Bonds 支持以下单点登录：

| 提供商 | 配置 |
|--------|------|
| **GitHub** | OAuth App Client ID 和 Secret |
| **Google** | OAuth Client ID 和 Secret |

在管理面板中配置。启用后，登录页面会显示「使用 GitHub 登录」/「使用 Google 登录」按钮。

如果 OAuth 邮箱与已有 Bonds 账户匹配，会自动关联。

### OAuth 回调流程

```
GET /api/auth/:provider → 重定向到 OAuth 提供商
GET /api/auth/:provider/callback → JWT → 重定向到 /auth/callback?token=xxx
```

## OIDC（OpenID Connect）

Bonds 支持通用 OIDC 提供商，用于企业 SSO：

| 设置 | 说明 |
|------|------|
| **Client ID** | OIDC 客户端 ID |
| **Client Secret** | OIDC 客户端密钥 |
| **Discovery URL** | 提供商的 `.well-known/openid-configuration` 端点 |
| **显示名** | 登录页按钮标签（默认：「SSO」） |

兼容 Authentik、Keycloak、Azure AD、Okta 等 OIDC 标准提供商。在管理面板中配置。
