# Configuration

Bonds uses a hybrid configuration model: **infrastructure settings** are configured via environment variables, while **application settings** are managed safely through the admin panel in the web UI.

## Environment Variables

Copy the example file to get started:

```bash
cp server/.env.example server/.env
```

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG` | `false` | Enable debug mode: request logging, SQL logging, Swagger UI (default on) |
| `JWT_SECRET` | — | **Required in production.** Generate a 256-bit key with `openssl rand -hex 32`, then persist and reuse it across restarts. |
| `SETTINGS_ENC_KEY` | _(empty)_ | Optional. Enables AES-256-GCM encryption-at-rest for sensitive system settings (SMTP password, OAuth client secrets, geocoding API keys). See [Encrypting Sensitive Settings](#encrypting-sensitive-settings) below. |
| `SERVER_PORT` | `8080` | Port the server listens on |
| `SERVER_HOST` | `0.0.0.0` | Host address the server binds to |
| `DB_DRIVER` | `sqlite` | Database driver: `sqlite` or `postgres` |
| `DB_DSN` | `bonds.db` | Database connection string |
| `APP_ENV` | `development` | Set to `production` for production use |

### Storage & Search

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_UPLOAD_DIR` | `uploads` | Directory for uploaded files |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Full-text search index directory |
| `BACKUP_DIR` | `data/backups` | Directory for automatic backups |

### Database Connection

**SQLite** (default, zero configuration):
```bash
DB_DRIVER=sqlite
DB_DSN=bonds.db
```

**PostgreSQL**:
```bash
DB_DRIVER=postgres
DB_DSN="host=localhost port=5432 user=bonds password=secret dbname=bonds sslmode=disable"
```

## Admin Settings (Web UI)

Most application settings are configured through the **Admin Settings** panel, accessible to users with admin privileges. These include:

- **General**: Application name, public URL, announcement banner.
- **Authentication**: Password login toggle, registration toggle.
- **JWT**: Token expiry, refresh window.
- **SMTP**: Mail server host, port, optional credentials, sender address. Leave both username and password empty to skip SMTP AUTH for unauthenticated relays.
- **OAuth**: GitHub and Google OAuth client credentials.
- **OIDC**: OpenID Connect provider for SSO (Authentik, Keycloak, etc.).
- **WebAuthn**: Relying Party configuration for passkey authentication.
- **Telegram**: Bot token for Telegram notifications.
- **Geocoding**: Active provider, privacy precision, per-provider credentials, and self-hosted Photon URL.
- **Storage**: Max upload size for files and documents (configured inside UI, not via env vars).
- **Backup**: Cron schedule, retention period for automatic backups.
- **Swagger**: Enable or disable API documentation UI independently of debug mode.

::: tip Migration from Environment Variables
On first startup, Bonds seeds these admin settings from environment variables if present. After that, all changes are made through the admin panel. Environment variables for these settings are only used as initial seed values.

Geocoding provider credentials and self-hosted URLs are configured only through the admin panel; they are not imported from a generic environment API key.
:::

## Encrypting Sensitive Settings

By default, sensitive system settings (SMTP password, OAuth client secrets, geocoding API keys) are stored as plaintext in the database. Anyone who can read the database file or a backup archive recovers every credential the deployment uses.

Set `SETTINGS_ENC_KEY` to enable AES-256-GCM encryption-at-rest for these values:

```bash
# Generate a random key once and store it alongside other secrets
SETTINGS_ENC_KEY="$(openssl rand -hex 32)"
```

Behaviour:

- The key is **never written to the database**, so a stolen DB backup alone cannot recover plaintext.
- Encrypted rows are tagged with the prefix `enc:v1:`. Already-encrypted rows are detected and skipped on re-encryption.
- On startup, any pre-existing plaintext rows in the secret-key whitelist are **automatically migrated** to ciphertext (idempotent).
- Leave the variable empty to keep the legacy plaintext behaviour. Single-instance deployments are not forced to migrate.
- The admin **GET /api/admin/settings** endpoint always redacts secret keys to `***`. Submitting `***` on **PUT** keeps the existing value untouched, so admin UIs can round-trip non-secret edits safely.

Currently encrypted at rest when the key is set:

| Field | Storage |
|-------|---------|
| `system_settings.value` for `smtp.password` and any `secret.*` key | AES-256-GCM |
| `geocoding_provider_configs.config` (one structured config per provider) | AES-256-GCM |
| `oauth_providers.client_secret` (GitHub, Google, GitLab, Discord, OIDC) | AES-256-GCM |

::: warning Losing the key
If you set `SETTINGS_ENC_KEY` and then lose it, encrypted secrets are unrecoverable. Treat this key like `JWT_SECRET` and back it up out-of-band.
:::

## Production Checklist

1. **Set `JWT_SECRET`**: Run `export JWT_SECRET="$(openssl rand -hex 32)"` once, store the 256-bit value in a protected environment or secret store, and reuse it across restarts. Plan rotation: it invalidates existing sessions and can require DAV subscription credentials to be entered again because their encryption derives from this secret.
2. **Set `SETTINGS_ENC_KEY`**: Recommended for production. Encrypts SMTP/OAuth/geocoding credentials at rest.
3. **Set `APP_ENV=production`**: Disables debug features.
4. **Set `APP_URL`**: Your public URL, used in emails and OAuth callbacks.
5. **Configure SMTP**: Required for email notifications and invitations.
6. **Use HTTPS**: Required for WebAuthn; recommended for all deployments.
7. **Backup**: Configure automatic backups via the admin panel.

## Docker Environment Example

```yaml
services:
  bonds:
    image: ghcr.io/naiba/bonds:latest
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=${JWT_SECRET:?Set a persisted 256-bit JWT secret before startup}
      - SETTINGS_ENC_KEY=${SETTINGS_ENC_KEY:?Set a persisted settings encryption key before startup}
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
