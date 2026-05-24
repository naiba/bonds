# i18n & Multi-Calendar Fixes — Design Spec

- **Date**: 2026-05-22
- **Author**: naiba (with Claude assistance)
- **Status**: Approved for implementation
- **Delivery**: 5 sequential commits directly to `main` (no PR), each commit keeps CI green.

## 1. Background

A two-pass audit of `/root/bonds` (Go + React personal CRM) surfaced ~17 multi-language and multi-calendar defects, plus 2 reminder-specific calendar bugs. Symptoms users reported earlier:

1. Mood-tracking labels stay English even after switching account to Chinese.
2. Personalize "Sync translations" leaves labels in the original language.
3. Top-bar language toggle only flips between zh/en even though Preferences exposes more.

Bugs 1–3 were fixed in a previous session (now Commit 1 of this plan). This spec collects every other finding and turns them into a single coherent delivery.

## 2. Goals

- All user-visible strings honor `user.Preferences.Locale` everywhere — including async/cron paths (email, push, reminder scheduler) that bypass the HTTP `Accept-Language` middleware.
- All recurring date logic for lunar (or other non-Gregorian) entries computes the correct annual occurrence both in-app (reminder scheduler, notifications activation) and on export (CalDAV).
- The frontend keeps i18n, AntD, and dayjs locale state in sync with the user's saved preference at boot and after save.
- Locale and timezone validation is robust; invalid stored values cannot break the app.
- Test coverage grows alongside fixes: ~30 new unit tests + 3 new E2E scenarios.

## 3. Non-Goals

- Adding new languages beyond the existing en / zh / es.
- Adding new calendar systems beyond gregorian + lunar.
- UI redesign of any settings page.
- Replacing every raw `<DatePicker />` in the codebase — only forms whose semantics are "personal important date" (Life Events, Journal, Task, Vault personal-date fields) get `CalendarDatePicker`. Administrative/technical dates (address effective_from, call timestamp, API token expiration) stay Gregorian-only by design.

## 4. Architecture Touchpoints

| Area | Files |
|---|---|
| Backend i18n | `server/internal/i18n/`, locale middleware, `pkg/response/response.go` |
| Locale-aware services | `services/auth.go`, `services/invitation.go`, `services/notifications.go`, `services/reminder_scheduler.go`, `services/personalize.go` |
| Models / seed | `models/contact_important_date.go`, `models/seed_vault.go`, `database/backfill_*.go` |
| Calendar | `internal/calendar/`, `internal/dav/caldav_backend.go`, helpers shared with reminders |
| Frontend i18n / AntD / dayjs | `web/src/i18n.ts`, `web/src/ThemedApp.tsx`, `web/src/utils/dayjsLocale.ts` (new), `web/src/pages/settings/Preferences.tsx` |
| Frontend calendar / timezone | `web/src/components/CalendarDatePicker.tsx`, `web/src/utils/dateFormat.ts`, `web/src/pages/contacts/modules/LifeEventsModule.tsx`, `web/src/pages/journal/JournalDetail.tsx`, `web/src/pages/tasks/TaskEditModal.tsx`, `web/src/pages/vaults/VaultDetail.tsx` |

## 5. Commit Plan

Each commit must leave `bun run lint`, `bun run test`, `bun run build`, and `go test ./...` passing.

### Commit 1 — Land existing i18n fixes

Commit currently uncommitted work from the previous session:

- 8 modified files: `web/src/api/index.ts`, `web/src/components/Layout.tsx`, `web/src/i18n.ts`, `web/src/pages/auth/{AcceptInvite,Login,OAuthLink,Register}.tsx`, `web/src/pages/settings/Preferences.tsx`.
- 3 new files: `web/src/components/LanguageSwitcher.tsx`, `web/src/test/AcceptLanguageInterceptor.test.ts`, `web/src/test/LanguageSwitcher.test.tsx`.

Provides `SUPPORTED_LANGUAGES`, `normalizeLanguageCode`, Accept-Language interceptor — all later commits depend on these.

Commit message: `fix(i18n): wire Accept-Language header and add language switcher`

### Commit 2 — Backend i18n + locale robustness

**A2 — Locale validation**
- `server/internal/dto/settings.go:30`: add `validate:"required,oneof=en zh es"` to `UpdateLocaleRequest.Locale`.
- `server/internal/services/preferences_test.go:144-145`: change the existing "de" assertion to expect a 422.
- New tests: `TestUpdateLocaleRejectsUnsupportedLocale`, `TestUpdateLocaleAcceptsZhEsEn`.

**A4 / E18 / E19 — Localize async/cron paths**

New helper `services/locale_resolver.go`:

```go
func ResolveUserLocale(user *models.User) string {
    if user == nil || user.Preferences == nil {
        return "en"
    }
    switch user.Preferences.Locale {
    case "en", "zh", "es":
        return user.Preferences.Locale
    default:
        return "en"
    }
}
```

New i18n keys (en/zh/es, all three together):

```
email.verify.subject
email.verify.body
email.invite.subject
email.invite.body
email.notification_test.subject
email.notification_test.body
reminder.subject
reminder.body_with_contact
reminder.body_no_contact
reminder.unknown_contact
```

Refactor callers to:

```go
locale := ResolveUserLocale(user)
subject := i18n.T(locale, "email.verify.subject")
body := i18n.Tf(locale, "email.verify.body", map[string]any{"Link": link})
```

For substitutions use existing `fmt.Sprintf("%s", …)` style (matches current codebase). Translation strings keep `%s` placeholders in the same order across en/zh/es.

Touched files:
- `services/auth.go:71-77`
- `services/invitation.go:72-79`
- `services/notifications.go:78-79,127-128,195-196`
- `services/reminder_scheduler.go:82-87` (body wiring; date formatter comes in Commit 3)

New tests:
- `auth_test.go::TestSendVerifyEmailHonorsUserLocale`
- `invitation_test.go::TestInvitationEmailHonorsInviterLocale`
- `notifications_test.go::TestVerifyChannelHonorsUserLocale`, `TestPushBodyHonorsUserLocale`
- `reminder_scheduler_test.go::TestReminderEmailSubjectHonorsLocale`, `TestReminderBodyHonorsLocale`

**B6 — ContactImportantDateType translation key**

- `models/contact_important_date.go`: add `LabelTranslationKey string \`gorm:"size:128"\`` to `ContactImportantDateType`.
- GORM `AutoMigrate(AllModels()...)` already runs in `cmd/server/main.go`; the new column is created automatically — no manual migration file needed.
- `models/seed_vault.go:23-58` `seedContactImportantDateTypes`: write `LabelTranslationKey` alongside `Label`.
- `services/personalize.go`: add the table to `vaultSyncEntities`.
- New `database/backfill_contact_important_date_types.go` (mirrors existing `backfill_task_statuses.go` pattern): on startup, find rows where `LabelTranslationKey = ''` and `Label` matches the seeded English defaults (`Birthday`, `Anniversary`, etc.), set the matching key.
- Wire backfill into `cmd/server/main.go` alongside the existing `models.BackfillTaskStatuses(db)` call at line 73.

New tests:
- `seed_vault_test.go::TestSeedContactImportantDateTypesPersistsTranslationKey`
- `personalize_test.go::TestSyncAllTranslationsCoversImportantDateTypes`
- `backfill_contact_important_date_types_test.go::TestBackfillMapsSeededTypesByName`, `TestBackfillSkipsCustomTypes`

Commit message: `fix(i18n): localize email/notification/cron paths and validate locale`

### Commit 3 — Backend calendar + timezone

Implementation order within this commit:
1. Add `UpcomingOccurrences` to `Converter` (gregorian + lunar implementations).
2. Extract `NextFireAt(reminder, user, now) time.Time` into a shared helper, then refactor `calcInitialSchedule` to use it (D14 timezone parameter included).
3. Switch `notifications.go:299-339` over to the shared helper (Reminder Bug 1).
4. CalDAV RDATE emission (D13).
5. Notification body date formatter (Reminder Bug 2) — depends on locale plumbing from Commit 2.

**D13 — CalDAV emits RDATE for non-Gregorian**

`server/internal/dav/caldav_backend.go:511-523`:

```go
if d.CalendarType == "" || d.CalendarType == "gregorian" {
    // existing RRULE=YEARLY path
} else {
    converter := calendar.Get(d.CalendarType)
    if converter != nil {
        // compute next 10 occurrences and emit RDATE lines
        for _, occ := range converter.UpcomingOccurrences(originalDate, 10) {
            fmt.Fprintf(buf, "RDATE;VALUE=DATE:%s\r\n", occ.Format("20060102"))
        }
    }
}
```

Add `UpcomingOccurrences(seed time.Time, n int) []time.Time` to the `Converter` interface. Gregorian returns a slice of `n` annual occurrences computed by adding years. Lunar implements via the 6tail lunar-go conversion for each successive lunar year.

New tests:
- `caldav_backend_test.go::TestExportImportantDateLunarUsesRDATE`
- `caldav_backend_test.go::TestExportImportantDateGregorianUsesRRULE`

**D14 — Reminder uses user timezone**

- `services/reminder_scheduler.go:201-215, 224-246`: extend `calcInitialSchedule` and `calcNextYearlySchedule` to accept `tz *time.Location`; build the 09:00 fire time with that location.
- All call sites: load `time.LoadLocation(user.Preferences.Timezone)`, fallback `time.UTC` on error.
- New tests: `reminder_scheduler_test.go::TestCalcInitialScheduleUsesUserTimezone` (Asia/Tokyo user → fire UTC time is the day before at 00:00).

**Reminder Bug 1 — `ScheduleAllContactReminders` honors calendar**

`services/notifications.go:299-339`: replace the hand-rolled `time.Date(now.Year(), r.Month, r.Day, …)` block with the same lunar-aware logic in `reminder.calcInitialSchedule`. Extract a shared helper `services/reminder_next_fire.go::NextFireAt(reminder, user, now) time.Time` and call it from both places.

New test: `notifications_test.go::TestScheduleAllContactRemindersHandlesLunarReminder`.

**Reminder Bug 2 — Notification body includes formatted date**

New helper `services/reminder_date_format.go`:

```go
func FormatReminderDate(reminder *models.ContactReminder, user *models.User, locale string) string {
    if user.Preferences.EnableAlternativeCalendar && reminder.CalendarType != "" && reminder.CalendarType != "gregorian" {
        // e.g. "农历八月十五" — uses calendar.Get(...).Display(date, locale)
        return calendar.Get(reminder.CalendarType).Display(reminder.OriginalDate, locale)
    }
    return formatGregorianDate(reminder.NextFireAt, locale)
}
```

`reminder_scheduler.go:82-87`: extend template variables to include <code v-pre>{{.Date}}</code>; update `reminder.body_with_contact` / `reminder.body_no_contact` i18n keys to include the date placeholder.

New tests:
- `reminder_date_format_test.go::TestLunarWithAlternativeCalendarShowsLunarString`
- `reminder_date_format_test.go::TestLunarWithoutAlternativeCalendarShowsGregorian`
- `reminder_date_format_test.go::TestGregorianAlwaysShowsGregorian`
- `reminder_date_format_test.go::TestLocaleControlsMonthNames`
- `reminder_scheduler_test.go::TestReminderBodyIncludesFormattedDate`

Commit message: `fix(reminders): honor user timezone and calendar in scheduling and notifications`

### Commit 4 — Frontend i18n linkage

**A1 — Preferences.locale ↔ i18n**

- `web/src/pages/settings/Preferences.tsx`: on save success, call `i18n.changeLanguage(normalizeLanguageCode(prefs.locale))`.
- `web/src/App.tsx` (or wherever auth state hydrates): after fetching the user's prefs, call `i18n.changeLanguage(normalizeLanguageCode(prefs.locale))` once.

New tests:
- `web/src/test/PreferencesLocaleSync.test.tsx::TestSaveLocaleChangesI18nLanguage`
- `web/src/test/PreferencesLocaleSync.test.tsx::TestBootstrapAppliesStoredLocale`

**A5 — AntD ConfigProvider locale**

- New `web/src/i18n/antdLocale.ts`: maps `en → enUS`, `zh → zhCN`, `es → esES`.
- `web/src/ThemedApp.tsx`: wrap children in `<ConfigProvider locale={antdLocaleFor(currentCode)}>`; re-render when `i18n.language` changes.

New test: `web/src/test/AntdLocale.test.tsx::TestDatePickerMonthIsZhWhenI18nIsZh`.

**C9 — `check-i18n-keys.mjs` covers es**

- `web/scripts/check-i18n-keys.mjs:30-31`: change pairwise diff from `[en, zh]` to all pairs of `[en, zh, es]`.
- CI execution validates.

**C10 — es.json key + namespace parity**

- Run an AI translation pass to fill ~174 missing keys.
- Rename namespaces in es.json: `calendar_types` → `calendar`, `oauth_login` → `oauth`.
- Validated by passing C9 lint.

**C11 — Admin toasts via `t()`**

- `web/src/pages/admin/Backups.tsx:92,101`: route through `t()`.
- `web/src/pages/admin/OAuthProviders.tsx:90,101,111`: same.
- Add corresponding keys to en/zh/es.

Test: extend `web/e2e/admin.spec.ts` with a zh assertion (toast text in Chinese after switching language).

**C12 / E22 — dayjs localization**

- New `web/src/utils/dayjsLocale.ts`:
  ```ts
  import dayjs from "dayjs";
  import "dayjs/locale/zh-cn";
  import "dayjs/locale/es";
  export function applyDayjsLocale(code: SupportedLanguageCode) {
    dayjs.locale(code === "zh" ? "zh-cn" : code);
  }
  ```
- Wire `i18n.on("languageChanged", applyDayjsLocale)` once at boot.

New test: `web/src/test/DayjsLocale.test.ts::TestDayjsFormatRespectsLanguage` (zh → "九月", es → "septiembre").

Commit message: `fix(web): sync locale across i18n, antd, and dayjs`

### Commit 5 — Frontend calendar + timezone

**D16 — Replace raw DatePicker on personal-date forms**

Swap `<DatePicker />` for `<CalendarDatePicker />` on:
- `web/src/pages/contacts/modules/LifeEventsModule.tsx:332,349`
- `web/src/pages/journal/JournalDetail.tsx:606`
- `web/src/pages/tasks/TaskEditModal.tsx:356`
- `web/src/pages/vaults/VaultDetail.tsx:752,1222` (confirm at edit time that both lines are personal-date fields; if either is administrative — e.g., vault creation timestamp — skip it and document why)

Not touched (administrative or technical dates):
- `AddressesModule.tsx:257,264`, `CallsModule.tsx:213`, `ApiTokens.tsx:233`.

New E2E scenarios in `web/e2e/calendar.spec.ts`:
- "life event with lunar date persists and re-renders correctly"
- "task with lunar due date persists and re-renders correctly"

**D17 — useDateFormat applies user timezone**

`web/src/utils/dateFormat.ts`: read user timezone from the prefs store (via hook or context), pass through `dayjs.tz(date, tz)` before formatting.

New test: `web/src/test/DateFormatTimezone.test.tsx::TestUseDateFormatUsesUserTimezone`.

**E20 — Preferences uses `Intl.supportedValuesOf('timeZone')`**

`web/src/pages/settings/Preferences.tsx`: replace the hardcoded ~12-zone list with `Intl.supportedValuesOf('timeZone').map(z => ({ value: z, label: z }))`. Fall back to the hardcoded list if `Intl.supportedValuesOf` is undefined (older browsers).

New test: `web/src/test/PreferencesTimezone.test.tsx::TestTimezoneListContainsCommonZones` (contains `Asia/Tokyo`, `Europe/Madrid`).

Commit message: `fix(web): support lunar input on personal-date forms and user timezone formatting`

## 6. Test Summary

| Commit | Go unit | Frontend unit (vitest) | E2E (Playwright) |
|---|---|---|---|
| 1 | 0 | 0 (existing) | 0 |
| 2 | ~12 | 0 | 0 |
| 3 | ~10 | 0 | 0 |
| 4 | 0 | ~5 | admin.spec.ts +zh assert |
| 5 | 0 | ~3 | calendar.spec.ts +2 scenarios |

Total new tests: ~22 Go + ~8 frontend unit + 3 E2E.

## 7. Risk & Rollback

- Each commit is independently revertible via `git revert`.
- Commit 2's `B6` migration is forward-only (adds a column) — GORM `AutoMigrate` will create it on startup; rolling back the commit would leave an unused column, which is harmless.
- Commit 3's CalDAV change replaces RRULE with RDATE for lunar entries — calendar clients that synced earlier will see the old RRULE replaced by RDATE on next sync. Apple Calendar / Thunderbird handle this gracefully.
- Commit 4's es.json AI translation may have minor wording quirks; subsequent native-speaker passes can adjust without code changes.

## 8. Open Questions

None at time of writing. Any uncertainties surfaced during implementation should be flagged and resolved before the affected commit lands.

## 9. References

- Audit report 1 — broad i18n + calendar (sections A–E, 22 findings).
- Audit report 2 — reminder multi-calendar (Bug 1 & 2).
- Previous bug fixes (Commit 1 base) — `LanguageSwitcher`, Accept-Language interceptor.
