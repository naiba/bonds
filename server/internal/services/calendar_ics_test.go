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

	timeline := &models.TimelineEvent{VaultID: vault.ID, StartedAt: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)}
	if err := db.Create(timeline).Error; err != nil {
		t.Fatalf("create timeline event failed: %v", err)
	}
	summary := "Graduation"
	if err := db.Create(&models.LifeEvent{
		TimelineEventID: timeline.ID,
		LifeEventTypeID: 0,
		HappenedAt:      time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
		Summary:         &summary,
	}).Error; err != nil {
		t.Fatalf("create life event failed: %v", err)
	}

	svc := NewCalendarICSService(db)
	data, err := svc.ExportVault(vault.ID)
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
	data, err := svc.ExportVault(vault.ID)
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
	data, err := svc.ExportVault(vault.ID)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "BEGIN:VCALENDAR") || !strings.Contains(out, "END:VCALENDAR") {
		t.Errorf("expected a valid empty VCALENDAR, got:\n%s", out)
	}
}
