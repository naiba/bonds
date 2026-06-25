# More Features

## Audit Log (Feed)

Bonds records a feed of all changes across contacts, providing a complete audit trail:

- Contact created, updated, deleted.
- Notes added, edited, removed.
- Reminders created, triggered.
- Tasks, gifts, loans, activities, and other entity changes.

The feed is accessible per vault at `GET /api/vaults/:vault_id/feed`, showing who made what change and when.

## Personal Access Tokens {#personal-access-tokens}

Bonds allows you to create Personal Access Tokens, also shown in the UI as API Tokens, for secure API access and DAV synchronization:

- **Location**: Manage your tokens under **Settings > API Tokens**.
- **Creation**: Specify a custom description and an optional expiration period.
- **Security**: The token is only shown once upon creation. Ensure you copy it immediately.
- **Usage**: Use the token as a password in external integrations and DAV clients. When Two-Factor Authentication is active, standard password logins are disabled for CardDAV and CalDAV sync. You must use a Personal Access Token instead.
- **AI agents**: Use a Personal Access Token as the Bearer token for the built-in [`/mcp` endpoint](/features/ai-agents).
- **Format**: All Personal Access Tokens are prefixed with `bonds_` for easy identification.

## Geocoding

Bonds can geocode addresses to obtain latitude/longitude coordinates. Two providers are supported:

| Provider | Cost | Configuration |
|----------|------|--------------|
| **Nominatim** | Free (OSM) | No API key needed |
| **LocationIQ** | Freemium | Requires API key |

Geocoding runs asynchronously when an address is created. If it fails, the address is still saved. Coordinates are simply left empty.

Configure the provider and API key in the admin panel.

## Shoutrrr / Telegram Notifications {#telegram-notifications}

Receive reminder notifications through Shoutrrr-compatible URLs, including Telegram:

### Setup

1. Create a Telegram bot via [@BotFather](https://t.me/BotFather).
2. Get the destination chat ID.
3. In **Settings > Notifications**, add a Shoutrrr notification channel.
4. Use a Telegram Shoutrrr URL such as `telegram://token@telegram?channels=123456`.
5. Choose the preferred send time for this channel.

### Getting Your Chat ID

Send a message to your bot, then visit:
```
https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
```
Look for the `chat.id` field in the response.

The preferred send time is used for new reminders, backfilled existing reminders, and future recurring reminder schedules. If it is left blank or invalid, Bonds falls back to `09:00`.

## Internationalization (i18n)

Bonds supports English and Chinese in both frontend and backend:

- **Frontend**: React i18next with `en.json` and `zh.json` locale files.
- **Backend**: Embedded JSON locale files, parsed from the `Accept-Language` header.

The language is detected automatically from the browser's language setting. Users can also switch languages manually.

## User Invitations

Invite others to join your account with controlled permissions:

| Permission Level | Value | Access |
|-----------------|-------|--------|
| Manager | 100 | Full access to vault management |
| Editor | 200 | Can create and edit contacts |
| Viewer | 300 | Read-only access |

Invitations are sent via email with a unique token link, valid for 7 days.

## Currencies

Bonds includes a comprehensive currency table (160+ currencies) for tracking financial data like money loans and gifts. Currencies are linked to accounts and can be managed through personalization settings.

## Calendar Support

Bonds supports multiple calendar systems:

- **Gregorian**: Standard calendar (default).
- **Lunar (Chinese)**: Traditional Chinese calendar powered by `6tail/lunar-go`.

The calendar system uses a converter interface, making it extensible to additional calendar types.
