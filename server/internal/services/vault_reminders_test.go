package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultReminderTest(t *testing.T) (*VaultReminderService, *ReminderService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "vr-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	contactSvc := NewContactService(db)
	contact, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})

	return NewVaultReminderService(db), NewReminderService(db), contact.ID, vault.ID
}

func TestListVaultReminders(t *testing.T) {
	vrSvc, rSvc, contactID, vaultID := setupVaultReminderTest(t)

	day := 15
	month := 6
	_, _ = rSvc.Create(contactID, vaultID, dto.CreateReminderRequest{
		Label: "Birthday", Day: &day, Month: &month, Type: "one_time",
	})

	reminders, err := vrSvc.List(vaultID)
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
	vrSvc, _, _, vaultID := setupVaultReminderTest(t)

	reminders, err := vrSvc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 0 {
		t.Errorf("Expected 0 reminders, got %d", len(reminders))
	}
}
