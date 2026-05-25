---
name: testing-bonds-runtime
description: Test the Bonds app end-to-end locally through the browser. Use when verifying dependency changes or full-stack runtime behavior across registration, vaults, and contacts.
---

# Bonds Runtime Testing

## Devin Secrets Needed

None for the local self-registration smoke flow. The default local seed enables registration and disables required email verification when SMTP is not configured.

## Verified local setup

From the repo root, run the backend and frontend in separate shells:

```bash
cd server
GOPROXY=https://goproxy.cn,direct \
SERVER_PORT=8080 \
SERVER_HOST=0.0.0.0 \
DB_DRIVER=sqlite \
DB_DSN=/absolute/path/to/repo/.devin-test/bonds-e2e.db \
STORAGE_UPLOAD_DIR=/absolute/path/to/repo/.devin-test/uploads \
BLEVE_INDEX_PATH=/absolute/path/to/repo/.devin-test/bleve \
BACKUP_DIR=/absolute/path/to/repo/.devin-test/backups \
JWT_SECRET=devin-local-e2e-secret \
APP_URL=http://localhost:5173 \
go run cmd/server/main.go
```

```bash
cd web
PLAYWRIGHT_SERVER_PORT=8080 bun dev --host 0.0.0.0
```

Vite serves the app at `http://localhost:5173` and proxies `/api` to the Go backend on port 8080.

## Browser smoke flow

1. Open `http://localhost:5173/register`.
2. Assert the page renders `Create an account` and the `Create account` button.
3. Register a unique local user using a unique `@example.com` email and a password such as `Password123!`.
4. Assert the app redirects to `/vaults` and shows the authenticated `Vaults` page.
5. Create a vault with a recognizable name.
6. Assert the vault dashboard loads and shows `Add contact`.
7. Create a contact with first name, last name, and nickname.
8. Assert the contact detail page shows the expected name, nickname, first/last fields, and `Active` status.

## Notes and caveats

- If backend startup logs that Bleve search could not initialize, search may be disabled; this does not block the registration/vault/contact smoke flow, but do not claim search was tested.
- Ant Design deprecation warnings might appear in the browser console; record them as DX warnings unless they block runtime behavior.
- Use screen recording with annotations for UI testing evidence.
