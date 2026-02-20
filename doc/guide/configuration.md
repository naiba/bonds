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
| `DEBUG` | `false` | Enable debug mode: request logging, SQL logging, Swagger UI |
| `JWT_SECRET` | — | **Required in production.** Signing key for auth tokens |
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

::: tip Migration from Environment Variables
On first startup, Bonds seeds these admin settings from environment variables if present. After that, all changes are made through the admin panel. Environment variables for these settings are only used as initial seed values.
:::

## Production Checklist

1. **Set `JWT_SECRET`** — Use a strong, random string (32+ characters)
2. **Set `APP_ENV=production`** — Disables debug features
3. **Set `APP_URL`** — Your public URL (used in emails, OAuth callbacks)
4. **Configure SMTP** — Required for email notifications and invitations
5. **Use HTTPS** — Required for WebAuthn; recommended for all deployments
6. **Backup** — Configure automatic backups via the admin panel

## Docker Environment Example

```yaml
services:
  bonds:
    image: ghcr.io/naiba/bonds:latest
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=change-me-to-a-random-string
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
