package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
)

// TestVaultTaskCreateProjectsLunarDueDate pins the lunar-create contract for
// the kanban path: the TaskEditModal sends a CreateVaultTaskRequest with
// CalendarType="lunar" + Original*; the server must project DueAt to gregorian
// (so kanban ordering is consistent) while persisting the lunar anchor so the
// reminder scheduler can reuse it for recurrences. Symmetric to
// TestCreateTaskWithLunarDueDate in task_lunar_test.go.
func TestVaultTaskCreateProjectsLunarDueDate(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	placeholder := time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC)
	day, month, year := 15, 8, 2025
	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
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

// TestVaultTaskCreateWithoutDueDateClearsCalendar — kanban siblings of
// TestCreateTaskWithoutDueDateIgnoresCalendarType. A standalone "someday/maybe"
// kanban card sent with lunar metadata must still land as a clean gregorian
// row so a later edit that re-adds a due date doesn't pick up the stale anchor.
func TestVaultTaskCreateWithoutDueDateClearsCalendar(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	day, month, year := 1, 1, 2025
	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
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

// TestVaultTaskUpdateSwitchesLunarToGregorianClearsOriginals — Update path must
// also clear stale Original* when the user switches back to gregorian, otherwise
// the lunar anchor stays attached and silently resurfaces on the next edit
// that picks up "leave calendar_type unchanged" semantics.
func TestVaultTaskUpdateSwitchesLunarToGregorianClearsOriginals(t *testing.T) {
	svc, _, vaultID, _, userID := setupVaultTaskTest(t)

	due := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day, month, year := 15, 8, 2025
	task, err := svc.Create(vaultID, userID, dto.CreateVaultTaskRequest{
		Label:         "Lunar task",
		DueAt:         &due,
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newDue := time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC)
	updated, err := svc.Update(task.ID, vaultID, dto.UpdateVaultTaskRequest{
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
