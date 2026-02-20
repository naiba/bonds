# Authentication

Bonds supports multiple authentication methods, from traditional password login to modern passkeys.

## Password + JWT

The default authentication flow:

1. Register with email and password
2. Login returns a JWT token
3. Token is sent in the `Authorization: Bearer <token>` header
4. Tokens expire after 24 hours (configurable via `JWT_EXPIRY_HRS`)
5. Tokens can be refreshed within 7 days (configurable via `JWT_REFRESH_HRS`)

## Two-Factor Authentication (TOTP)

Add an extra layer of security with TOTP-based 2FA:

1. **Enable** — Go to Settings → Security → Enable 2FA
2. **Scan QR code** — Use any authenticator app (Google Authenticator, Authy, 1Password, etc.)
3. **Save recovery codes** — 8 one-time-use recovery codes are generated. Store them safely.
4. **Confirm** — Enter a TOTP code to activate

### Login with 2FA

When 2FA is enabled, login is a two-step process:

1. Enter email + password → server returns `requires_two_factor: true` + a temporary token
2. Enter TOTP code (or a recovery code) → server returns the full JWT

### Recovery Codes

8 random 8-character codes are generated when 2FA is enabled. Each code can only be used once. Use them if you lose access to your authenticator app.

## WebAuthn / FIDO2

Bonds supports passwordless login via WebAuthn:

- **Hardware keys** — YubiKey, Titan Security Key, etc.
- **Biometrics** — Touch ID, Face ID, Windows Hello
- **Passkeys** — iCloud Keychain, Android passkeys

### Setup

1. Go to Settings → Security → Register a new passkey
2. Follow your browser's prompt to create a credential
3. The passkey is now linked to your account

### Requirements

- HTTPS is **required** (except `localhost` for development)
- Configure WebAuthn settings in the admin panel:
  - **RP ID** — Your domain (e.g., `bonds.example.com`)
  - **RP Display Name** — Shown to users during authentication
  - **RP Origins** — Allowed origins (e.g., `https://bonds.example.com`)

## OAuth Login

Bonds supports single sign-on via:

| Provider | Configuration |
|----------|--------------|
| **GitHub** | OAuth App client ID and secret |
| **Google** | OAuth client ID and secret |

Configure these in the admin panel. When enabled, "Login with GitHub" / "Login with Google" buttons appear on the login page.

If the OAuth email matches an existing Bonds account, the accounts are automatically linked.

### OAuth Callback Flow

```
GET /api/auth/:provider → Redirect to OAuth provider
GET /api/auth/:provider/callback → JWT → Redirect to /auth/callback?token=xxx
```

## OIDC (OpenID Connect)

Bonds supports generic OIDC providers for enterprise SSO:

| Setting | Description |
|---------|-------------|
| **Client ID** | OIDC client ID |
| **Client Secret** | OIDC client secret |
| **Discovery URL** | Provider's `.well-known/openid-configuration` endpoint |
| **Display Name** | Button label on login page (default: "SSO") |

Compatible with Authentik, Keycloak, Azure AD, Okta, and other OIDC-compliant providers. Configure in the admin panel.
