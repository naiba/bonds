# Development

## Prerequisites

- **Go** 1.25+
- **[Bun](https://bun.sh)** 1.x (used instead of npm/yarn)
- **Make** (GNU Make)

## Setup

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Install all dependencies (Go modules + Bun packages)
make setup

# Generate API client (required before first build)
make gen-api

# Start both frontend and backend in dev mode
make dev
```

This runs the Go backend on `:8080` and the Vite dev server on `:5173`. The frontend proxies API requests to the backend automatically.

## Code Generation Pipeline

The frontend TypeScript API client is **auto-generated** from the backend's OpenAPI/Swagger spec:

```
Go handlers (swag annotations)
    ↓  make swagger
server/docs/swagger.json
    ↓  make gen-api
web/src/api/generated/   ← gitignored
    ↓
web/src/api/index.ts     ← entry point
```

After changing any backend API (handlers, DTOs, routes):

```bash
make gen-api   # Regenerate swagger.json + TypeScript client
```

::: warning
Never manually edit files in `web/src/api/generated/`. They are overwritten on every generation.
:::

## Useful Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start frontend + backend in dev mode |
| `make build` | Build backend + frontend separately |
| `make build-all` | Build single binary with embedded frontend |
| `make test` | Run all tests (backend + frontend) |
| `make test-server` | Run backend tests only |
| `make test-web` | Run frontend tests only |
| `make test-e2e` | Run Playwright E2E tests |
| `make lint` | Run linters (`go vet` + ESLint) |
| `make swagger` | Regenerate Swagger/OpenAPI docs |
| `make gen-api` | Regenerate Swagger + TypeScript API client |
| `make clean` | Clean all build artifacts |
| `make setup` | Install all dependencies |

## Project Structure

```
server/                    # Go backend
  cmd/server/main.go       # Entry point
  internal/
    handlers/               # HTTP handlers (Echo)
    services/               # Business logic
    models/                 # GORM models
    dto/                    # Request/response structs
    middleware/              # JWT auth, CORS, etc.
    search/                 # Bleve full-text search
    dav/                    # CardDAV/CalDAV server
    cron/                   # Cron scheduler
    i18n/                   # Backend i18n
  pkg/
    avatar/                 # Initials avatar generation
    response/               # API response helpers

web/                       # React frontend
  src/
    api/                    # Auto-generated API client
    components/             # Shared components
    pages/                  # Route pages
    stores/                 # Auth + theme context
    locales/                # i18n (en.json, zh.json)
    utils/                  # Utility functions
  e2e/                      # Playwright tests
```

## Backend Architecture

Every feature follows: **Handler** (HTTP layer) → **Service** (business logic) → **DTO** (request/response) → **Model** (GORM).

- Handlers bind requests, validate, delegate to services, and return via `response.*` helpers
- Services receive DTOs, return DTOs, and hold `*gorm.DB` for queries
- Models are pure GORM structs with no business logic

## Testing

```bash
# Backend tests (in-memory SQLite)
cd server && go test ./... -v -count=1

# Frontend unit tests (Vitest)
cd web && bun run test

# E2E tests (Playwright — auto-starts servers)
cd web && bunx playwright test
```

## API Documentation (Swagger)
Bonds auto-generates OpenAPI docs covering all 286 API endpoints.

To access the Swagger UI, either enable debug mode or toggle it on in Admin > Settings > Swagger:
```bash
# Option 1: Debug mode (Swagger enabled by default)
DEBUG=true ./bonds-server
# Option 2: Enable via Admin UI without debug mode
# Go to Admin > Settings > Swagger > Enable
```

Then open http://localhost:8080/swagger/index.html

> Swagger UI defaults to the `DEBUG` flag, but can be independently toggled from the Admin Settings page.
