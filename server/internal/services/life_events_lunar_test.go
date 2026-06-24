package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
)

// TestAddLifeEventWithLunarCalendar pins the lunar-create contract for life
// events: the request carries original_day/month/year + calendar_type='lunar',
// the server must persist all three Original* fields verbatim AND overwrite
// HappenedAt with the lunar→gregorian projection so timeline ordering stays
// consistent regardless of what gregorian guess the client sent.
func TestAddLifeEventWithLunarCalendar(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)

	te, err := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(),
		Label:     "Lunar timeline",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEvent failed: %v", err)
	}

	// Client sends a placeholder gregorian date; server should overwrite it.
	placeholder := time.Date(2025, 1, 1, 9, 30, 0, 0, time.UTC)
	day, month, year := 15, 1, 2025
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      placeholder,
		Summary:         "Lantern festival memory",
		CalendarType:    "lunar",
		OriginalDay:     &day,
		OriginalMonth:   &month,
		OriginalYear:    &year,
	})
	if err != nil {
		t.Fatalf("AddLifeEvent failed: %v", err)
	}
	if le.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got %q", le.CalendarType)
	}
	if le.OriginalDay == nil || *le.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay=15, got %v", le.OriginalDay)
	}
	if le.OriginalMonth == nil || *le.OriginalMonth != 1 {
		t.Errorf("Expected OriginalMonth=1, got %v", le.OriginalMonth)
	}
	if le.OriginalYear == nil || *le.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear=2025, got %v", le.OriginalYear)
	}
	// 2025 lunar 1/15 = 2025-02-12 gregorian.
	if le.HappenedAt.Month() != time.February || le.HappenedAt.Year() != 2025 {
		t.Errorf("Expected HappenedAt projected into Feb 2025, got %v", le.HappenedAt)
	}
	if le.HappenedAt.Hour() != 9 || le.HappenedAt.Minute() != 30 {
		t.Errorf("Expected clock fields preserved (09:30), got %v", le.HappenedAt)
	}
}

// TestUpdateLifeEventClearsLunarWhenSwitchedBackToGregorian guards the reverse
// transition: an edit that resends an empty/gregorian calendar_type must wipe
// the lunar Original* metadata so a previously-lunar event doesn't keep
// projecting on every subsequent edit.
func TestUpdateLifeEventClearsLunarWhenSwitchedBackToGregorian(t *testing.T) {
	svc, contactID, vaultID := setupLifeEventTest(t)
	te, _ := svc.CreateTimelineEvent(contactID, vaultID, dto.CreateTimelineEventRequest{
		StartedAt: time.Now(), Label: "T",
	})
	day, month, year := 15, 1, 2025
	le, err := svc.AddLifeEvent(contactID, te.ID, vaultID, dto.CreateLifeEventRequest{
		LifeEventTypeID: 1,
		HappenedAt:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CalendarType:    "lunar",
		OriginalDay:     &day, OriginalMonth: &month, OriginalYear: &year,
	})
	if err != nil {
		t.Fatalf("AddLifeEvent: %v", err)
	}

	updated, err := svc.UpdateLifeEvent(contactID, te.ID, le.ID, vaultID, dto.UpdateLifeEventRequest{
		HappenedAt:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		CalendarType: "gregorian",
	})
	if err != nil {
		t.Fatalf("UpdateLifeEvent: %v", err)
	}
	if updated.CalendarType != "gregorian" {
		t.Errorf("Expected CalendarType cleared to 'gregorian', got %q", updated.CalendarType)
	}
	if updated.OriginalDay != nil || updated.OriginalMonth != nil || updated.OriginalYear != nil {
		t.Errorf("Expected Original* cleared, got %v/%v/%v", updated.OriginalDay, updated.OriginalMonth, updated.OriginalYear)
	}
	if updated.HappenedAt.Month() != time.June || updated.HappenedAt.Day() != 15 {
		t.Errorf("Expected HappenedAt to be the supplied gregorian date, got %v", updated.HappenedAt)
	}
}
