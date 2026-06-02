package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

// TestRescheduleRecurringReminderRespectsUserTimezone guards the bug where
// the scheduler used the server's local timezone for the "9 AM next year"
// computation. A user in Asia/Tokyo whose server runs UTC was getting
// reminders fire at 09:00 UTC (18:00 their time) instead of 09:00 their
// time (00:00 UTC). After the fix, the rescheduled fire time must reflect
// the user's preference.
func TestRescheduleRecurringReminderRespectsUserTimezone(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	resp, err := auth.Register(dto.RegisterRequest{
		FirstName: "Tokyo",
		LastName:  "User",
		Email:     "tokyo-user@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	tz := "Asia/Tokyo"
	if err := db.Model(&models.User{}).Where("id = ?", resp.User.ID).
		Update("timezone", &tz).Error; err != nil {
		t.Fatalf("set tz: %v", err)
	}

	vault := models.Vault{Name: "Vault", AccountID: resp.User.AccountID}
	if err := db.Create(&vault).Error; err != nil {
		t.Fatalf("vault: %v", err)
	}
	first, last := "Anniversary", "Person"
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
	preferredTime := "16:30"
	channel := models.UserNotificationChannel{
		UserID:        &resp.User.ID,
		Type:          "email",
		Content:       "tokyo-user@example.com",
		PreferredTime: &preferredTime,
		Active:        true,
		VerifiedAt:    &now,
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

	var rescheduled models.ContactReminderScheduled
	if err := db.Where("contact_reminder_id = ? AND triggered_at IS NULL", reminder.ID).
		Order("scheduled_at ASC").First(&rescheduled).Error; err != nil {
		t.Fatalf("find rescheduled row: %v", err)
	}

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load Asia/Tokyo: %v", err)
	}
	local := rescheduled.ScheduledAt.In(tokyo)
	if local.Hour() != 16 || local.Minute() != 30 {
		t.Errorf("rescheduled fire time = %s; expected 16:30 in Asia/Tokyo (got %02d:%02d local)",
			rescheduled.ScheduledAt.Format(time.RFC3339), local.Hour(), local.Minute())
	}
}
