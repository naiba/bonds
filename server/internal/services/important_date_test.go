package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupImportantDateTest(t *testing.T) (*ImportantDateService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "important-date-test@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewImportantDateService(db), contact.ID
}

func TestCreateImportantDate(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	day := 15
	month := 6
	year := 1990
	date, err := svc.Create(contactID, dto.CreateImportantDateRequest{
		Label: "Birthday",
		Day:   &day,
		Month: &month,
		Year:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", date.Label)
	}
	if date.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, date.ContactID)
	}
	if date.Day == nil || *date.Day != 15 {
		t.Errorf("Expected day 15, got %v", date.Day)
	}
	if date.Month == nil || *date.Month != 6 {
		t.Errorf("Expected month 6, got %v", date.Month)
	}
	if date.Year == nil || *date.Year != 1990 {
		t.Errorf("Expected year 1990, got %v", date.Year)
	}
	if date.ID == 0 {
		t.Error("Expected important date ID to be non-zero")
	}
}

func TestListImportantDates(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	_, err := svc.Create(contactID, dto.CreateImportantDateRequest{Label: "Birthday"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, dto.CreateImportantDateRequest{Label: "Anniversary"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dates, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(dates) != 2 {
		t.Errorf("Expected 2 important dates, got %d", len(dates))
	}
}

func TestUpdateImportantDate(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	created, err := svc.Create(contactID, dto.CreateImportantDateRequest{Label: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	day := 25
	month := 12
	updated, err := svc.Update(created.ID, contactID, dto.UpdateImportantDateRequest{
		Label: "Updated",
		Day:   &day,
		Month: &month,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Updated" {
		t.Errorf("Expected label 'Updated', got '%s'", updated.Label)
	}
	if updated.Day == nil || *updated.Day != 25 {
		t.Errorf("Expected day 25, got %v", updated.Day)
	}
	if updated.Month == nil || *updated.Month != 12 {
		t.Errorf("Expected month 12, got %v", updated.Month)
	}
}

func TestDeleteImportantDate(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	created, err := svc.Create(contactID, dto.CreateImportantDateRequest{Label: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	dates, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("Expected 0 important dates after delete, got %d", len(dates))
	}
}

func TestDeleteImportantDateNotFound(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	err := svc.Delete(9999, contactID)
	if err != ErrImportantDateNotFound {
		t.Errorf("Expected ErrImportantDateNotFound, got %v", err)
	}
}

func TestCreateImportantDateLunar(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	origDay := 15
	origMonth := 1
	origYear := 2025
	date, err := svc.Create(contactID, dto.CreateImportantDateRequest{
		Label:         "Lunar New Year",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.Label != "Lunar New Year" {
		t.Errorf("Expected label 'Lunar New Year', got '%s'", date.Label)
	}
	if date.CalendarType != "lunar" {
		t.Errorf("Expected calendar_type 'lunar', got '%s'", date.CalendarType)
	}
	if date.OriginalDay == nil || *date.OriginalDay != 15 {
		t.Errorf("Expected original_day 15, got %v", date.OriginalDay)
	}
	if date.OriginalMonth == nil || *date.OriginalMonth != 1 {
		t.Errorf("Expected original_month 1, got %v", date.OriginalMonth)
	}
	if date.OriginalYear == nil || *date.OriginalYear != 2025 {
		t.Errorf("Expected original_year 2025, got %v", date.OriginalYear)
	}
	if date.Day == nil || date.Month == nil || date.Year == nil {
		t.Fatal("Expected gregorian day/month/year to be set after conversion")
	}
	if *date.Year != 2025 {
		t.Errorf("Expected gregorian year 2025, got %d", *date.Year)
	}
	if *date.Month != 2 {
		t.Errorf("Expected gregorian month 2 (Feb), got %d", *date.Month)
	}
}

func TestCreateImportantDateGregorianDefault(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	day := 25
	month := 12
	year := 2025
	date, err := svc.Create(contactID, dto.CreateImportantDateRequest{
		Label: "Christmas",
		Day:   &day,
		Month: &month,
		Year:  &year,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if date.CalendarType != "gregorian" {
		t.Errorf("Expected default calendar_type 'gregorian', got '%s'", date.CalendarType)
	}
	if date.OriginalDay != nil {
		t.Errorf("Expected nil original_day for gregorian, got %v", date.OriginalDay)
	}
}

func TestUpdateImportantDateLunar(t *testing.T) {
	svc, contactID := setupImportantDateTest(t)

	created, err := svc.Create(contactID, dto.CreateImportantDateRequest{Label: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	origDay := 8
	origMonth := 8
	origYear := 2025
	updated, err := svc.Update(created.ID, contactID, dto.UpdateImportantDateRequest{
		Label:         "Mid-Autumn",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.CalendarType != "lunar" {
		t.Errorf("Expected calendar_type 'lunar', got '%s'", updated.CalendarType)
	}
	if updated.OriginalDay == nil || *updated.OriginalDay != 8 {
		t.Errorf("Expected original_day 8, got %v", updated.OriginalDay)
	}
	if updated.Day == nil || updated.Month == nil {
		t.Fatal("Expected gregorian day/month to be set")
	}
}
