package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var monicaScheduledReminderFixture = []byte(`{"version":"1.0-preview.1","account":{"data":[{"type":"contacts","values":[{"uuid":"synthetic-reminder-owner","properties":{"first_name":"Duplicate"},"data":[{"type":"reminders","values":[{"uuid":"synthetic-reminder","properties":{"title":"Call annually","description":"Discuss the annual gathering","frequency_type":"year","frequency_number":1,"initial_date":"2027-01-01"}}]}]}]}]}}`)

func TestMonicaReminderSchedulingFailureRollsBackRecord(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	hook := "monica:reject_initial_schedule"
	injected := errors.New("initial schedule unavailable")
	if err := svc.db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_reminder_scheduled" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Create().Remove(hook)
	response, err := NewMonicaImportService(svc.db, "").Import(vault, user, monicaScheduledReminderFixture)
	if err != nil {
		t.Fatal(err)
	}
	if response.ImportedReminders != 0 || response.ImportedNotes != 0 || len(response.Errors) != 1 {
		t.Fatalf("partial reminder reported successful: %+v", response)
	}
	for _, model := range []any{&models.ContactReminder{}, &models.ContactReminderScheduled{}, &models.Note{}} {
		var count int64
		if err := svc.db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("failed reminder left %T rows: %d", model, count)
		}
	}
	// Successful contact import survives a per-record failure, and retry can
	// finish the reminder without recreating its contact.
	if err := svc.db.Callback().Create().Remove(hook); err != nil {
		t.Fatal(err)
	}
	retry, err := NewMonicaImportService(svc.db, "").Import(vault, user, monicaScheduledReminderFixture)
	if err != nil {
		t.Fatal(err)
	}
	if retry.ImportedContacts != 0 || retry.ImportedReminders != 1 || retry.ImportedNotes != 1 || len(retry.Errors) != 0 {
		t.Fatalf("retry did not complete reminder: %+v", retry)
	}
	var reminders []models.ContactReminder
	if err := svc.db.Find(&reminders).Error; err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 1 {
		t.Fatalf("expected one reminder: %d", len(reminders))
	}
	assertMonicaReminderSchedules(t, svc.db, reminders[0].ID, user)
}

func assertMonicaReminderSchedules(t *testing.T, db *gorm.DB, reminderID uint, userID string) {
	t.Helper()
	var channels []models.UserNotificationChannel
	if err := db.Where("user_id = ? AND active = ?", userID, true).Find(&channels).Error; err != nil {
		t.Fatal(err)
	}
	if len(channels) == 0 {
		t.Fatal("fixture has no active channels")
	}
	var schedules []models.ContactReminderScheduled
	if err := db.Where("contact_reminder_id = ?", reminderID).Find(&schedules).Error; err != nil {
		t.Fatal(err)
	}
	if len(schedules) != len(channels) {
		t.Fatalf("expected %d schedules, got %d", len(channels), len(schedules))
	}
	for _, channel := range channels {
		found := false
		for _, schedule := range schedules {
			if schedule.UserNotificationChannelID == channel.ID && !schedule.ScheduledAt.IsZero() && schedule.TriggeredAt == nil {
				found = true
			}
		}
		if !found {
			t.Fatalf("channel %d missing pending schedule", channel.ID)
		}
	}
}

func TestMonicaReminderSchedulingHoldsOwnerUntilCommit(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent parent row lock")
	}
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Update("distant_uuid", "synthetic-reminder-owner").Error; err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
	boundary, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "monica:pause_initial_schedule"
	if err := svc.db.Callback().Query().Before("gorm:query").Register(hook, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*models.Contact); !ok || paused.Load() {
			return
		}
		var count int64
		if err := tx.Session(&gorm.Session{NewDB: true}).Model(&models.ContactReminder{}).Where("contact_id = ?", source.ID).Count(&count).Error; err != nil {
			tx.AddError(err)
			return
		}
		if count > 0 && paused.CompareAndSwap(false, true) {
			close(boundary)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	type result struct {
		response *dto.MonicaImportResponse
		err      error
	}
	imported := make(chan result, 1)
	go func() {
		r, e := NewMonicaImportService(svc.db.WithContext(ctx), "").Import(vault, user, monicaScheduledReminderFixture)
		imported <- result{r, e}
	}()
	awaitMergeBoundarySignal(t, boundary)
	merged := make(chan error, 1)
	go func() {
		_, e := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vault, user, req)
		merged <- e
	}()
	waiting := waitForContactMergeLock(svc.db)
	close(resume)
	var outcome result
	select {
	case outcome = <-imported:
	case <-time.After(15 * time.Second):
		t.Fatal("import did not finish")
	}
	mergeErr := awaitMergeBoundaryError(t, merged)
	if outcome.err != nil || outcome.response == nil {
		t.Fatalf("import failed: %v", outcome.err)
	}
	if outcome.response.ImportedReminders != 1 || outcome.response.ImportedNotes != 1 || len(outcome.response.Errors) != 0 {
		t.Fatalf("incomplete import: %+v", outcome.response)
	}
	if !waiting || !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
		t.Fatalf("merge did not wait and request fresh review: waited=%v err=%v", waiting, mergeErr)
	}
	var reminder models.ContactReminder
	if err := svc.db.Where("contact_id = ?", source.ID).First(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	assertMonicaReminderSchedules(t, svc.db, reminder.ID, user)
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	var moved models.ContactReminder
	if err := svc.db.First(&moved, reminder.ID).Error; err != nil {
		t.Fatal(err)
	}
	if moved.ContactID != target.ID {
		t.Fatal("reminder did not move to survivor")
	}
	assertMonicaReminderSchedules(t, svc.db, reminder.ID, user)
	var note models.Note
	if err := svc.db.Where("source_uuid = ?", "synthetic-reminder").First(&note).Error; err != nil {
		t.Fatal(err)
	}
	if note.ContactID != target.ID || note.Body != "Discuss the annual gathering" {
		t.Fatalf("description lost: %+v", note)
	}
}
