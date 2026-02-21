# Getting Started

## Option 1: Docker (Recommended)

The easiest way to run Bonds is with Docker:

```bash
# Download docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# Start the service
docker compose up -d
```

Open **http://localhost:8080** and create your account.

### Customize Settings

Edit `docker-compose.yml` to set your JWT secret and other options:

```yaml
environment:
  - JWT_SECRET=your-secret-key-here   # ⚠️ Change this in production!
  - APP_URL=https://bonds.example.com
  - APP_ENV=production
```

### Persistent Storage

The default `docker-compose.yml` mounts a volume for the SQLite database and uploaded files. Your data persists across container restarts.

## Option 2: Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/naiba/bonds/releases):

```bash
# Set required environment variables
export JWT_SECRET=your-secret-key-here
export APP_ENV=production

# Run the server
./bonds-server
```

The server starts at **http://localhost:8080** with an embedded frontend and SQLite database.

### Data Directories

By default, Bonds stores data in the working directory:

| Path | Purpose |
|------|---------|
| `bonds.db` | SQLite database |
| `uploads/` | Uploaded files (photos, documents) |
| `data/bonds.bleve/` | Full-text search index |
| `data/backups/` | Automatic backups |

## Option 3: Build from Source

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

## First Steps After Login

::: tip First User = Instance Admin
The **first user** to register on a new Bonds instance is automatically granted **instance admin** privileges. This user can access the admin panel to manage system settings, other users, and backups. Additional admins can be promoted from the admin panel.
:::

1. **Create a Vault** — Vaults are isolated containers for your contacts. You might create one for "Family", another for "Work".
2. **Add Contacts** — Create contacts inside a vault. Add their details, photos, notes.
3. **Set Up Reminders** — Never forget a birthday or important date. Configure email or Telegram notifications.
4. **Invite Others** — Share a vault with family members by sending email invitations with appropriate permission levels.

## System Requirements

- **CPU**: Any modern 64-bit processor
- **RAM**: ~50 MB at idle, scales with usage
- **Disk**: Minimal; depends on uploaded files
- **OS**: Linux (amd64, arm64), macOS, Windows
- **Database**: SQLite (bundled) or PostgreSQL 14+
