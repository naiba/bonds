package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPreferenceTest(t *testing.T) (*PreferenceService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "preferences-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewPreferenceService(db), resp.User.ID
}

func TestPreferenceGet(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	prefs, err := svc.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if prefs.Locale == "" {
		t.Error("Expected locale to be non-empty")
	}
	if prefs.DateFormat == "" {
		t.Error("Expected date_format to be non-empty")
	}
	if prefs.NameOrder == "" {
		t.Error("Expected name_order to be non-empty")
	}
}

func TestPreferenceUpdateNameOrder(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	err := svc.UpdateNameOrder(userID, dto.UpdateNameOrderRequest{NameOrder: "%last_name% %first_name%"})
	if err != nil {
		t.Fatalf("UpdateNameOrder failed: %v", err)
	}

	prefs, err := svc.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if prefs.NameOrder != "%last_name% %first_name%" {
		t.Errorf("Expected name_order '%%last_name%% %%first_name%%', got '%s'", prefs.NameOrder)
	}
}

func TestPreferenceUpdateDateFormat(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	err := svc.UpdateDateFormat(userID, dto.UpdateDateFormatRequest{DateFormat: "DD/MM/YYYY"})
	if err != nil {
		t.Fatalf("UpdateDateFormat failed: %v", err)
	}

	prefs, err := svc.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if prefs.DateFormat != "DD/MM/YYYY" {
		t.Errorf("Expected date_format 'DD/MM/YYYY', got '%s'", prefs.DateFormat)
	}
}

func TestPreferenceUpdateTimezone(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	err := svc.UpdateTimezone(userID, dto.UpdateTimezoneRequest{Timezone: "America/New_York"})
	if err != nil {
		t.Fatalf("UpdateTimezone failed: %v", err)
	}

	prefs, err := svc.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if prefs.Timezone != "America/New_York" {
		t.Errorf("Expected timezone 'America/New_York', got '%s'", prefs.Timezone)
	}
}

func TestPreferenceUpdateLocale(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	err := svc.UpdateLocale(userID, dto.UpdateLocaleRequest{Locale: "fr"})
	if err != nil {
		t.Fatalf("UpdateLocale failed: %v", err)
	}

	prefs, err := svc.Get(userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if prefs.Locale != "fr" {
		t.Errorf("Expected locale 'fr', got '%s'", prefs.Locale)
	}
}

func TestPreferenceUpdateAll(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	prefs, err := svc.UpdateAll(userID, dto.UpdatePreferencesRequest{
		NameOrder:  "last_first",
		DateFormat: "DD/MM/YYYY",
		Timezone:   "Asia/Shanghai",
		Locale:     "zh",
	})
	if err != nil {
		t.Fatalf("UpdateAll failed: %v", err)
	}
	if prefs.NameOrder != "last_first" {
		t.Errorf("Expected name_order 'last_first', got '%s'", prefs.NameOrder)
	}
	if prefs.DateFormat != "DD/MM/YYYY" {
		t.Errorf("Expected date_format 'DD/MM/YYYY', got '%s'", prefs.DateFormat)
	}
	if prefs.Timezone != "Asia/Shanghai" {
		t.Errorf("Expected timezone 'Asia/Shanghai', got '%s'", prefs.Timezone)
	}
	if prefs.Locale != "zh" {
		t.Errorf("Expected locale 'zh', got '%s'", prefs.Locale)
	}
}

func TestPreferenceUpdateAllPartial(t *testing.T) {
	svc, userID := setupPreferenceTest(t)

	if err := svc.UpdateLocale(userID, dto.UpdateLocaleRequest{Locale: "de"}); err != nil {
		t.Fatalf("UpdateLocale failed: %v", err)
	}

	prefs, err := svc.UpdateAll(userID, dto.UpdatePreferencesRequest{
		Timezone: "Europe/Berlin",
	})
	if err != nil {
		t.Fatalf("UpdateAll partial failed: %v", err)
	}
	if prefs.Timezone != "Europe/Berlin" {
		t.Errorf("Expected timezone 'Europe/Berlin', got '%s'", prefs.Timezone)
	}
	if prefs.Locale != "de" {
		t.Errorf("Expected locale to remain 'de', got '%s'", prefs.Locale)
	}
}
