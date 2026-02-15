package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// mockMailer implements the Mailer interface for testing.
// It records all Send calls and can return a configurable error.
type mockMailer struct {
	calls   []mockMailerCall
	sendErr error // if set, Send() returns this error
}

type mockMailerCall struct {
	To, Subject, Body string
}

func (m *mockMailer) Send(to, subject, body string) error {
	m.calls = append(m.calls, mockMailerCall{To: to, Subject: subject, Body: body})
	return m.sendErr
}

func (m *mockMailer) Close() {}

type reminderSchedulerTestContext struct {
	svc       *ReminderSchedulerService
	db        *gorm.DB
	mailer    *mockMailer
	userID    string
	accountID string
	vaultID   string
	contactID string
}

func setupReminderSchedulerTest(t *testing.T) *reminderSchedulerTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "scheduler-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John", LastName: "Doe"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// GORM many2many creates this as a 2-column pivot; recreate with full schema.
	db.Exec("DROP TABLE IF EXISTS contact_reminder_scheduled")
	db.Exec(`CREATE TABLE contact_reminder_scheduled (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_notification_channel_id INTEGER NOT NULL,
		contact_reminder_id INTEGER NOT NULL,
		scheduled_at DATETIME NOT NULL,
		triggered_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	mailer := &mockMailer{}
	svc := NewReminderSchedulerService(db, mailer)

	return &reminderSchedulerTestContext{
		svc:       svc,
		db:        db,
		mailer:    mailer,
		userID:    resp.User.ID,
		accountID: resp.User.AccountID,
		vaultID:   vault.ID,
		contactID: contact.ID,
	}
}

// createEmailChannel creates an active email notification channel for the user.
func (ctx *reminderSchedulerTestContext) createEmailChannel(t *testing.T) *models.UserNotificationChannel {
	t.Helper()
	ch := models.UserNotificationChannel{
		UserID:  &ctx.userID,
		Type:    "email",
		Content: "scheduler-test@example.com",
	}
	if err := ctx.db.Create(&ch).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}
	// Active defaults to false due to GORM zero-value, explicitly update
	if err := ctx.db.Model(&ch).Update("active", true).Error; err != nil {
		t.Fatalf("update channel active failed: %v", err)
	}
	return &ch
}

// createReminder creates a ContactReminder with the given type and optional frequency.
func (ctx *reminderSchedulerTestContext) createReminder(t *testing.T, reminderType string, freqNum *int) *models.ContactReminder {
	t.Helper()
	reminder := models.ContactReminder{
		ContactID:       ctx.contactID,
		Label:           "Test Reminder",
		Type:            reminderType,
		FrequencyNumber: freqNum,
	}
	if err := ctx.db.Create(&reminder).Error; err != nil {
		t.Fatalf("create reminder failed: %v", err)
	}
	return &reminder
}

// createScheduled creates a ContactReminderScheduled entry.
func (ctx *reminderSchedulerTestContext) createScheduled(t *testing.T, reminderID, channelID uint, scheduledAt time.Time, triggeredAt *time.Time) *models.ContactReminderScheduled {
	t.Helper()
	s := models.ContactReminderScheduled{
		UserNotificationChannelID: channelID,
		ContactReminderID:         reminderID,
		ScheduledAt:               scheduledAt,
		TriggeredAt:               triggeredAt,
	}
	if err := ctx.db.Create(&s).Error; err != nil {
		t.Fatalf("create scheduled failed: %v", err)
	}
	return &s
}

func freqPtr(v int) *int {
	return &v
}

// --- Test 1: No due reminders → no action ---

func TestProcessDueReminders_NoDueReminders(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)

	// No scheduled reminders exist at all
	ctx.svc.ProcessDueReminders()

	if len(ctx.mailer.calls) != 0 {
		t.Errorf("expected 0 mailer calls, got %d", len(ctx.mailer.calls))
	}

	var count int64
	ctx.db.Model(&models.UserNotificationSent{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 sent records, got %d", count)
	}
}

// --- Test 2: One due reminder → sends email, marks triggered ---

func TestProcessDueReminders_OneDueReminder(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	scheduled := ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	// Verify mailer was called once with correct args
	if len(ctx.mailer.calls) != 1 {
		t.Fatalf("expected 1 mailer call, got %d", len(ctx.mailer.calls))
	}
	if ctx.mailer.calls[0].To != "scheduler-test@example.com" {
		t.Errorf("expected to='scheduler-test@example.com', got '%s'", ctx.mailer.calls[0].To)
	}
	if ctx.mailer.calls[0].Subject != "Reminder: Test Reminder" {
		t.Errorf("expected subject='Reminder: Test Reminder', got '%s'", ctx.mailer.calls[0].Subject)
	}

	// Verify TriggeredAt is set
	var updated models.ContactReminderScheduled
	if err := ctx.db.First(&updated, scheduled.ID).Error; err != nil {
		t.Fatalf("failed to reload scheduled: %v", err)
	}
	if updated.TriggeredAt == nil {
		t.Fatal("expected TriggeredAt to be set")
	}

	// Verify UserNotificationSent created with no error
	var sent models.UserNotificationSent
	if err := ctx.db.Where("user_notification_channel_id = ?", ch.ID).First(&sent).Error; err != nil {
		t.Fatalf("expected UserNotificationSent record: %v", err)
	}
	if sent.Error != nil {
		t.Errorf("expected no error on sent notification, got '%s'", *sent.Error)
	}
	if sent.SubjectLine != "Reminder: Test Reminder" {
		t.Errorf("expected subject_line='Reminder: Test Reminder', got '%s'", sent.SubjectLine)
	}

	// Verify ContactReminder.NumberTimesTriggered incremented
	var updatedReminder models.ContactReminder
	if err := ctx.db.First(&updatedReminder, reminder.ID).Error; err != nil {
		t.Fatalf("failed to reload reminder: %v", err)
	}
	if updatedReminder.NumberTimesTriggered != 1 {
		t.Errorf("expected NumberTimesTriggered=1, got %d", updatedReminder.NumberTimesTriggered)
	}
	if updatedReminder.LastTriggeredAt == nil {
		t.Error("expected LastTriggeredAt to be set")
	}
}

// --- Test 3: Already triggered → skipped ---

func TestProcessDueReminders_AlreadyTriggered(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	triggeredAt := time.Now().Add(-5 * time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, &triggeredAt)

	ctx.svc.ProcessDueReminders()

	if len(ctx.mailer.calls) != 0 {
		t.Errorf("expected 0 mailer calls for already-triggered, got %d", len(ctx.mailer.calls))
	}
}

// --- Test 4: Inactive channel → skipped ---

func TestProcessDueReminders_InactiveChannel(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	// Create an inactive channel (Active defaults to false, don't update)
	ch := models.UserNotificationChannel{
		UserID:  &ctx.userID,
		Type:    "email",
		Content: "scheduler-test@example.com",
	}
	if err := ctx.db.Create(&ch).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	reminder := ctx.createReminder(t, "one_time", nil)
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	if len(ctx.mailer.calls) != 0 {
		t.Errorf("expected 0 mailer calls for inactive channel, got %d", len(ctx.mailer.calls))
	}

	// Verify no UserNotificationSent record created
	var count int64
	ctx.db.Model(&models.UserNotificationSent{}).Where("user_notification_channel_id = ?", ch.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 sent records for inactive channel, got %d", count)
	}
}

// --- Test 5: Send failure → fails counter incremented, UserNotificationSent records error ---

func TestProcessDueReminders_SendFailure(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ctx.mailer.sendErr = errors.New("SMTP connection refused")
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "one_time", nil)
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	// Verify mailer was called (it records even on failure)
	if len(ctx.mailer.calls) != 1 {
		t.Fatalf("expected 1 mailer call, got %d", len(ctx.mailer.calls))
	}

	// Verify UserNotificationSent has error
	var sent models.UserNotificationSent
	if err := ctx.db.Where("user_notification_channel_id = ?", ch.ID).First(&sent).Error; err != nil {
		t.Fatalf("expected UserNotificationSent record: %v", err)
	}
	if sent.Error == nil {
		t.Fatal("expected error on sent notification, got nil")
	}
	if *sent.Error != "SMTP connection refused" {
		t.Errorf("expected error='SMTP connection refused', got '%s'", *sent.Error)
	}

	// Verify channel fails incremented
	var updatedCh models.UserNotificationChannel
	if err := ctx.db.First(&updatedCh, ch.ID).Error; err != nil {
		t.Fatalf("failed to reload channel: %v", err)
	}
	if updatedCh.Fails != 1 {
		t.Errorf("expected channel.Fails=1, got %d", updatedCh.Fails)
	}
	// Channel should still be active
	if !updatedCh.Active {
		t.Error("expected channel to remain active after 1 failure")
	}
}

// --- Test 6: Auto-disable after maxChannelFails ---

func TestProcessDueReminders_AutoDisableAfterMaxFails(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ctx.mailer.sendErr = errors.New("permanent failure")
	ch := ctx.createEmailChannel(t)

	// Set fails to maxChannelFails - 1 so this failure reaches the threshold
	if err := ctx.db.Model(ch).Update("fails", maxChannelFails-1).Error; err != nil {
		t.Fatalf("failed to set fails: %v", err)
	}

	reminder := ctx.createReminder(t, "one_time", nil)
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	var updatedCh models.UserNotificationChannel
	if err := ctx.db.First(&updatedCh, ch.ID).Error; err != nil {
		t.Fatalf("failed to reload channel: %v", err)
	}
	if updatedCh.Fails != maxChannelFails {
		t.Errorf("expected channel.Fails=%d, got %d", maxChannelFails, updatedCh.Fails)
	}
	if updatedCh.Active {
		t.Error("expected channel to be auto-disabled after reaching maxChannelFails")
	}
}

// --- Test 7: Recurring week → new scheduled entry 7 days later ---

func TestRescheduleRecurringWeek(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "recurring_week", freqPtr(1))
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	// Verify a new scheduled entry was created
	var allScheduled []models.ContactReminderScheduled
	if err := ctx.db.Where("contact_reminder_id = ?", reminder.ID).Find(&allScheduled).Error; err != nil {
		t.Fatalf("failed to query scheduled: %v", err)
	}
	if len(allScheduled) != 2 {
		t.Fatalf("expected 2 scheduled entries (original + rescheduled), got %d", len(allScheduled))
	}

	// Find the new untriggered future entry
	var found bool
	for _, s := range allScheduled {
		if s.TriggeredAt == nil && s.ScheduledAt.After(time.Now()) {
			found = true
			expectedMin := time.Now().Add(6 * 24 * time.Hour)
			expectedMax := time.Now().Add(8 * 24 * time.Hour)
			if s.ScheduledAt.Before(expectedMin) || s.ScheduledAt.After(expectedMax) {
				t.Errorf("expected rescheduled time ~7 days from now, got %v", s.ScheduledAt)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find a new future scheduled entry for weekly recurrence")
	}
}

// --- Test 8: Recurring month → new scheduled entry 1 month later ---

func TestRescheduleRecurringMonth(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "recurring_month", freqPtr(1))
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	var allScheduled []models.ContactReminderScheduled
	if err := ctx.db.Where("contact_reminder_id = ?", reminder.ID).Find(&allScheduled).Error; err != nil {
		t.Fatalf("failed to query scheduled: %v", err)
	}
	if len(allScheduled) != 2 {
		t.Fatalf("expected 2 scheduled entries, got %d", len(allScheduled))
	}

	var found bool
	for _, s := range allScheduled {
		if s.TriggeredAt == nil && s.ScheduledAt.After(time.Now()) {
			found = true
			expectedMin := time.Now().Add(27 * 24 * time.Hour)
			expectedMax := time.Now().Add(32 * 24 * time.Hour)
			if s.ScheduledAt.Before(expectedMin) || s.ScheduledAt.After(expectedMax) {
				t.Errorf("expected rescheduled time ~1 month from now, got %v", s.ScheduledAt)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find a new future scheduled entry for monthly recurrence")
	}
}

// --- Test 9: Recurring year → new scheduled entry 1 year later ---

func TestRescheduleRecurringYear(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)
	reminder := ctx.createReminder(t, "recurring_year", freqPtr(1))
	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	var allScheduled []models.ContactReminderScheduled
	if err := ctx.db.Where("contact_reminder_id = ?", reminder.ID).Find(&allScheduled).Error; err != nil {
		t.Fatalf("failed to query scheduled: %v", err)
	}
	if len(allScheduled) != 2 {
		t.Fatalf("expected 2 scheduled entries, got %d", len(allScheduled))
	}

	var found bool
	for _, s := range allScheduled {
		if s.TriggeredAt == nil && s.ScheduledAt.After(time.Now()) {
			found = true
			expectedMin := time.Now().Add(364 * 24 * time.Hour)
			expectedMax := time.Now().Add(367 * 24 * time.Hour)
			if s.ScheduledAt.Before(expectedMin) || s.ScheduledAt.After(expectedMax) {
				t.Errorf("expected rescheduled time ~1 year from now, got %v", s.ScheduledAt)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find a new future scheduled entry for yearly recurrence")
	}
}

func TestRescheduleRecurringYearLunar(t *testing.T) {
	ctx := setupReminderSchedulerTest(t)
	ch := ctx.createEmailChannel(t)

	origDay := 15
	origMonth := 1
	reminder := &models.ContactReminder{
		ContactID:     ctx.contactID,
		Label:         "Lunar Birthday",
		Type:          "recurring_year",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
	}
	if err := ctx.db.Create(reminder).Error; err != nil {
		t.Fatalf("create reminder failed: %v", err)
	}

	pastTime := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	ctx.createScheduled(t, reminder.ID, ch.ID, pastTime, nil)

	ctx.svc.ProcessDueReminders()

	var allScheduled []models.ContactReminderScheduled
	if err := ctx.db.Where("contact_reminder_id = ?", reminder.ID).Find(&allScheduled).Error; err != nil {
		t.Fatalf("failed to query scheduled: %v", err)
	}
	if len(allScheduled) != 2 {
		t.Fatalf("expected 2 scheduled entries (original + rescheduled), got %d", len(allScheduled))
	}

	var found bool
	for _, s := range allScheduled {
		if s.TriggeredAt == nil && s.ScheduledAt.After(time.Now()) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find a new future scheduled entry for lunar yearly recurrence")
	}
}

func TestCalcNextYearlyScheduleLunar(t *testing.T) {
	origDay := 15
	origMonth := 8
	reminder := &models.ContactReminder{
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nextTime, ok := calcNextYearlySchedule(reminder, now)
	if !ok {
		t.Fatal("expected calcNextYearlySchedule to return ok=true for lunar")
	}
	if nextTime.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", nextTime.Year())
	}
	if nextTime.Month() < 9 || nextTime.Month() > 10 {
		t.Errorf("expected month 9-10 for lunar 八月十五, got %d", nextTime.Month())
	}
}

func TestCalcNextYearlyScheduleGregorian(t *testing.T) {
	origDay := 15
	origMonth := 6
	reminder := &models.ContactReminder{
		CalendarType:  "gregorian",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, ok := calcNextYearlySchedule(reminder, now)
	if ok {
		t.Error("expected calcNextYearlySchedule to return ok=false for gregorian (uses standard path)")
	}
}

func TestCalcNextYearlyScheduleNoOriginal(t *testing.T) {
	reminder := &models.ContactReminder{
		CalendarType: "lunar",
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, ok := calcNextYearlySchedule(reminder, now)
	if ok {
		t.Error("expected ok=false when OriginalMonth/OriginalDay are nil")
	}
}

func TestBuildContactName(t *testing.T) {
	first := "Alice"
	last := "Smith"

	tests := []struct {
		name     string
		contact  models.Contact
		expected string
	}{
		{
			name:     "first and last",
			contact:  models.Contact{FirstName: &first, LastName: &last},
			expected: "Alice Smith",
		},
		{
			name:     "first only",
			contact:  models.Contact{FirstName: &first},
			expected: "Alice",
		},
		{
			name:     "last only",
			contact:  models.Contact{LastName: &last},
			expected: "Smith",
		},
		{
			name:     "neither",
			contact:  models.Contact{},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildContactName(&tt.contact)
			if result != tt.expected {
				t.Errorf("buildContactName() = %q, want %q", result, tt.expected)
			}
		})
	}
}
