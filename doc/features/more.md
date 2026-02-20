# More Features

## Audit Log (Feed)

Bonds records a feed of all changes across contacts, providing a complete audit trail:

- Contact created, updated, deleted
- Notes added, edited, removed
- Reminders created, triggered
- Tasks, gifts, debts, activities, and other entity changes

The feed is accessible per vault at `GET /api/vaults/:vault_id/feed`, showing who made what change and when.

## Geocoding

Bonds can geocode addresses to obtain latitude/longitude coordinates. Two providers are supported:

| Provider | Cost | Configuration |
|----------|------|--------------|
| **Nominatim** | Free (OSM) | No API key needed |
| **LocationIQ** | Freemium | Requires API key |

Geocoding runs asynchronously when an address is created. If it fails, the address is still saved — coordinates are simply left empty.

Configure the provider and API key in the admin panel.

## Telegram Notifications {#telegram-notifications}

Receive reminder notifications via Telegram:

### Setup

1. Create a Telegram bot via [@BotFather](https://t.me/BotFather)
2. Copy the bot token
3. Enter it in the admin panel under Telegram settings
4. In your user settings, add a Telegram notification channel with your chat ID

### Getting Your Chat ID

Send a message to your bot, then visit:
```
https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
```
Look for the `chat.id` field in the response.

## Internationalization (i18n)

Bonds supports English and Chinese in both frontend and backend:

- **Frontend**: React i18next with `en.json` and `zh.json` locale files
- **Backend**: Embedded JSON locale files, parsed from the `Accept-Language` header

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

Bonds includes a comprehensive currency table (160+ currencies) for tracking financial data like debts and gifts. Currencies are linked to accounts and can be managed through personalization settings.

## Calendar Support

Bonds supports multiple calendar systems:

- **Gregorian** — Standard calendar (default)
- **Lunar (Chinese)** — Traditional Chinese calendar powered by `6tail/lunar-go`

The calendar system uses a converter interface, making it extensible to additional calendar types.
