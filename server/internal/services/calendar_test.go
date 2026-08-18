package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupCalendarTest(t *testing.T) (*CalendarService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "calendar-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewCalendarService(db), vault.ID, resp.User.ID, contact.ID
}

func TestCalendarEmpty(t *testing.T) {
	svc, vaultID, userID, _ := setupCalendarTest(t)

	cal, err := svc.GetCalendar(vaultID, userID, 1, 2025, "en")
	if err != nil {
		t.Fatalf("GetCalendar failed: %v", err)
	}
	if len(cal.ImportantDates) != 0 {
		t.Errorf("Expected 0 important dates, got %d", len(cal.ImportantDates))
	}
	if len(cal.Reminders) != 0 {
		t.Errorf("Expected 0 reminders, got %d", len(cal.Reminders))
	}
}

func TestCalendarWithDates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "calendar-dates@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	day := 15
	month := 3
	year := 2025
	importantDate := &models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Birthday",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}
	if err := db.Create(importantDate).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}

	reminder := &models.ContactReminder{
		ContactID: contact.ID,
		Label:     "Call Jane",
		Day:       &day,
		Month:     &month,
		Type:      "one_time",
	}
	if err := db.Create(reminder).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	svc := NewCalendarService(db)
	cal, err := svc.GetCalendar(vault.ID, resp.User.ID, 3, 2025, "en")
	if err != nil {
		t.Fatalf("GetCalendar failed: %v", err)
	}
	if len(cal.ImportantDates) != 1 {
		t.Errorf("Expected 1 important date, got %d", len(cal.ImportantDates))
	}
	if len(cal.ImportantDates) > 0 && cal.ImportantDates[0].Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", cal.ImportantDates[0].Label)
	}
	if len(cal.ImportantDates) > 0 && cal.ImportantDates[0].ContactName != "Jane" {
		t.Errorf("Expected contact_name 'Jane', got '%s'", cal.ImportantDates[0].ContactName)
	}
	if len(cal.Reminders) != 1 {
		t.Errorf("Expected 1 reminder, got %d", len(cal.Reminders))
	}
	if len(cal.Reminders) > 0 && cal.Reminders[0].Label != "Call Jane" {
		t.Errorf("Expected label 'Call Jane', got '%s'", cal.Reminders[0].Label)
	}
	if len(cal.Reminders) > 0 && cal.Reminders[0].ContactName != "Jane" {
		t.Errorf("Expected contact_name 'Jane', got '%s'", cal.Reminders[0].ContactName)
	}
}

func TestCalendarWithLunarDates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "calendar-lunar@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Lunar"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Lunar 正月十五 2025 → Gregorian Feb 12, 2025
	calType := "lunar"
	origDay := 15
	origMonth := 1
	origYear := 2025
	gregDay := 12
	gregMonth := 2
	gregYear := 2025

	importantDate := &models.ContactImportantDate{
		ContactID:     contact.ID,
		Label:         "Lunar Birthday",
		Day:           &gregDay,
		Month:         &gregMonth,
		Year:          &gregYear,
		CalendarType:  calType,
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
	}
	if err := db.Create(importantDate).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}

	reminder := &models.ContactReminder{
		ContactID:     contact.ID,
		Label:         "Lunar Reminder",
		Day:           &gregDay,
		Month:         &gregMonth,
		Year:          &gregYear,
		CalendarType:  calType,
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		OriginalYear:  &origYear,
		Type:          "one_time",
	}
	if err := db.Create(reminder).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	svc := NewCalendarService(db)
	cal, err := svc.GetCalendar(vault.ID, resp.User.ID, 2, 2025, "en")
	if err != nil {
		t.Fatalf("GetCalendar failed: %v", err)
	}

	// Assert important date lunar fields
	if len(cal.ImportantDates) != 1 {
		t.Fatalf("Expected 1 important date, got %d", len(cal.ImportantDates))
	}
	d := cal.ImportantDates[0]
	if d.ContactName != "Lunar" {
		t.Errorf("Expected contact_name 'Lunar', got '%s'", d.ContactName)
	}
	if d.Label != "Lunar Birthday" {
		t.Errorf("Expected label 'Lunar Birthday', got '%s'", d.Label)
	}
	if d.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got '%s'", d.CalendarType)
	}
	if d.OriginalDay == nil || *d.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay 15, got %v", d.OriginalDay)
	}
	if d.OriginalMonth == nil || *d.OriginalMonth != 1 {
		t.Errorf("Expected OriginalMonth 1, got %v", d.OriginalMonth)
	}
	if d.OriginalYear == nil || *d.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear 2025, got %v", d.OriginalYear)
	}
	if d.Day == nil || *d.Day != 12 {
		t.Errorf("Expected gregorian Day 12, got %v", d.Day)
	}
	if d.Month == nil || *d.Month != 2 {
		t.Errorf("Expected gregorian Month 2, got %v", d.Month)
	}

	// Assert reminder lunar fields
	if len(cal.Reminders) != 1 {
		t.Fatalf("Expected 1 reminder, got %d", len(cal.Reminders))
	}
	r := cal.Reminders[0]
	if r.Label != "Lunar Reminder" {
		t.Errorf("Expected label 'Lunar Reminder', got '%s'", r.Label)
	}
	if r.CalendarType != "lunar" {
		t.Errorf("Expected CalendarType 'lunar', got '%s'", r.CalendarType)
	}
	if r.OriginalDay == nil || *r.OriginalDay != 15 {
		t.Errorf("Expected OriginalDay 15, got %v", r.OriginalDay)
	}
	if r.OriginalMonth == nil || *r.OriginalMonth != 1 {
		t.Errorf("Expected OriginalMonth 1, got %v", r.OriginalMonth)
	}
	if r.OriginalYear == nil || *r.OriginalYear != 2025 {
		t.Errorf("Expected OriginalYear 2025, got %v", r.OriginalYear)
	}
	if r.Day == nil || *r.Day != 12 {
		t.Errorf("Expected gregorian Day 12, got %v", r.Day)
	}
	if r.Month == nil || *r.Month != 2 {
		t.Errorf("Expected gregorian Month 2, got %v", r.Month)
	}
}

func TestCalendarProjectsYearlessLunarDatesForRequestedYear(t *testing.T) {
	svc, vaultID, userID, contactID := setupCalendarTest(t)

	storedDay, storedMonth := 19, 2 // Canonical 2000 projection; not the requested-year occurrence.
	originalDay, originalMonth := 15, 1
	if err := svc.db.Create(&models.ContactImportantDate{
		ContactID:     contactID,
		Label:         "Yearless lunar date",
		DatePrecision: importantDatePrecisionMonthDay,
		Day:           &storedDay,
		Month:         &storedMonth,
		CalendarType:  "lunar",
		OriginalDay:   &originalDay,
		OriginalMonth: &originalMonth,
	}).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}
	if err := svc.db.Create(&models.ContactReminder{
		ContactID:     contactID,
		Label:         "Yearless lunar reminder",
		Day:           &storedDay,
		Month:         &storedMonth,
		CalendarType:  "lunar",
		OriginalDay:   &originalDay,
		OriginalMonth: &originalMonth,
		Type:          "recurring_year",
	}).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	calendar2026, err := svc.GetCalendar(vaultID, userID, 3, 2026, "en")
	if err != nil {
		t.Fatalf("GetCalendar 2026 failed: %v", err)
	}
	if len(calendar2026.ImportantDates) != 1 || calendar2026.ImportantDates[0].Day == nil || *calendar2026.ImportantDates[0].Day != 3 {
		t.Fatalf("Expected lunar 1/15 on 2026-03-03, got %+v", calendar2026.ImportantDates)
	}
	if len(calendar2026.Reminders) != 1 || calendar2026.Reminders[0].Day == nil || *calendar2026.Reminders[0].Day != 3 {
		t.Fatalf("Expected recurring lunar reminder on 2026-03-03, got %+v", calendar2026.Reminders)
	}

	wrongMonth2026, err := svc.GetCalendar(vaultID, userID, 2, 2026, "en")
	if err != nil {
		t.Fatalf("GetCalendar wrong month failed: %v", err)
	}
	if len(wrongMonth2026.ImportantDates) != 0 || len(wrongMonth2026.Reminders) != 0 {
		t.Fatalf("Expected canonical projection month to be ignored in 2026, got dates=%d reminders=%d", len(wrongMonth2026.ImportantDates), len(wrongMonth2026.Reminders))
	}

	calendar2027, err := svc.GetCalendar(vaultID, userID, 2, 2027, "en")
	if err != nil {
		t.Fatalf("GetCalendar 2027 failed: %v", err)
	}
	if len(calendar2027.ImportantDates) != 1 || calendar2027.ImportantDates[0].Day == nil || *calendar2027.ImportantDates[0].Day != 20 {
		t.Fatalf("Expected lunar 1/15 on 2027-02-20, got %+v", calendar2027.ImportantDates)
	}
	if len(calendar2027.Reminders) != 1 || calendar2027.Reminders[0].Day == nil || *calendar2027.Reminders[0].Day != 20 {
		t.Fatalf("Expected recurring lunar reminder on 2027-02-20, got %+v", calendar2027.Reminders)
	}
}

func TestCalendarDayFiltersOneTimeRemindersByYear(t *testing.T) {
	svc, vaultID, userID, contactID := setupCalendarTest(t)
	day, month, year := 15, 3, 2025
	if err := svc.db.Create(&models.ContactReminder{
		ContactID: contactID,
		Label:     "Only in 2025",
		Day:       &day,
		Month:     &month,
		Year:      &year,
		Type:      "one_time",
	}).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	wrongYear, err := svc.GetCalendarDay(vaultID, userID, 2026, month, day, "en")
	if err != nil {
		t.Fatalf("GetCalendarDay wrong year failed: %v", err)
	}
	if len(wrongYear.Reminders) != 0 {
		t.Fatalf("Expected no one-time reminders in 2026, got %+v", wrongYear.Reminders)
	}

	matchingYear, err := svc.GetCalendarDay(vaultID, userID, year, month, day, "en")
	if err != nil {
		t.Fatalf("GetCalendarDay matching year failed: %v", err)
	}
	if len(matchingYear.Reminders) != 1 {
		t.Fatalf("Expected one reminder in 2025, got %+v", matchingYear.Reminders)
	}
}

func TestCalendarUsesVaultNameOrderOverride(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "calendar-name-order@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Name Order Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	override := "%last_name%, %first_name% {nickname? (%nickname%)}"
	if err := db.Model(&models.User{}).Where("id = ?", resp.User.ID).Update("name_order", override).Error; err != nil {
		t.Fatalf("update user name order failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{
		FirstName: "Jane",
		LastName:  "Doe",
		Nickname:  "JD",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	day := 15
	month := 3
	year := 2025
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Birthday",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}).Error; err != nil {
		t.Fatalf("Create important date failed: %v", err)
	}
	if err := db.Create(&models.ContactReminder{
		ContactID: contact.ID,
		Label:     "Call Jane",
		Day:       &day,
		Month:     &month,
		Type:      "one_time",
	}).Error; err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	svc := NewCalendarService(db)
	cal, err := svc.GetCalendar(vault.ID, resp.User.ID, 3, 2025, "en")
	if err != nil {
		t.Fatalf("GetCalendar failed: %v", err)
	}
	if len(cal.ImportantDates) != 1 {
		t.Fatalf("Expected 1 important date, got %d", len(cal.ImportantDates))
	}
	if cal.ImportantDates[0].ContactName != "Doe, Jane (JD)" {
		t.Errorf("Expected important date contact_name to use vault name_order override, got %q", cal.ImportantDates[0].ContactName)
	}
	if len(cal.Reminders) != 1 {
		t.Fatalf("Expected 1 reminder, got %d", len(cal.Reminders))
	}
	if cal.Reminders[0].ContactName != "Doe, Jane (JD)" {
		t.Errorf("Expected reminder contact_name to use vault name_order override, got %q", cal.Reminders[0].ContactName)
	}
}

func TestParseIntParam(t *testing.T) {
	if v := ParseIntParam("", 5); v != 5 {
		t.Errorf("Expected default 5 for empty string, got %d", v)
	}
	if v := ParseIntParam("abc", 10); v != 10 {
		t.Errorf("Expected default 10 for invalid string, got %d", v)
	}
	if v := ParseIntParam("42", 0); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}
}
