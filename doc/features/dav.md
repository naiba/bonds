# CardDAV / CalDAV

Bonds includes a built-in CardDAV and CalDAV server, allowing you to sync contacts and calendars with external applications like Apple Contacts, Thunderbird, GNOME Contacts, and other DAV-compatible clients.

## Endpoints

| Protocol | URL | Description |
|----------|-----|-------------|
| CardDAV | `/dav/` | Contact synchronization |
| CalDAV | `/dav/` | Calendar synchronization (important dates, tasks) |
| Discovery | `/.well-known/carddav` | Auto-discovery redirect |
| Discovery | `/.well-known/caldav` | Auto-discovery redirect |

## Authentication

DAV clients use **HTTP Basic Auth** (not JWT), since most DAV clients don't support token-based authentication.

| Scenario | Credentials |
|----------|-------------|
| 2FA **disabled** | Email + password, or email + [Personal Access Token](/features/more#personal-access-tokens) |
| 2FA **enabled** | Email + Personal Access Token **only** (password is blocked) |

::: warning
When you enable 2FA, any DAV clients using your password will stop syncing. Update them to use a Personal Access Token from **Settings → API Tokens**.
:::

## What Gets Synced

### CardDAV (Contacts → vCard 4.0)

| Bonds Field | vCard Property |
|-------------|---------------|
| First + Last name | `FN`, `N` |
| Phone numbers | `TEL` |
| Email addresses | `EMAIL` |
| Addresses | `ADR` |

### CalDAV

| Bonds Entity | iCal Type | Notes |
|-------------|-----------|-------|
| Important dates | `VEVENT` | With `RRULE=YEARLY` for recurring dates |
| Tasks | `VTODO` | Task due dates and status |

## Client Setup

### Apple Contacts / Calendar (macOS / iOS)

1. Go to **Settings → Accounts → Add Account → Other**
2. Choose **Add CardDAV Account** or **Add CalDAV Account**
3. Enter:
   - Server: `https://your-bonds-domain.com`
   - Username: your email
   - Password: your password (if 2FA is enabled, use a Personal Access Token)

The well-known URLs (`/.well-known/carddav`, `/.well-known/caldav`) enable automatic discovery.

### Thunderbird

1. Open **Address Book → New → CardDAV Address Book**
2. Enter the URL: `https://your-bonds-domain.com/dav/`
3. Authenticate with your Bonds credentials

## ETags

Bonds uses the `UpdatedAt` timestamp (Unix epoch) as the ETag for all DAV resources. Clients use ETags to detect changes and sync efficiently.
