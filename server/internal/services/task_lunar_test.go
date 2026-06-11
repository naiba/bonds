package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
)

// TestCreateTaskWithLunarDueDate verifies the lunar-create contract for tasks
// mirrors LifeEvent/Post: original_* persists, DueAt gets re-projected so kanban
// position-by-due-date stays consistent across calendars.
func TestCreateTaskWithLunarDueDate(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	placeholder := time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC)
	day, month, year := 15, 8, 2025
	task, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:         "Send mooncakes",
		DueAt:         &placeholder,
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if task.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got %q", task.CalendarType)
	}
	if task.OriginalDay == nil || *task.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay=15, got %v", task.OriginalDay)
	}
	if task.OriginalMonth == nil || *task.OriginalMonth != 8 {
		t.Errorf("Expected OriginalMonth=8, got %v", task.OriginalMonth)
	}
	if task.OriginalYear == nil || *task.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear=2025, got %v", task.OriginalYear)
	}
	if task.DueAt == nil {
		t.Fatal("Expected DueAt to remain non-nil after projection")
	}
	// Mid-Autumn 2025 lunar 8/15 = 2025-10-06 gregorian.
	if task.DueAt.Year() != 2025 || task.DueAt.Month() != time.October {
		t.Errorf("Expected DueAt projected into Oct 2025, got %v", task.DueAt)
	}
	if task.DueAt.Hour() != 14 {
		t.Errorf("Expected clock field preserved (14:00), got %v", task.DueAt)
	}
}

// TestCreateTaskWithoutDueDateIgnoresCalendarType pins the nil-DueAt invariant:
// a dateless task has no calendar semantics, so even an explicit lunar request
// must collapse to a clean gregorian record. Stops orphaned Original* metadata
// from silently riding along and confusing the reminder scheduler if a later
// edit reattaches a due date.
func TestCreateTaskWithoutDueDateIgnoresCalendarType(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)

	day, month, year := 1, 1, 2025
	task, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label:         "Someday/maybe",
		DueAt:         nil,
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if task.CalendarType != "gregorian" {
		t.Errorf("Expected CalendarType pinned to 'gregorian' for dateless task, got %q", task.CalendarType)
	}
	if task.OriginalDay != nil || task.OriginalMonth != nil || task.OriginalYear != nil {
		t.Errorf("Expected Original* cleared for dateless task, got %v/%v/%v",
			task.OriginalDay, task.OriginalMonth, task.OriginalYear)
	}
}

// TestUpdateTaskSwitchesLunarToGregorianClearsOriginals — same contract as
// LifeEvent's reverse-transition test; protects against persistent lunar
// metadata after the user changes their mind.
func TestUpdateTaskSwitchesLunarToGregorianClearsOriginals(t *testing.T) {
	svc, contactID, vaultID, userID := setupTaskTest(t)
	due := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day, month, year := 15, 8, 2025
	task, err := svc.Create(contactID, vaultID, userID, dto.CreateTaskRequest{
		Label: "Lunar task", DueAt: &due,
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newDue := time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC)
	updated, err := svc.Update(task.ID, contactID, vaultID, dto.UpdateTaskRequest{
		Label:        "Lunar task",
		DueAt:        &newDue,
		CalendarType: "gregorian",
	}, userID)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.CalendarType != "gregorian" {
		t.Errorf("Expected CalendarType 'gregorian', got %q", updated.CalendarType)
	}
	if updated.OriginalDay != nil || updated.OriginalMonth != nil || updated.OriginalYear != nil {
		t.Errorf("Expected Original* cleared, got %v/%v/%v",
			updated.OriginalDay, updated.OriginalMonth, updated.OriginalYear)
	}
	if updated.DueAt == nil || updated.DueAt.Month() != time.December || updated.DueAt.Day() != 25 {
		t.Errorf("Expected DueAt at 2025-12-25, got %v", updated.DueAt)
	}
}
