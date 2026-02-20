package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupReportTest(t *testing.T) (*ReportService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-test@example.com",
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

	return NewReportService(db), vault.ID, contact.ID
}

func TestAddressReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-addr@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	country := "US"
	city := "New York"
	province := "NY"
	addr := &models.Address{
		VaultID:  vault.ID,
		Country:  &country,
		City:     &city,
		Province: &province,
	}
	if err := db.Create(addr).Error; err != nil {
		t.Fatalf("Create address failed: %v", err)
	}

	svc := NewReportService(db)
	report, err := svc.AddressReport(vault.ID)
	if err != nil {
		t.Fatalf("AddressReport failed: %v", err)
	}
	if len(report) != 1 {
		t.Errorf("Expected 1 address report item, got %d", len(report))
	}
	if len(report) > 0 && report[0].Country != "US" {
		t.Errorf("Expected country 'US', got '%s'", report[0].Country)
	}
}

func TestAddressReportEmpty(t *testing.T) {
	svc, vaultID, _ := setupReportTest(t)

	report, err := svc.AddressReport(vaultID)
	if err != nil {
		t.Fatalf("AddressReport failed: %v", err)
	}
	if len(report) != 0 {
		t.Errorf("Expected 0 address report items, got %d", len(report))
	}
}

func TestImportantDatesReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-dates@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	day := 25
	month := 12
	importantDate := &models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Christmas",
		Day:       &day,
		Month:     &month,
	}
	if err := db.Create(importantDate).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}

	svc := NewReportService(db)
	report, err := svc.ImportantDatesReport(vault.ID)
	if err != nil {
		t.Fatalf("ImportantDatesReport failed: %v", err)
	}
	if len(report) != 1 {
		t.Errorf("Expected 1 important date report item, got %d", len(report))
	}
	if len(report) > 0 && report[0].Label != "Christmas" {
		t.Errorf("Expected label 'Christmas', got '%s'", report[0].Label)
	}
}

func TestImportantDatesReportLunar(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-lunar@example.com",
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
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Lunar"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	origDay := 15
	origMonth := 1
	origYear := 2025
	gregDay := 12
	gregMonth := 2
	gregYear := 2025
	importantDate := &models.ContactImportantDate{
		ContactID:     contact.ID,
		Label:         "Lunar New Year",
		Day:           &gregDay,
		Month:         &gregMonth,
		Year:          &gregYear,
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	}
	if err := db.Create(importantDate).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}

	svc := NewReportService(db)
	report, err := svc.ImportantDatesReport(vault.ID)
	if err != nil {
		t.Fatalf("ImportantDatesReport failed: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("Expected 1 important date report item, got %d", len(report))
	}
	item := report[0]
	if item.Label != "Lunar New Year" {
		t.Errorf("Expected label 'Lunar New Year', got '%s'", item.Label)
	}
	if item.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got '%s'", item.CalendarType)
	}
	if item.OriginalDay == nil || *item.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay 15, got %v", item.OriginalDay)
	}
	if item.OriginalMonth == nil || *item.OriginalMonth != 1 {
		t.Errorf("Expected OriginalMonth 1, got %v", item.OriginalMonth)
	}
	if item.OriginalYear == nil || *item.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear 2025, got %v", item.OriginalYear)
	}
	if item.Day == nil || *item.Day != 12 {
		t.Errorf("Expected gregorian Day 12, got %v", item.Day)
	}
	if item.Month == nil || *item.Month != 2 {
		t.Errorf("Expected gregorian Month 2, got %v", item.Month)
	}
}

func TestReportOverview(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-overview@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Overview Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact1, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	svc := NewReportService(db)

	overview, err := svc.Overview(vault.ID)
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if overview.TotalContacts != 2 {
		t.Errorf("Expected 2 contacts, got %d", overview.TotalContacts)
	}
	if overview.TotalAddresses != 0 {
		t.Errorf("Expected 0 addresses, got %d", overview.TotalAddresses)
	}
	if overview.TotalImportantDates != 0 {
		t.Errorf("Expected 0 important dates, got %d", overview.TotalImportantDates)
	}
	if overview.TotalMoodEntries != 0 {
		t.Errorf("Expected 0 mood entries, got %d", overview.TotalMoodEntries)
	}

	country := "US"
	city := "New York"
	addr := &models.Address{VaultID: vault.ID, Country: &country, City: &city}
	if err := db.Create(addr).Error; err != nil {
		t.Fatalf("Create address failed: %v", err)
	}

	day := 25
	month := 12
	importantDate := &models.ContactImportantDate{
		ContactID: contact1.ID,
		Label:     "Birthday",
		Day:       &day,
		Month:     &month,
	}
	if err := db.Create(importantDate).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}
	importantDate2 := &models.ContactImportantDate{
		ContactID: contact2.ID,
		Label:     "Anniversary",
		Day:       &day,
		Month:     &month,
	}
	if err := db.Create(importantDate2).Error; err != nil {
		t.Fatalf("Create important date 2 failed: %v", err)
	}

	var params []models.MoodTrackingParameter
	if err := db.Where("vault_id = ?", vault.ID).Find(&params).Error; err != nil {
		t.Fatalf("Find mood params failed: %v", err)
	}
	if len(params) > 0 {
		event := &models.MoodTrackingEvent{
			ContactID:               contact1.ID,
			MoodTrackingParameterID: params[0].ID,
			RatedAt:                 importantDate.CreatedAt,
		}
		if err := db.Create(event).Error; err != nil {
			t.Fatalf("Create mood event failed: %v", err)
		}
	}

	overview, err = svc.Overview(vault.ID)
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if overview.TotalContacts != 2 {
		t.Errorf("Expected 2 contacts, got %d", overview.TotalContacts)
	}
	if overview.TotalAddresses != 1 {
		t.Errorf("Expected 1 address, got %d", overview.TotalAddresses)
	}
	if overview.TotalImportantDates != 2 {
		t.Errorf("Expected 2 important dates, got %d", overview.TotalImportantDates)
	}
	if len(params) > 0 && overview.TotalMoodEntries != 1 {
		t.Errorf("Expected 1 mood entry, got %d", overview.TotalMoodEntries)
	}
}

func TestReportOverviewEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-overview-empty@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewReportService(db)
	overview, err := svc.Overview(vault.ID)
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	if overview.TotalContacts != 0 {
		t.Errorf("Expected 0 contacts, got %d", overview.TotalContacts)
	}
	if overview.TotalAddresses != 0 {
		t.Errorf("Expected 0 addresses, got %d", overview.TotalAddresses)
	}
	if overview.TotalImportantDates != 0 {
		t.Errorf("Expected 0 important dates, got %d", overview.TotalImportantDates)
	}
	if overview.TotalMoodEntries != 0 {
		t.Errorf("Expected 0 mood entries, got %d", overview.TotalMoodEntries)
	}
}

func TestMoodReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reports-mood@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	label := "Happy"
	param := &models.MoodTrackingParameter{
		VaultID:  vault.ID,
		Label:    &label,
		HexColor: "#00FF00",
	}
	if err := db.Create(param).Error; err != nil {
		t.Fatalf("Create mood parameter failed: %v", err)
	}

	svc := NewReportService(db)
	report, err := svc.MoodReport(vault.ID)
	if err != nil {
		t.Fatalf("MoodReport failed: %v", err)
	}
	if len(report) != 6 {
		t.Errorf("Expected 6 mood report items (5 seeded + 1 created), got %d", len(report))
	}
	found := false
	for _, r := range report {
		if r.ParameterLabel == "Happy" && r.HexColor == "#00FF00" {
			found = true
			if r.Count != 0 {
				t.Errorf("Expected count 0 for 'Happy', got %d", r.Count)
			}
		}
	}
	if !found {
		t.Error("Expected to find 'Happy' mood parameter in report")
	}
}
