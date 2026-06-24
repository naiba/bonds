# Admin & Settings

Bonds includes an admin panel for system-wide configuration and user management.

## Becoming an Admin

The first user to register on a new Bonds instance is automatically granted instance admin privileges. After that, existing admins can promote other users from the User Management page in the admin panel.

## Admin Panel

Users with admin privileges can access the admin panel to configure:

- **System settings**: All application-level configuration stored in the database.
- **User management**: View and manage all registered users.
- **Backup**: Configure automatic backups and trigger manual backups.

## System Settings

Most configuration is stored in the database. The admin panel provides a web UI to configure:

| Category | Settings |
|----------|----------|
| **General** | Application name, URL, announcement banner |
| **SMTP** | Mail server host, port, optional credentials, sender address. Empty username and password skip SMTP AUTH. |
| **OAuth** | GitHub and Google OAuth client credentials |
| **OIDC** | OpenID Connect provider for SSO |
| **WebAuthn** | Relying Party ID, display name, allowed origins |
| **Telegram** | Bot token for notifications |
| **Geocoding** | Provider selection and API key |
| **Storage** | Max upload size (managed here, not via environment variables) |
| **Backup** | Cron schedule, retention period |
| **Swagger** | Enable or disable API documentation UI |

::: tip
On first startup, these settings are seeded from environment variables if present. After that, changes are made exclusively through the admin panel.
:::

### Encryption at Rest

When `SETTINGS_ENC_KEY` is configured (see [Configuration, Encrypting Sensitive Settings](/guide/configuration#encrypting-sensitive-settings)), the following fields are AES-256-GCM encrypted in the database:

- `smtp.password`, `geocoding.api_key`, and any `secret.*` key in **system_settings**
- `client_secret` for every entry in **oauth_providers** (GitHub, Google, GitLab, Discord, OIDC)

The admin **GET /admin/settings** endpoint always redacts secret values to `***` regardless of whether encryption is enabled. Admin browsers and audit logs never see plaintext credentials. Submitting `***` on update keeps the existing value untouched, so the UI can round-trip non-secret edits without wiping credentials.

Existing plaintext rows are migrated transparently on the first boot after the key is set. The migration is idempotent and safe to re-run.

## Personalization {#personalization}

Account owners can customize many aspects of Bonds through the personalization settings at `/api/settings/personalize/:entity`.

Several personalization tables support reordering. You can move items, template pages, post template sections, group roles, or modules within template pages up or down using the UI buttons.

| Entity | What You Can Customize | Scope | Reorderable |
|--------|------------------------|-------|-------------|
| `genders` | Gender options | Account | Yes |
| `pronouns` | Pronoun options | Account | Yes |
| `address-types` | Address type labels | Account | Yes |
| `pet-categories` | Pet category types | Account | Yes |
| `contact-info-types` | Contact information types | Account | Yes |
| `relationship-types` | Relationship type definitions | Account | Yes (Sub-types) |
| `templates` | Contact page templates | Account | Yes (Pages) |
| `modules` | Module configuration | Account | Yes (Page modules) |
| `currencies` | Currency preferences | Account | No (Enabled switches) |
| `religions` | Religion options | Account | Yes |
| `call-reasons` | Call reason categories | Account | Yes (Sub-reasons) |
| `gift-occasions` | Gift occasion types | Account | Yes |
| `gift-states` | Gift tracking states | Account | Yes |
| `post-templates` | Journal post templates | Account | Yes (Sections) |
| `group-types` | Contact group types | Account | Yes (Roles) |
| `task-statuses` | Custom task statuses | Account | Yes |

Some built-in items, like email and phone contact information types, cannot be deleted.

## User Invitations

Invite others to your account via email:

1. Go to Settings, Invitations.
2. Enter the recipient email and choose a permission level.
3. An email is sent with an invitation link, valid for 7 days.
4. The recipient creates an account and is automatically linked to your account.

Permission levels: **Manager** (100), **Editor** (200), **Viewer** (300).

## Backup System

Bonds includes an automatic backup system:

- **Scheduled backups**: Configure a cron schedule in the admin panel (6-field format with seconds, e.g., `0 0 2 * * *` for 2 AM daily).
- **Manual backups**: Trigger a backup on demand from the admin panel.
- **Retention**: Old backups are automatically cleaned after a configurable number of days (default: 30).
- **Storage**: Backups are stored in the directory configured by `BACKUP_DIR` (default: `data/backups`).

## Cron Scheduler

Reminder delivery, CardDAV/CalDAV sync, and automatic backups all run through an internal cron scheduler:

- Single-process SQLite deployments work out of the box. Every job runs at most once per scheduled tick.
- **Multi-replica PostgreSQL deployments** (e.g. Kubernetes Deployments with `replicas: 2`, Docker Compose `deploy.replicas`, load-balanced pods) are also safe. Each job is gated by a `pg_try_advisory_xact_lock` plus an atomic conditional `UPDATE` on the `crons` table. Two replicas firing the same job at the same instant cannot both execute it.
- Crashed replicas cannot wedge a job. The advisory lock is released automatically when the holding transaction ends.

No configuration is required. The scheduler picks the correct strategy based on the active database driver.
