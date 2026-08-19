package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	calendarPkg "github.com/naiba/bonds/internal/calendar"
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

	// A date recorded WITH a year still repeats yearly — the year is display-only
	// (see CalendarService.GetCalendar), so the subscription must not stop at it.
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

	if !strings.Contains(out, "DTSTART;VALUE=DATE:20250315\r\nRRULE:FREQ=YEARLY\r\nSUMMARY:Jane - Anniversary") {
		t.Errorf("expected dated important date to keep yearly recurrence (year is display-only)\n---\n%s", out)
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
		ContactID:     contact.ID,
		Label:         "Lunar Fest",
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		IsYearUnknown: true,
		DatePrecision: "full",
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

func TestExportVaultICSLunarReminderIgnoresStored2000Projection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(db)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Ical",
		LastName:  "Reminder",
		Email:     "ical-lunar-reminder@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Lunar Reminder Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	contact, err := NewContactService(db).CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	converter, ok := calendarPkg.Get(calendarPkg.Lunar)
	if !ok {
		t.Fatal("expected lunar converter")
	}
	origDay, origMonth := 1, 1
	stored, err := calendarOccurrenceInYear(converter, calendarPkg.DateInfo{Day: origDay, Month: origMonth}, calendarProjectionReferenceYear)
	if err != nil {
		t.Fatalf("project storage reference date: %v", err)
	}
	storedDay, storedMonth, storedYear := stored.Day, stored.Month, stored.Year
	reminder := models.ContactReminder{
		ContactID:     contact.ID,
		Label:         "Lunar Reminder",
		Day:           &storedDay,
		Month:         &storedMonth,
		Year:          &storedYear,
		CalendarType:  "lunar",
		OriginalDay:   &origDay,
		OriginalMonth: &origMonth,
		Type:          "recurring_year",
	}
	if err := db.Create(&reminder).Error; err != nil {
		t.Fatalf("create lunar reminder failed: %v", err)
	}

	data, err := NewCalendarICSService(db).ExportVault(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	block := icsEventBlockForSummary(t, string(data), "Lunar Reminder")
	currentYear := time.Now().UTC().Year()
	expectedStart, err := calendarOccurrenceInYear(converter, calendarPkg.DateInfo{Day: origDay, Month: origMonth}, currentYear)
	if err != nil {
		t.Fatalf("project current occurrence: %v", err)
	}
	expectedStartValue := fmt.Sprintf("%04d%02d%02d", expectedStart.Year, expectedStart.Month, expectedStart.Day)
	if !strings.Contains(block, "DTSTART;VALUE=DATE:"+expectedStartValue) {
		t.Fatalf("expected current-year DTSTART %s, got\n---\n%s", expectedStartValue, block)
	}
	if strings.Contains(block, "DTSTART;VALUE=DATE:2000") {
		t.Fatalf("expected DTSTART to ignore fixed 2000 storage projection\n---\n%s", block)
	}

	startCalendarYear := calendarYearForGregorianDate(converter, time.Date(expectedStart.Year, time.Month(expectedStart.Month), expectedStart.Day, 0, 0, 0, 0, time.UTC))
	beyondOldHorizon, err := converter.ToGregorian(calendarPkg.DateInfo{Day: origDay, Month: origMonth, Year: startCalendarYear + 20})
	if err != nil {
		t.Fatalf("project occurrence beyond old horizon: %v", err)
	}
	beyondOldHorizonValue := fmt.Sprintf("%04d%02d%02d", beyondOldHorizon.Year, beyondOldHorizon.Month, beyondOldHorizon.Day)
	if !strings.Contains(block, beyondOldHorizonValue) {
		t.Fatalf("expected rolling recurrence to include %s beyond the old 10-year horizon\n---\n%s", beyondOldHorizonValue, block)
	}
}

func icsEventBlockForSummary(t *testing.T, output, summary string) string {
	t.Helper()
	marker := "SUMMARY:" + summary + "\r\n"
	markerIndex := strings.Index(output, marker)
	if markerIndex < 0 {
		t.Fatalf("missing event summary %q\n---\n%s", summary, output)
	}
	start := strings.LastIndex(output[:markerIndex], "BEGIN:VEVENT")
	endOffset := strings.Index(output[markerIndex:], "END:VEVENT")
	if start < 0 || endOffset < 0 {
		t.Fatalf("invalid event block for summary %q\n---\n%s", summary, output)
	}
	return output[start : markerIndex+endOffset+len("END:VEVENT")]
}
