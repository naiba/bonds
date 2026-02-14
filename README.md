# Bonds

> [中文文档](README_zh.md)

A modern, community-driven personal relationship manager — inspired by [Monica](https://github.com/monicahq/monica), rebuilt with **Go** and **React**.

## Why Bonds?

Monica is a beloved open-source personal CRM with 24k+ stars. But as a side project maintained by a tiny team ([their own words](https://github.com/monicahq/monica/issues/6626)), development has slowed — 700+ open issues and limited bandwidth.

**Bonds** picks up where Monica left off:

- **Fast & lightweight** — Single binary, starts in milliseconds, minimal memory
- **Easy to deploy** — One binary + SQLite. No PHP, no Composer, no Node runtime
- **Modern UI** — React 19 + TypeScript, smooth SPA experience
- **Well tested** — 347 backend tests, 54 frontend tests, 6 E2E spec files
- **Community first** — Built for contributions and fast iteration

> **Credits**: Bonds stands on the shoulders of [@djaiss](https://github.com/djaiss), [@asbiin](https://github.com/asbiin), and the entire Monica community. The original Monica remains available under AGPL-3.0 at [monicahq/monica](https://github.com/monicahq/monica).

## Features

- **Contacts** — Full lifecycle management with notes, tasks, reminders, gifts, debts, activities, life events, pets, and more
- **Vaults** — Multi-vault data isolation with role-based access (Manager / Editor / Viewer)
- **Reminders** — One-time and recurring (weekly/monthly/yearly), with email and Telegram notifications
- **Full-text Search** — Bleve-powered CJK-aware search across contacts and notes
- **CardDAV / CalDAV** — Sync contacts and calendars with Apple, Thunderbird, and other DAV clients
- **vCard Import/Export** — Bulk import `.vcf` files, export individual or all contacts
- **File Upload** — Photos and documents attached to contacts, with generated initials avatars
- **Two-Factor Auth (TOTP)** — TOTP-based 2FA with recovery codes
- **WebAuthn / FIDO2** — Passkey login (hardware keys, biometrics)
- **OAuth Login** — GitHub and Google single sign-on
- **User Invitations** — Invite others to your account via email with permission levels
- **Audit Log** — Feed of all changes across contacts
- **Geocoding** — Address coordinates via Nominatim (free) or LocationIQ
- **Telegram Notifications** — Reminder delivery via Telegram bot
- **i18n** — English and Chinese, frontend and backend

## Quick Start

### Option 1: Docker (Recommended)

```bash
git clone https://github.com/naiba/bonds.git
cd bonds
docker compose up -d
```

Open **http://localhost:8080** and create your account.

To persist data and customize settings:

```yaml
# docker-compose.yml — environment section
environment:
  - JWT_SECRET=your-secret-key-here   # ⚠️ Change this!
  - SERVER_PORT=8080
  - DB_DSN=/app/data/bonds.db
```

### Option 2: Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/naiba/bonds/releases), then:

```bash
export JWT_SECRET=your-secret-key-here
./bonds-server
```

The server starts at **http://localhost:8080** with an embedded frontend and SQLite database.

### Option 3: Build from Source

**Prerequisites**: Go 1.25+, [Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Install dependencies
make setup

# Build a single binary (frontend embedded)
make build-all

# Run it
export JWT_SECRET=your-secret-key-here
./server/bin/bonds-server
```

## Configuration

Bonds is configured via environment variables. Copy the example file to get started:

```bash
cp server/.env.example server/.env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | — | **Required in production.** Signing key for auth tokens |
| `SERVER_PORT` | `8080` | Port the server listens on |
| `DB_DSN` | `bonds.db` | SQLite database file path |
| `APP_ENV` | `development` | Set to `production` for production use |
| `APP_URL` | `http://localhost:8080` | Public URL (used in emails and OAuth callbacks) |
| `SMTP_HOST` | — | SMTP server for sending emails |
| `SMTP_PORT` | — | SMTP port |
| `SMTP_USERNAME` | — | SMTP username |
| `SMTP_PASSWORD` | — | SMTP password |
| `SMTP_FROM` | — | Sender email address |
| `STORAGE_UPLOAD_DIR` | `uploads` | File upload directory |
| `STORAGE_MAX_SIZE` | `10485760` | Max upload size in bytes (10 MB) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram bot token for notifications |
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Full-text search index directory |
| `OAUTH_GITHUB_KEY` | — | GitHub OAuth App client ID |
| `OAUTH_GITHUB_SECRET` | — | GitHub OAuth App client secret |
| `OAUTH_GOOGLE_KEY` | — | Google OAuth client ID |
| `OAUTH_GOOGLE_SECRET` | — | Google OAuth client secret |
| `GEOCODING_PROVIDER` | `nominatim` | Geocoding provider (`nominatim` or `locationiq`) |
| `GEOCODING_API_KEY` | — | API key for LocationIQ |
| `WEBAUTHN_RP_ID` | — | WebAuthn Relying Party ID (e.g. `bonds.example.com`) |
| `WEBAUTHN_RP_DISPLAY_NAME` | — | WebAuthn display name |
| `WEBAUTHN_RP_ORIGINS` | — | Allowed WebAuthn origins (comma-separated) |

## Development

```bash
# Start both frontend and backend in dev mode
make dev
```

This runs the Go backend on `:8080` and the Vite dev server on `:5173`. The frontend automatically proxies API requests to the backend.

Other useful commands:

```bash
make test          # Run all tests (backend + frontend)
make test-e2e      # Run end-to-end tests (Playwright)
make lint          # Run linters
make clean         # Clean build artifacts
```

## Relationship to Monica

Bonds is a ground-up rewrite inspired by [Monica](https://github.com/monicahq/monica) (AGPL-3.0). It re-implements Monica's data model and feature set using a completely different tech stack (Go + React instead of PHP/Laravel + Vue). It contains no code from the original project.

## License

[AGPL-3.0](LICENSE) — Same license as the original Monica project.
