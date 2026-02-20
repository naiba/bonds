# Admin & Settings

Bonds includes an admin panel for system-wide configuration and user management.

## Admin Panel

Users with admin privileges can access the admin panel to configure:

- **System settings** — All application-level configuration stored in the database
- **User management** — View and manage all registered users
- **Backup** — Configure automatic backups and trigger manual backups

## System Settings

As of v0.2.0, most configuration has moved from environment variables to database-backed settings. The admin panel provides a web UI to configure:

| Category | Settings |
|----------|----------|
| **General** | Application name, URL, announcement banner |
| **SMTP** | Mail server host, port, credentials, sender address |
| **OAuth** | GitHub and Google OAuth client credentials |
| **OIDC** | OpenID Connect provider for SSO |
| **WebAuthn** | Relying Party ID, display name, allowed origins |
| **Telegram** | Bot token for notifications |
| **Geocoding** | Provider selection and API key |
| **Backup** | Cron schedule, retention period |

::: tip
On first startup, these settings are seeded from environment variables if present. After that, changes are made exclusively through the admin panel.
:::

## Personalization {#personalization}

Account owners can customize many aspects of Bonds through the personalization settings at `/api/settings/personalize/:entity`:

| Entity | What You Can Customize |
|--------|----------------------|
| `genders` | Gender options |
| `pronouns` | Pronoun options |
| `address-types` | Address type labels |
| `pet-categories` | Pet category types |
| `contact-info-types` | Contact information types (email, phone, etc.) |
| `call-reasons` | Call reason categories |
| `religions` | Religion options |
| `gift-occasions` | Gift occasion types |
| `gift-states` | Gift tracking states |
| `group-types` | Contact group types |
| `post-templates` | Journal post templates |
| `relationship-types` | Relationship type definitions |
| `templates` | Contact page templates |
| `modules` | Module configuration |
| `currencies` | Currency preferences |

Some built-in items (like email and phone contact information types) cannot be deleted.

## User Invitations

Invite others to your account via email:

1. Go to Settings → Invitations
2. Enter the recipient's email and choose a permission level
3. An email is sent with an invitation link (valid for 7 days)
4. The recipient creates an account and is automatically linked to your account

Permission levels: **Manager** (100), **Editor** (200), **Viewer** (300).

## Backup System

Bonds includes an automatic backup system:

- **Scheduled backups** — Configure a cron schedule in the admin panel (6-field format with seconds, e.g., `0 0 2 * * *` for 2 AM daily)
- **Manual backups** — Trigger a backup on demand from the admin panel
- **Retention** — Old backups are auto-cleaned after a configurable number of days (default: 30)
- **Storage** — Backups are stored in the directory configured by `BACKUP_DIR` (default: `data/backups`)
