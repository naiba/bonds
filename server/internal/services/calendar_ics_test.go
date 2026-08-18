package services

import (
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func TestExportVaultICS(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "User",
		Email:     "ical-export@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Ical Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	day, month, year := 15, 3, 2025
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Birthday",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}).Error; err != nil {
		t.Fatalf("create important date failed: %v", err)
	}

	if err := db.Create(&models.ContactReminder{
		ContactID: contact.ID,
		Label:     "Call Jane",
		Day:       &day,
		Month:     &month,
		Type:      "one_time",
	}).Error; err != nil {
		t.Fatalf("create reminder failed: %v", err)
	}

	due := time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ContactTask{
		VaultID:    vault.ID,
		AuthorName: "Ical User",
		Label:      "Standalone Task",
		Status:     models.TaskStatusTodo,
		DueAt:      &due,
	}).Error; err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.Activity{
		VaultID: vault.ID, StartDate: &start, StartPrecision: "day",
		EndStatus: "none", Title: "Graduation",
	}).Error; err != nil {
		t.Fatalf("create activity failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)

	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"END:VCALENDAR",
		"BEGIN:VEVENT",
		"BEGIN:VTODO",
		"SUMMARY:Jane - Birthday",
		"SUMMARY:Call Jane",
		"SUMMARY:Standalone Task",
		"SUMMARY:Graduation",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected ICS output to contain %q\n---\n%s", want, out)
		}
	}
}

func TestExportVaultICSImportantDateSummaryIncludesContactName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "Summary",
		Email:     "ical-summary@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Summary Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane", LastName: "Doe"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	day, month, year := 15, 3, 2025
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Birthdate",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}).Error; err != nil {
		t.Fatalf("create important date failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, "SUMMARY:Jane Doe - Birthdate") {
		t.Fatalf("expected important date summary to include contact name\n---\n%s", out)
	}
	if strings.Contains(out, "SUMMARY:Birthdate\r\n") {
		t.Fatalf("expected no standalone important date summary\n---\n%s", out)
	}
}

func TestExportVaultICSUsesVaultNameOrderOverride(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "Override",
		Email:     "ical-name-order@example.com",
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

	day, month, year := 15, 3, 2025
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Birthdate",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}).Error; err != nil {
		t.Fatalf("create important date failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, "SUMMARY:Doe\\, Jane (JD) - Birthdate") {
		t.Fatalf("expected important date summary to use vault name_order override\n---\n%s", out)
	}
}

func TestExportVaultICSEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Empty",
		LastName:  "User",
		Email:     "ical-empty@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Empty Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "BEGIN:VCALENDAR") || !strings.Contains(out, "END:VCALENDAR") {
		t.Errorf("expected a valid empty VCALENDAR, got:\n%s", out)
	}
}

func TestExportVaultICSRecurrenceRules(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "Recur",
		Email:     "ical-recur@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Recur Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// A date recorded WITH a year is a single occurrence → no RRULE.
	day, month, year := 15, 3, 2025
	if err := db.Create(&models.ContactImportantDate{
		ContactID: contact.ID,
		Label:     "Anniversary",
		Day:       &day,
		Month:     &month,
		Year:      &year,
	}).Error; err != nil {
		t.Fatalf("create dated important date failed: %v", err)
	}
	// A year-unknown date repeats yearly.
	if err := db.Create(&models.ContactImportantDate{
		ContactID:     contact.ID,
		Label:         "Birthday",
		Day:           &day,
		Month:         &month,
		IsYearUnknown: true,
	}).Error; err != nil {
		t.Fatalf("create year-unknown important date failed: %v", err)
	}

	// Reminders map their Type onto an RRULE; one_time gets none.
	if err := db.Create(&models.ContactReminder{
		ContactID: contact.ID,
		Label:     "One Off",
		Day:       &day,
		Month:     &month,
		Type:      "one_time",
	}).Error; err != nil {
		t.Fatalf("create one_time reminder failed: %v", err)
	}
	interval := 2
	if err := db.Create(&models.ContactReminder{
		ContactID:       contact.ID,
		Label:           "Monthly",
		Day:             &day,
		Month:           &month,
		Type:            "recurring_month",
		FrequencyNumber: &interval,
	}).Error; err != nil {
		t.Fatalf("create recurring_month reminder failed: %v", err)
	}
	if err := db.Create(&models.ContactReminder{
		ContactID: contact.ID,
		Label:     "Yearly",
		Day:       &day,
		Month:     &month,
		Type:      "recurring_year",
	}).Error; err != nil {
		t.Fatalf("create recurring_year reminder failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, "DTSTART;VALUE=DATE:20250315\r\nSUMMARY:Jane - Anniversary\r\n") {
		t.Errorf("expected dated important date to be a single occurrence without RRULE\n---\n%s", out)
	}
	if strings.Contains(out, "DTSTART;VALUE=DATE:20250315\r\nSUMMARY:Jane - Anniversary\r\nRRULE") {
		t.Errorf("expected dated important date to carry no RRULE\n---\n%s", out)
	}
	if !strings.Contains(out, "DTSTART;VALUE=DATE:"+time.Now().Format("2006")+"0315\r\nRRULE:FREQ=YEARLY\r\nSUMMARY:Jane - Birthday") {
		t.Errorf("expected year-unknown date projected onto the current year with RRULE:FREQ=YEARLY\n---\n%s", out)
	}
	if !strings.Contains(out, "DTSTART;VALUE=DATE:"+time.Now().Format("2006")+"0315\r\nRRULE:FREQ=MONTHLY;INTERVAL=2\r\nSUMMARY:Monthly") {
		t.Errorf("expected monthly reminder with INTERVAL=2\n---\n%s", out)
	}
	if !strings.Contains(out, "DTSTART;VALUE=DATE:"+time.Now().Format("2006")+"0315\r\nRRULE:FREQ=YEARLY\r\nSUMMARY:Yearly") {
		t.Errorf("expected yearly reminder with RRULE:FREQ=YEARLY\n---\n%s", out)
	}
	if !strings.Contains(out, "SUMMARY:One Off\r\n") {
		t.Errorf("expected one_time reminder in output\n---\n%s", out)
	}
	idx := strings.Index(out, "SUMMARY:One Off\r\n")
	if idx < 0 {
		t.Errorf("expected one_time reminder in output\n---\n%s", out)
	} else {
		start := strings.LastIndex(out[:idx], "BEGIN:VEVENT")
		end := strings.Index(out[idx:], "END:VEVENT")
		if block := out[start : idx+end+len("END:VEVENT")]; strings.Contains(block, "RRULE") {
			t.Errorf("expected one_time reminder to carry no RRULE\n---\n%s", out)
		}
	}
}

func TestExportVaultICSLunarMonthDayProjection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "Lunar",
		Email:     "ical-lunar@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Lunar Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Lunar month+day only: stored Year is nil, so DTSTART must be projected
	// onto the current year and coincide with the first RDATE instead of
	// degrading to a phantom Jan 1.
	origDay, origMonth := 5, 8
	if err := db.Create(&models.ContactImportantDate{
		ContactID:      contact.ID,
		Label:          "Lunar Fest",
		CalendarType:   "lunar",
		OriginalDay:    &origDay,
		OriginalMonth:  &origMonth,
		IsYearUnknown:  true,
		DatePrecision:  "full",
	}).Error; err != nil {
		t.Fatalf("create lunar important date failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, "DTSTART;VALUE=DATE:") {
		t.Fatalf("expected a projected DTSTART\n---\n%s", out)
	}
	if !strings.Contains(out, "RDATE;VALUE=DATE:") {
		t.Fatalf("expected RDATE projections for lunar recurrence\n---\n%s", out)
	}
	// The DTSTART must be the first RDATE entry, never a Jan 1 phantom.
	if strings.Contains(out, "DTSTART;VALUE=DATE:"+time.Now().Format("2006")+"0101") {
		t.Errorf("expected DTSTART to be a projected lunar date, not Jan 1 of the current year\n---\n%s", out)
	}
}
