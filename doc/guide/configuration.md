# Configuration

Bonds uses a hybrid configuration model: **infrastructure settings** are configured via environment variables, while **application settings** are managed through the admin panel in the web UI.

## Environment Variables

Copy the example file to get started:

```bash
cp server/.env.example server/.env
```

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG` | `false` | Enable debug mode: request logging, SQL logging, Swagger UI (default on) |
| `JWT_SECRET` | — | **Required in production.** Signing key for auth tokens |
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

**SQLite** (default — zero configuration):
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

- **SMTP** — Mail server settings for sending notifications and invitations
- **OAuth** — GitHub and Google OAuth client credentials
- **OIDC** — OpenID Connect provider for SSO (Authentik, Keycloak, etc.)
- **WebAuthn** — Relying Party configuration for passkey authentication
- **Telegram** — Bot token for Telegram notifications
- **Geocoding** — Provider and API key for address geocoding
- **Announcement** — Global banner text displayed to all users
- **Backup** — Cron schedule, retention period for automatic backups
 **Swagger** — Enable/disable API documentation UI independently of debug mode

::: tip Migration from Environment Variables
On first startup, Bonds seeds these admin settings from environment variables if present. After that, all changes are made through the admin panel. Environment variables for these settings are only used as initial seed values.
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
- Encrypted rows are tagged with the prefix `enc:v1:` — already-encrypted rows are detected and skipped on re-encryption.
- On startup, any pre-existing plaintext rows in the secret-key whitelist are **automatically migrated** to ciphertext (idempotent).
- Leave the variable empty to keep the legacy plaintext behaviour. Single-instance deployments are not forced to migrate.
- The admin **GET /api/admin/settings** endpoint always redacts secret keys to `***`. Submitting `***` on **PUT** keeps the existing value untouched, so admin UIs can round-trip non-secret edits safely.

Currently encrypted at rest when the key is set:

| Field | Storage |
|-------|---------|
| `system_settings.value` for `smtp.password`, `geocoding.api_key`, and any `secret.*` key | AES-256-GCM |
| `oauth_providers.client_secret` (GitHub, Google, GitLab, Discord, OIDC) | AES-256-GCM |

::: warning Losing the key
If you set `SETTINGS_ENC_KEY` and then lose it, encrypted secrets are unrecoverable. Treat this key like `JWT_SECRET` — back it up out-of-band.
:::

## Production Checklist

1. **Set `JWT_SECRET`** — Use a strong, random string (32+ characters)
2. **Set `SETTINGS_ENC_KEY`** — Recommended for production. Encrypts SMTP/OAuth/geocoding credentials at rest
3. **Set `APP_ENV=production`** — Disables debug features
4. **Set `APP_URL`** — Your public URL (used in emails, OAuth callbacks)
5. **Configure SMTP** — Required for email notifications and invitations
6. **Use HTTPS** — Required for WebAuthn; recommended for all deployments
7. **Backup** — Configure automatic backups via the admin panel

## Docker Environment Example

```yaml
services:
  bonds:
    image: ghcr.io/naiba/bonds:latest
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=change-me-to-a-random-string
      - SETTINGS_ENC_KEY=change-me-to-another-random-string
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
