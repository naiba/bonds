package services

import (
	"testing"

	_ "github.com/naiba/bonds/internal/calendar"
)

func TestApplyCalendarFieldsGregorian(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	day := 15
	month := 6
	year := 2025
	dayP, monthP, yearP := &day, &month, &year

	applyCalendarFields(&calType, &origDay, &origMonth, &origYear, &dayP, &monthP, &yearP,
		"gregorian", nil, nil, nil)

	if calType != "gregorian" {
		t.Errorf("Expected calType 'gregorian', got '%s'", calType)
	}
	if origDay != nil {
		t.Errorf("Expected origDay nil, got %v", origDay)
	}
	if origMonth != nil {
		t.Errorf("Expected origMonth nil, got %v", origMonth)
	}
	if origYear != nil {
		t.Errorf("Expected origYear nil, got %v", origYear)
	}
	if *dayP != 15 || *monthP != 6 || *yearP != 2025 {
		t.Errorf("Gregorian day/month/year should be unchanged, got %d/%d/%d", *dayP, *monthP, *yearP)
	}

	calType = ""
	origDay = nil
	origMonth = nil
	origYear = nil
	day2 := 1
	month2 := 1
	year2 := 2025
	dayP2, monthP2, yearP2 := &day2, &month2, &year2

	applyCalendarFields(&calType, &origDay, &origMonth, &origYear, &dayP2, &monthP2, &yearP2,
		"", nil, nil, nil)

	if calType != "gregorian" {
		t.Errorf("Expected calType 'gregorian' for empty reqCalType, got '%s'", calType)
	}
	if origDay != nil || origMonth != nil || origYear != nil {
		t.Errorf("Expected all original fields nil for empty reqCalType")
	}
}

func TestApplyCalendarFieldsLunar(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	day := 1
	month := 1
	year := 2025
	dayP, monthP, yearP := &day, &month, &year

	reqOrigDay := 15
	reqOrigMonth := 1
	reqOrigYear := 2025
	applyCalendarFields(&calType, &origDay, &origMonth, &origYear, &dayP, &monthP, &yearP,
		"lunar", &reqOrigDay, &reqOrigMonth, &reqOrigYear)

	if calType != "lunar" {
		t.Errorf("Expected calType 'lunar', got '%s'", calType)
	}
	if origDay == nil || *origDay != 15 {
		t.Errorf("Expected origDay 15, got %v", origDay)
	}
	if origMonth == nil || *origMonth != 1 {
		t.Errorf("Expected origMonth 1, got %v", origMonth)
	}
	if origYear == nil || *origYear != 2025 {
		t.Errorf("Expected origYear 2025, got %v", origYear)
	}
	if dayP == nil || monthP == nil || yearP == nil {
		t.Fatal("Expected converted gregorian pointers to be non-nil")
	}
	if *monthP != 2 {
		t.Errorf("Expected converted gregorian month 2 (Feb), got %d", *monthP)
	}
	if *yearP != 2025 {
		t.Errorf("Expected converted gregorian year 2025, got %d", *yearP)
	}
	if *dayP < 1 || *dayP > 28 {
		t.Errorf("Expected converted gregorian day in valid Feb range, got %d", *dayP)
	}
}

func TestApplyCalendarFieldsUnsupported(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	day := 10
	month := 5
	year := 2025
	dayP, monthP, yearP := &day, &month, &year

	reqOrigDay := 10
	reqOrigMonth := 5
	reqOrigYear := 2568
	applyCalendarFields(&calType, &origDay, &origMonth, &origYear, &dayP, &monthP, &yearP,
		"buddhist", &reqOrigDay, &reqOrigMonth, &reqOrigYear)

	if calType != "gregorian" {
		t.Errorf("Expected calType 'gregorian' for unsupported type, got '%s'", calType)
	}
	if *dayP != 10 || *monthP != 5 || *yearP != 2025 {
		t.Errorf("Expected day/month/year unchanged for unsupported type, got %d/%d/%d", *dayP, *monthP, *yearP)
	}
}

func TestApplyCalendarFieldsNilOriginal(t *testing.T) {
	var calType string
	var origDay, origMonth, origYear *int
	day := 20
	month := 3
	year := 2025
	dayP, monthP, yearP := &day, &month, &year

	applyCalendarFields(&calType, &origDay, &origMonth, &origYear, &dayP, &monthP, &yearP,
		"lunar", nil, nil, nil)

	if calType != "lunar" {
		t.Errorf("Expected calType 'lunar', got '%s'", calType)
	}
	if origDay != nil {
		t.Errorf("Expected origDay nil when reqOrigDay is nil, got %v", origDay)
	}
	if origMonth != nil {
		t.Errorf("Expected origMonth nil when reqOrigMonth is nil, got %v", origMonth)
	}
	if *dayP != 20 || *monthP != 3 || *yearP != 2025 {
		t.Errorf("Expected day/month/year unchanged when originals are nil, got %d/%d/%d", *dayP, *monthP, *yearP)
	}
}
