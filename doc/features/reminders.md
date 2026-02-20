# Reminders

Bonds has a built-in reminder system that notifies you about important dates and events via email or Telegram.

## Types

| Type | Behavior |
|------|----------|
| **One-time** | Triggers once at the scheduled date/time, then done |
| **Weekly** | Repeats every week |
| **Monthly** | Repeats every month |
| **Yearly** | Repeats every year (ideal for birthdays) |

## How It Works

1. **Create a reminder** on a contact — choose the date, time, type, and a label
2. **The cron scheduler** runs every minute, scanning for due reminders
3. **Notifications are sent** through your configured notification channels
4. **For recurring reminders**, the next occurrence is automatically scheduled based on the previous scheduled time (not current time, to prevent drift)

## Notification Channels

Bonds supports two notification channels:

### Email

Configure SMTP settings in the admin panel. Email notifications are enabled by default when a user registers — a notification channel is automatically created with the user's email address.

### Telegram

Set up a Telegram bot and configure the bot token in the admin panel. Users then link their Telegram chat ID to receive reminder notifications via the bot.

See [Telegram Notifications](/features/more#telegram-notifications) for setup details.

## Channel Reliability

Each notification channel tracks a failure counter. If a channel fails **10 consecutive times**, it is automatically disabled to prevent spam. You can re-enable it manually from user settings after fixing the underlying issue.

## Notification History

Every notification attempt is recorded in `UserNotificationSent`, including:
- Delivery status (success/failure)
- Error message (if failed)
- Timestamp
