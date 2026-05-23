package services

import (
	"testing"
	"time"

	_ "github.com/naiba/bonds/internal/calendar"
)

func TestApplyTimeCalendarFieldsGregorian(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	tm := time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC)

	applyTimeCalendarFields(&calType, &origDay, &origMonth, &origYear, &tm,
		"gregorian", nil, nil, nil)

	if calType != "gregorian" {
		t.Errorf("Expected calType 'gregorian', got %q", calType)
	}
	if origDay != nil || origMonth != nil || origYear != nil {
		t.Errorf("Expected Original* nil for gregorian, got %v/%v/%v", origDay, origMonth, origYear)
	}
	if !tm.Equal(time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("time should be unchanged, got %v", tm)
	}
}

func TestApplyTimeCalendarFieldsEmpty(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	tm := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	applyTimeCalendarFields(&calType, &origDay, &origMonth, &origYear, &tm, "", nil, nil, nil)
	if calType != "gregorian" {
		t.Errorf("Expected calType 'gregorian' for empty request, got %q", calType)
	}
	if origDay != nil || origMonth != nil || origYear != nil {
		t.Errorf("Expected Original* nil for empty request, got %v/%v/%v", origDay, origMonth, origYear)
	}
}

func TestApplyTimeCalendarFieldsLunarProjectsAndPreservesClock(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	// Start from an arbitrary placeholder time; clock fields must survive
	// the lunar→gregorian projection so a 09:30 reminder stays 09:30.
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tm := time.Date(2025, 1, 1, 9, 30, 45, 123456789, loc)

	reqDay, reqMonth, reqYear := 15, 1, 2025
	applyTimeCalendarFields(&calType, &origDay, &origMonth, &origYear, &tm,
		"lunar", &reqDay, &reqMonth, &reqYear)

	if calType != "lunar" {
		t.Errorf("Expected calType 'lunar', got %q", calType)
	}
	if origDay == nil || *origDay != 15 {
		t.Errorf("Expected origDay=15, got %v", origDay)
	}
	if origMonth == nil || *origMonth != 1 {
		t.Errorf("Expected origMonth=1, got %v", origMonth)
	}
	if origYear == nil || *origYear != 2025 {
		t.Errorf("Expected origYear=2025, got %v", origYear)
	}
	// 2025 lunar month 1 day 15 = 2025-02-12 gregorian (lantern festival).
	if tm.Year() != 2025 || tm.Month() != time.February {
		t.Errorf("Expected projected date in Feb 2025, got %v", tm)
	}
	if tm.Day() < 1 || tm.Day() > 28 {
		t.Errorf("Expected projected day in Feb range, got %d", tm.Day())
	}
	if tm.Hour() != 9 || tm.Minute() != 30 || tm.Second() != 45 || tm.Nanosecond() != 123456789 {
		t.Errorf("Clock fields should survive projection, got %v", tm)
	}
	if tm.Location().String() != loc.String() {
		t.Errorf("Location should survive projection, got %v want %v", tm.Location(), loc)
	}
}

func TestApplyTimeCalendarFieldsUnsupportedFallsBack(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	tm := time.Date(2025, 3, 1, 8, 0, 0, 0, time.UTC)
	original := tm

	reqDay, reqMonth, reqYear := 10, 5, 2568
	applyTimeCalendarFields(&calType, &origDay, &origMonth, &origYear, &tm,
		"buddhist", &reqDay, &reqMonth, &reqYear)

	if calType != "gregorian" {
		t.Errorf("Expected fallback calType 'gregorian', got %q", calType)
	}
	if origDay != nil || origMonth != nil || origYear != nil {
		t.Errorf("Expected Original* cleared on unsupported, got %v/%v/%v", origDay, origMonth, origYear)
	}
	if !tm.Equal(original) {
		t.Errorf("time should be unchanged on fallback, got %v want %v", tm, original)
	}
}

func TestApplyTimeCalendarFieldsLunarWithoutOriginalDayLeavesTimeAlone(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	tm := time.Date(2025, 7, 4, 14, 0, 0, 0, time.UTC)
	original := tm

	applyTimeCalendarFields(&calType, &origDay, &origMonth, &origYear, &tm, "lunar", nil, nil, nil)
	if calType != "lunar" {
		t.Errorf("Expected calType 'lunar', got %q", calType)
	}
	if !tm.Equal(original) {
		t.Errorf("time should be unchanged when Original* missing, got %v want %v", tm, original)
	}
}
