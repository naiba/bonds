# Bonds

[![Test](https://github.com/naiba/bonds/actions/workflows/test.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/test.yml)
[![Release](https://github.com/naiba/bonds/actions/workflows/release.yml/badge.svg)](https://github.com/naiba/bonds/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/naiba/bonds)](https://github.com/naiba/bonds/releases)

📖 [Documentation](https://naiba.github.io/bonds/) | [中文文档](README_zh.md) | 💬 [Discord](https://discord.gg/qp6TfrVekV)

<a href="https://www.producthunt.com/products/bonds?embed=true&amp;utm_source=badge-featured&amp;utm_medium=badge&amp;utm_campaign=badge-bonds" target="_blank" rel="noopener noreferrer"><img alt="Bonds - Remember everything about the people who matter. | Product Hunt" width="250" height="54" src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=1091729&amp;theme=light&amp;t=1772852214754"></a>

A modern, community-driven personal relationship manager, inspired by [Monica](https://github.com/monicahq/monica), rebuilt with **Go** and **React**.

## Why Bonds?

Monica is a beloved open-source personal CRM with 24k+ stars. But as a side project maintained by a tiny team ([their own words](https://github.com/monicahq/monica/issues/6626)), development has slowed, leaving 700+ open issues and limited bandwidth.

**Bonds** picks up where Monica left off:

- **Fast and lightweight**: Single binary, starts in milliseconds, minimal memory.
- **Easy to deploy**: One binary + SQLite. No PHP, no Composer, no Node runtime.
- **Modern UI**: React 19 + TypeScript, smooth SPA experience.
- **Well tested**: 1014 backend tests, 129 frontend tests, 180 E2E tests.
- **Community first**: Built for contributions and fast iteration.

> **Credits**: Bonds stands on the shoulders of [@djaiss](https://github.com/djaiss), [@asbiin](https://github.com/asbiin), and the entire Monica community. The original Monica remains available under AGPL-3.0 at [monicahq/monica](https://github.com/monicahq/monica).

## Features

- **Contacts**: Full lifecycle management with notes, tasks, reminders, gifts, money and item loans, activities, goals, pets, and more. Includes a needs-verification flag to keep your data fresh.
- **Vault Dashboard**: Responsive 3-column layout with a feed, activities, life metrics tracking (+1 counter), mood recording, upcoming reminders, and due tasks.
- **Vaults**: Multi-vault data isolation with role-based access (Manager, Editor, Viewer).
- **Reminders**: One-time and recurring (weekly, monthly, yearly), with email and Shoutrrr-compatible notifications.
- **Full-text Search**: Bleve-powered CJK-aware search across contacts and notes.
- **CardDAV / CalDAV**: Sync contacts and calendars with Apple, Thunderbird, and other DAV clients. Supports Personal Access Tokens.
  CardDAV REPORT request bodies are limited to 16 MiB (HTTP 413 above the limit); split larger multiget requests into batches. This does not limit the response size.
  Vault managers can configure CardDAV presence filters in **Vault Settings → General** (all/any of phone, email, names, nickname, company, job title or postal address). Empty filters preserve the default: all non-archived contacts. Excluded contacts remain in Bonds and relationships, disappear from devices on their next full sync, and return when filters are cleared. Stale writes to excluded contacts return 404; device creates/updates that do not match return 409 without saving. These settings apply to all vault members, not external DAV subscriptions, CalDAV or vCard exports.
  Company and job-title filters use the same employment projection as vCard exports: all normal Jobs entries, with legacy fields as a fallback for older contacts. ORG maps only its organization-name component to a vault-local company; departments are not separate companies, and escaped punctuation is preserved. Imports and subscription pulls share this mapping. Writes preserve other employment records; an explicit empty ORG with existing jobs returns a conflict instead of silently hiding or deleting them. Remove employment through Jobs in Bonds. Unchanged jobs are matched before edits or additions, independently of vCard property order. Removing TITLE clears a uniquely identified job title; ambiguous employment edits return a conflict. CardDAV writes require Editor or Manager access. Local Jobs changes take precedence over subscription updates under the existing local-wins policy. This includes company renaming and deletion, with subscription conflicts checked under the contact write lock. Legacy company relationships are retained as Jobs before a new ORG replaces the selected company. The legacy jobInformation endpoint treats an omitted/null company as a title-only edit of the selected job; use the Jobs deletion endpoints to remove employment. A single ORG/TITLE pair can be matched even if only one property is grouped; ambiguous unpaired titles are rejected.
- **DAV Sync Subscriptions**: Subscribe to and sync from external CardDAV address books directly into a vault.
- **Personal Access Tokens**: Generate secure API and sync tokens to access endpoints safely.
- **AI Agent Access (MCP)**: Built-in `/mcp` endpoint for MCP clients. Agents can discover capabilities, search vault data, fetch resources, and execute existing `/api` operations with the same auth and permissions. See [AI Agent Access](https://naiba.github.io/bonds/features/ai-agents).
- **CSV Import**: Import contacts from a CSV file with a user-defined column mapping (name, email, phone, birthday, address, tags, groups, notes).
- **Monica Import**: Migrate contacts directly from a Monica instance via API.
- **vCard Import/Export**: Bulk import `.vcf` files, export individual or all contacts.
  Jobs are exported as grouped `ORG`/`TITLE` pairs, including newly added employment, using the same projection as CardDAV filters. Existing employment selection and independent legacy relationships are retained. Local job/company changes push committed cards to active subscriptions independently of CardDAV presence filters. Repeated pulls of a rejected remote ETag remain skipped; a different remote version is evaluated normally.
- **File Upload**: Contact media with photos and videos, document attachments, and generated initials avatars. Storage size limits managed directly from the UI.
- **Two-Factor Auth (TOTP)**: TOTP-based 2FA with recovery codes.
- **WebAuthn / FIDO2**: Passkey login (hardware keys, biometrics).
- **OAuth Login**: GitHub and Google single sign-on.
- **User Invitations**: Invite others to your account via email with permission levels.
- **Audit Log**: Feed of all changes across contacts.
- **Geocoding**: Extensible provider configuration for Nominatim, LocationIQ, Geoapify, and public or self-hosted Photon.
- **Shoutrrr Notifications**: Reminder delivery via Telegram and other Shoutrrr-compatible channels.
- **i18n**: English and Chinese, frontend and backend.

## Quick Start

### Option 1: Docker (Recommended)

```bash
# Download docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# Generate and export a 256-bit JWT signing secret in this shell
export JWT_SECRET="$(openssl rand -hex 32)"

# Start the service
docker compose up -d
```

Open **http://localhost:8080** and create your account.

Generate the secret once, store it in a protected environment or secret store, and reuse the same value for every restart. Plan JWT secret rotation: it invalidates existing sessions and can require DAV subscription credentials to be entered again because their encryption derives from this secret.

### Option 2: Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/naiba/bonds/releases), then:

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
./bonds-server
```

The server starts at **http://localhost:8080** with an embedded frontend and SQLite database.

### Option 3: Build from Source

**Prerequisites**: Go 1.26+, [Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Install dependencies
make setup

# Build a single binary (frontend embedded)
make build-all

# Run it
export JWT_SECRET="$(openssl rand -hex 32)"
./server/bin/bonds-server
```

## Configuration

Bonds uses a **hybrid configuration** approach:

- **Environment variables**: For essential infrastructure settings (database, server, security).
- **Admin UI**: For all runtime settings (SMTP, OAuth, Telegram, WebAuthn, storage size limit, etc.).

On first startup, environment variables are seeded into the database. After that, manage settings from **Admin > System Settings** in the web UI.

```bash
cp server/.env.example server/.env
```

### Environment Variables (Required)

| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG` | `false` | Enable debug mode: Echo request logging, GORM SQL logging, Swagger UI (default on) |
| `JWT_SECRET` | — | **Required in production.** Generate with `openssl rand -hex 32`, then persist and reuse the 256-bit signing key across restarts. |
| `SETTINGS_ENC_KEY` | _(empty)_ | Optional. Enables AES-256-GCM encryption-at-rest for SMTP/OAuth/geocoding secrets. See [docs](https://naiba.github.io/bonds/guide/configuration#encrypting-sensitive-settings) |
| `SERVER_PORT` | `8080` | Port the server listens on |
| `SERVER_HOST` | `0.0.0.0` | Host address the server binds to |
| `DB_DSN` | `bonds.db` | Database connection string. SQLite: file path; PostgreSQL: `host=... port=5432 user=... password=... dbname=... sslmode=disable` |
| `DB_DRIVER` | `sqlite` | Database driver (`sqlite` or `postgres`) |
| `APP_ENV` | `development` | Set to `production` for production use |
| `STORAGE_UPLOAD_DIR` | `uploads` | File upload directory |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Full-text search index directory |
| `BACKUP_DIR` | `data/backups` | Directory to store backup files |

### Admin UI Settings

The following are managed from the **Admin > System Settings** page after login:

- **Application**: Name, URL, Announcement banner.
- **Authentication**: Password auth toggle, User registration toggle.
- **JWT**: Token expiry, Refresh window.
- **SMTP**: Host, Port, optional Username/Password, Sender email. Leave both Username and Password empty for unauthenticated relays.
- **OAuth / OIDC**: GitHub, Google, and OIDC/SSO credentials.
- **WebAuthn**: Relying Party ID, Display Name, Origins.
- **Telegram**: Bot token for notifications.
- **Geocoding**: Active provider, address privacy precision, per-provider credentials, and self-hosted Photon URL.
- **Storage**: Max upload size limit.
- **Backup**: Cron schedule, Retention days.
- **Swagger**: Enable or disable API documentation UI.

## Development

```bash
# Install dependencies
make setup

# Generate API client (required before first build)
make gen-api

# Start both frontend and backend in dev mode
make dev
```

This runs the Go backend on `:8080` and the Vite dev server on `:5173`. The frontend automatically proxies API requests to the backend.

### Code Generation Pipeline

The frontend TypeScript API client is **auto-generated** from the backend OpenAPI schema. The generated files are not committed to git. They are regenerated in CI and during development.

```
Go handlers (swag annotations)
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api (or bun run gen:api)
web/src/api/generated/   ← gitignored, regenerated on demand
    ↓
web/src/api/index.ts     ← entry point, imports generated modules
```

After changing any backend API (handlers, DTOs, routes), run:

```bash
make gen-api       # Regenerate swagger.json + TypeScript API client
```

### Useful Commands

```bash
make dev           # Start frontend + backend in dev mode
make build         # Build backend + frontend separately
make build-all     # Build single binary with embedded frontend
make test          # Run all tests (backend + frontend)
make test-e2e      # Run end-to-end tests (Playwright)
make lint          # Run linters (go vet + eslint)
make swagger       # Regenerate Swagger/OpenAPI docs only
make gen-api       # Regenerate Swagger docs + TypeScript API client
make clean         # Clean all build artifacts + generated files
make setup         # Install all dependencies
```

### API Documentation

Bonds provides auto-generated OpenAPI/Swagger documentation covering all API endpoints.

To access the Swagger UI, either enable debug mode or toggle it on in Admin > Settings > Swagger:
```bash
# Option 1: Debug mode (Swagger enabled by default)
DEBUG=true ./bonds-server
# Option 2: Enable via Admin UI without debug mode
# Go to Admin > Settings > Swagger > Enable
```

Then open http://localhost:8080/swagger/index.html

> Swagger UI defaults to the `DEBUG` flag, but can be independently toggled from the Admin Settings page.

## Relationship to Monica

Bonds is a ground-up rewrite inspired by [Monica](https://github.com/monicahq/monica) (AGPL-3.0). It re-implements Monica's data model and feature set using a completely different tech stack (Go + React instead of PHP/Laravel + Vue). It contains no code from the original project.

## License

[Business Source License 1.1](LICENSE) (BSL 1.1), Source Available license with the following terms:

- **Individuals**: Free for any non-commercial use.
- **Organizations**: Commercial use requires a paid license from the Licensor.
- **Prohibited**: Reselling, sublicensing, or offering as a managed/hosted service.
- **Change Date**: Each version automatically converts to [AGPL-3.0](LICENSE) on the fourth anniversary of that version's first publicly available distribution.

After its Change Date, that version becomes fully open source under AGPL-3.0.

By submitting code, documentation, translations, or any other contribution, you agree to the [contribution terms](CONTRIBUTING.md), including waiving all ownership and other rights or claims in that contribution to the fullest extent permitted by law.
