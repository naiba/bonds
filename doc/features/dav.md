# CardDAV / CalDAV

Bonds includes a built-in CardDAV and CalDAV server. This allows you to sync contacts and calendars with external applications like Apple Contacts, Thunderbird, GNOME Contacts, and other DAV-compatible clients.

## Endpoints

| Protocol | URL | Description |
|----------|-----|-------------|
| CardDAV | `/dav/` | Contact synchronization |
| CalDAV | `/dav/` | Calendar synchronization (important dates, tasks) |
| Discovery | `/.well-known/carddav` | Auto-discovery redirect |
| Discovery | `/.well-known/caldav` | Auto-discovery redirect |

## Authentication

DAV clients use **HTTP Basic Auth** (not JWT), because most DAV clients do not support token-based authentication.

| Scenario | Credentials |
|----------|-------------|
| 2FA **disabled** | Email + password, or email + Personal Access Token |
| 2FA **enabled** | Email + Personal Access Token **only** (password is blocked) |

### Personal Access Tokens for DAV Sync

Personal Access Tokens are highly recommended for DAV clients even when Two-Factor Authentication is disabled.

- You can generate tokens under **Settings > API Tokens**.
- Provide a clear description and an optional expiration period.
- Copy the token upon generation, as it is only displayed once.
- In your DAV client, enter your **email address** as the username and the generated token (prefixed with `bonds_`) as the password.

::: warning
When you enable 2FA, any DAV clients using your password will stop syncing. Update them to use a Personal Access Token instead.
:::

## What Gets Synced

### CardDAV (Contacts to vCard 4.0)

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

## DAV Sync Subscriptions

In addition to exposing Bonds as a DAV server, each vault can subscribe to external CardDAV address books from the vault's **DAV Sync** page.

- **Create a subscription** with the remote server URI, username, password, optional address book path, sync direction, and frequency.
- **Test Connection** checks the remote server and discovers available address books before saving. If one address book is found, Bonds selects it automatically.
- **Sync directions**: Pull Only imports remote contacts into the vault, Push Only sends local contact changes to the remote address book, and Bidirectional does both.
- **Schedule and manual runs**: the default frequency is 180 minutes, with options from 30 minutes to 24 hours. Use **Sync Now** to trigger an immediate run.
- **Sync logs** record created, updated, deleted, pushed, skipped, and error events for each subscription.
- Remote passwords are encrypted at rest using a key derived from `JWT_SECRET`.

## Client Setup

### Apple Contacts / Calendar (macOS / iOS)

1. Go to **Settings > Accounts > Add Account > Other**.
2. Choose **Add CardDAV Account** or **Add CalDAV Account**.
3. Enter:
   - Server: `https://your-bonds-domain.com`
   - Username: your email
   - Password: your password (if 2FA is enabled, use a Personal Access Token)

The well-known URLs (`/.well-known/carddav`, `/.well-known/caldav`) enable automatic discovery.

### Thunderbird

1. Open **Address Book > New > CardDAV Address Book**.
2. Enter the URL: `https://your-bonds-domain.com/dav/`
3. Authenticate with your Bonds credentials.

## ETags

Bonds uses the `UpdatedAt` timestamp (Unix epoch) as the ETag for all DAV resources. Clients use ETags to detect changes and sync efficiently.
