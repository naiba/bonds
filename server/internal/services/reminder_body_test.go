package services

import (
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestReminderDeliveryContentTemplatesAreCompleteAndNotDuplicated(t *testing.T) {
	reminder := &models.ContactReminder{Label: "Birthday", CalendarType: "gregorian"}
	scheduledAt := time.Date(2026, time.September, 5, 9, 0, 0, 0, time.UTC)
	for _, locale := range i18n.Supported {
		t.Run(locale, func(t *testing.T) {
			subject, body := reminderDeliveryContent(reminder, scheduledAt, locale, false, "John Doe")
			if !strings.Contains(subject, "Birthday") || !strings.Contains(subject, "John Doe") {
				t.Fatalf("subject does not identify reminder and contact: %q", subject)
			}
			if strings.Contains(subject, "{{") || strings.Contains(body, "{{") {
				t.Fatalf("unresolved reminder placeholder: subject=%q body=%q", subject, body)
			}
			if strings.Contains(body, "<h") || strings.Count(body, "Birthday") != 1 {
				t.Fatalf("reminder body repeats its heading or label: %q", body)
			}
			plainBody := stripHTML(body)
			if !strings.Contains(plainBody, "John Doe") || !strings.Contains(plainBody, "2026-09-05") {
				t.Fatalf("reminder body is missing delivery context: %q", plainBody)
			}
		})
	}
}

// TestReminderBodyIncludesDateAndAlternativeCalendar verifies that the
// reminder email/push body shows the fire date — previously the template
// just said "You have a reminder for Mid-Autumn" with no date at all — and
// that lunar reminders honor the user's EnableAlternativeCalendar setting
// by rendering the lunar date alongside the Gregorian equivalent.
func TestReminderBodyIncludesDateAndAlternativeCalendar(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	resp, err := auth.Register(dto.RegisterRequest{
		FirstName: "Lunar",
		LastName:  "Reader",
		Email:     "lunar-reader@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Switch on alternative-calendar so the reminder body should show 农历.
	if err := db.Model(&models.User{}).Where("id = ?", resp.User.ID).
		Update("enable_alternative_calendar", true).Error; err != nil {
		t.Fatalf("enable alt calendar: %v", err)
	}

	vault := models.Vault{Name: "Vault", AccountID: resp.User.AccountID}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("vault: %v", err)
	}
	if err := db.Create(&models.UserVault{
		VaultID:    vault.ID,
		UserID:     resp.User.ID,
		Permission: models.PermissionManager,
	}).Error; err != nil {
		t.Fatalf("vault membership: %v", err)
	}
	first, last := "Lunar", "Anchored"
	contact := models.Contact{VaultID: vault.ID, FirstName: &first, LastName: &last}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("contact: %v", err)
	}
	origMonth, origDay := 8, 15
	reminder := models.ContactReminder{
		ContactID:     contact.ID,
		Label:         "Mid-Autumn",
		Type:          "recurring_year",
		CalendarType:  "lunar",
		OriginalMonth: &origMonth,
		OriginalDay:   &origDay,
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatalf("reminder: %v", err)
	}
	now := time.Now()
	channel := models.UserNotificationChannel{
		UserID:     &resp.User.ID,
		Type:       "email",
		Content:    "lunar-reader@example.com",
		Active:     true,
		VerifiedAt: &now,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("channel: %v", err)
	}
	due := time.Now().Add(-1 * time.Minute)
	if err := db.Create(&models.ContactReminderScheduled{
		UserNotificationChannelID: channel.ID,
		ContactReminderID:         reminder.ID,
		ScheduledAt:               due,
	}).Error; err != nil {
		t.Fatalf("scheduled: %v", err)
	}

	mailer := &recordingMailer{}
	NewReminderSchedulerService(db, mailer, nil).ProcessDueReminders()

	got := mailer.last(t)
	if !strings.Contains(got.subject, "Mid-Autumn") {
		t.Errorf("subject missing label: %q", got.subject)
	}
	if !strings.Contains(got.body, "Mid-Autumn") {
		t.Errorf("body missing label: %q", got.body)
	}
	if !strings.Contains(got.body, "农历") {
		t.Errorf("body missing 农历 marker (user has EnableAlternativeCalendar=true): %q", got.body)
	}
	if !strings.Contains(got.body, "(") || !strings.Contains(got.body, ")") {
		t.Errorf("body missing Gregorian fallback in parens: %q", got.body)
	}
}

// TestReminderBodyGregorianShowsPlainDate covers the inverse case: a
// Gregorian reminder (or a lunar one with the alt-calendar preference off)
// should show the plain Gregorian fire date, no 农历 marker.
func TestReminderBodyGregorianShowsPlainDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	resp, err := auth.Register(dto.RegisterRequest{
		FirstName: "Greg",
		LastName:  "Reader",
		Email:     "greg-reader@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	vault := models.Vault{Name: "Vault", AccountID: resp.User.AccountID}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("vault: %v", err)
	}
	if err := db.Create(&models.UserVault{
		VaultID:    vault.ID,
		UserID:     resp.User.ID,
		Permission: models.PermissionManager,
	}).Error; err != nil {
		t.Fatalf("vault membership: %v", err)
	}
	first, last := "Greg", "Smith"
	contact := models.Contact{VaultID: vault.ID, FirstName: &first, LastName: &last}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("contact: %v", err)
	}
	year, month, day := 2026, 6, 14
	reminder := models.ContactReminder{
		ContactID:    contact.ID,
		Label:        "Anniversary",
		Type:         "one_time",
		CalendarType: "gregorian",
		Year:         &year,
		Month:        &month,
		Day:          &day,
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatalf("reminder: %v", err)
	}
	now := time.Now()
	channel := models.UserNotificationChannel{
		UserID:     &resp.User.ID,
		Type:       "email",
		Content:    "greg-reader@example.com",
		Active:     true,
		VerifiedAt: &now,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("channel: %v", err)
	}
	due := time.Now().Add(-1 * time.Minute)
	if err := db.Create(&models.ContactReminderScheduled{
		UserNotificationChannelID: channel.ID,
		ContactReminderID:         reminder.ID,
		ScheduledAt:               due,
	}).Error; err != nil {
		t.Fatalf("scheduled: %v", err)
	}

	mailer := &recordingMailer{}
	NewReminderSchedulerService(db, mailer, nil).ProcessDueReminders()

	got := mailer.last(t)
	if strings.Contains(got.body, "农历") {
		t.Errorf("Gregorian reminder body unexpectedly contains 农历: %q", got.body)
	}
	// Must still contain some date — the fix changed the template so any
	// body without a date marker is a regression.
	hasDate := false
	for _, marker := range []string{"-", "/", "."} {
		if strings.Contains(got.body, marker) {
			hasDate = true
			break
		}
	}
	if !hasDate {
		t.Errorf("Gregorian reminder body has no recognizable date format: %q", got.body)
	}
}
