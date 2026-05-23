package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
)

// TestCreatePostWithLunarWrittenAt pins the lunar-create contract for journal
// posts: WrittenAt anchors timeline sort, so the server must overwrite the
// client's projection with its own lunar→gregorian computation. Otherwise two
// clients on different calendar libraries could disagree on the gregorian
// equivalent and the post would jump around in the journal feed.
func TestCreatePostWithLunarWrittenAt(t *testing.T) {
	svc, journalID := setupPostTest(t)

	placeholder := time.Date(2025, 1, 1, 20, 15, 0, 0, time.UTC)
	day, month, year := 15, 1, 2025
	post, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:         "Lantern festival",
		Published:     true,
		WrittenAt:     placeholder,
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got %q", post.CalendarType)
	}
	if post.OriginalDay == nil || *post.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay=15, got %v", post.OriginalDay)
	}
	if post.OriginalMonth == nil || *post.OriginalMonth != 1 {
		t.Errorf("Expected OriginalMonth=1, got %v", post.OriginalMonth)
	}
	if post.OriginalYear == nil || *post.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear=2025, got %v", post.OriginalYear)
	}
	if post.WrittenAt.Year() != 2025 || post.WrittenAt.Month() != time.February {
		t.Errorf("Expected WrittenAt projected into Feb 2025, got %v", post.WrittenAt)
	}
	if post.WrittenAt.Hour() != 20 || post.WrittenAt.Minute() != 15 {
		t.Errorf("Expected clock fields preserved (20:15), got %v", post.WrittenAt)
	}
}

// TestUpdatePostSwitchesLunarToGregorianClearsOriginals — reverse transition.
func TestUpdatePostSwitchesLunarToGregorianClearsOriginals(t *testing.T) {
	svc, journalID := setupPostTest(t)

	day, month, year := 15, 1, 2025
	post, err := svc.Create(journalID, dto.CreatePostRequest{
		Title:         "Lunar entry",
		WrittenAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CalendarType:  "lunar",
		OriginalDay:   &day,
		OriginalMonth: &month,
		OriginalYear:  &year,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(post.ID, journalID, dto.UpdatePostRequest{
		Title:        "Lunar entry",
		WrittenAt:    time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		CalendarType: "gregorian",
	})
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
	if updated.WrittenAt.Month() != time.June || updated.WrittenAt.Day() != 15 {
		t.Errorf("Expected WrittenAt to be the supplied gregorian date, got %v", updated.WrittenAt)
	}
}
