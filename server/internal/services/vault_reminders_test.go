package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultReminderTest(t *testing.T) (*VaultReminderService, *ReminderService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vr-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	contactSvc := NewContactService(db)
	contact, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})

	return NewVaultReminderService(db), NewReminderService(db), contact.ID, vault.ID, resp.User.ID
}

func TestListVaultReminders(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID, userID := setupVaultReminderTest(t)

	day := 15
	month := 6
	_, _ = rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Birthday", Day: &day, Month: &month, Type: "one_time",
	})

	reminders, err := vrSvc.List(vaultID, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 1 {
		t.Errorf("Expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", reminders[0].Label)
	}
}

func TestListVaultRemindersEmpty(t *testing.T) {
	vrSvc, _, _, vaultID, userID := setupVaultReminderTest(t)

	reminders, err := vrSvc.List(vaultID, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 0 {
		t.Errorf("Expected 0 reminders, got %d", len(reminders))
	}
}

// Regression for #87: Upcoming Reminders widget previews the next reminders
// chronologically. Service must order by next-occurrence date (ascending),
// not by created_at, so the dashboard glance shows the soonest events first.
func TestListVaultReminders_OrderedByUpcomingDate(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID, userID := setupVaultReminderTest(t)

	now := time.Now()
	// Create three recurring-yearly reminders intentionally inserted in the
	// "wrong" chronological order so created_at != upcoming-date order.
	in60 := now.AddDate(0, 0, 60)
	in10 := now.AddDate(0, 0, 10)
	in30 := now.AddDate(0, 0, 30)

	specs := []struct {
		label string
		t     time.Time
	}{
		{"in60", in60},
		{"in10", in10},
		{"in30", in30},
	}
	for _, s := range specs {
		d, m := s.t.Day(), int(s.t.Month())
		if _, err := rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
			Label: s.label, Day: &d, Month: &m, Type: "recurring_year",
		}); err != nil {
			t.Fatalf("create %s: %v", s.label, err)
		}
	}

	reminders, err := vrSvc.List(vaultID, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 3 {
		t.Fatalf("expected 3 reminders, got %d", len(reminders))
	}

	wantOrder := []string{"in10", "in30", "in60"}
	for i, want := range wantOrder {
		if reminders[i].Label != want {
			gotLabels := make([]string, len(reminders))
			for j, r := range reminders {
				gotLabels[j] = r.Label
			}
			t.Fatalf("position %d: want %s, got %s (full order=%v)", i, want, reminders[i].Label, gotLabels)
		}
	}
}

func TestListVaultReminders_RendersMonthDayReminderWithoutYear(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID, userID := setupVaultReminderTest(t)

	day := 15
	month := 3
	_, err := rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Yearless reminder",
		Day:   &day,
		Month: &month,
		Type:  "recurring_year",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reminders, err := vrSvc.List(vaultID, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].Year != nil {
		t.Fatalf("expected vault reminder to remain yearless, got %v", reminders[0].Year)
	}
}

func TestListVaultRemindersUsesVaultNameOrderForContactName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	authSvc := NewAuthService(db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(db)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-reminder-name-order@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Formatting Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	if err := db.Model(&models.Vault{}).Where("id = ?", vault.ID).Update("name_order", "%last_name%, %first_name% {middle_name? %middle_name%} {nickname? (%nickname%)} {maiden_name? %maiden_name%}").Error; err != nil {
		t.Fatalf("Update vault name_order failed: %v", err)
	}
	contact, err := NewContactService(db).CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{
		FirstName:  "Alice",
		LastName:   "Zephyr",
		MiddleName: "Marie",
		Nickname:   "Ace",
		MaidenName: "Smith",
		Prefix:     "Dr.",
		Suffix:     "Jr.",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}
	reminderSvc := NewReminderService(db)
	day := 15
	month := 6
	if _, err := reminderSvc.Create(contact.ID, vault.ID, dto.CreateReminderRequest{Label: "Birthday", Day: &day, Month: &month, Type: "one_time"}); err != nil {
		t.Fatalf("Create reminder failed: %v", err)
	}

	reminders, err := NewVaultReminderService(db).List(vault.ID, resp.User.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 1 {
		t.Fatalf("Expected 1 reminder, got %d", len(reminders))
	}
	if reminders[0].ContactName != "Dr. Zephyr, Alice Marie (Ace) Smith Jr." {
		t.Fatalf("Expected formatted contact name 'Dr. Zephyr, Alice Marie (Ace) Smith Jr.', got '%s'", reminders[0].ContactName)
	}
	if reminders[0].ContactFirstName != "Alice" || reminders[0].ContactLastName != "Zephyr" {
		t.Fatalf("Expected raw contact names Alice/Zephyr, got %s/%s", reminders[0].ContactFirstName, reminders[0].ContactLastName)
	}
}
