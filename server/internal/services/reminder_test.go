package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupReminderTest(t *testing.T) (*ReminderService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "reminder-test@example.com",
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

	return NewReminderService(db), contact.ID
}

func intPtr(v int) *int {
	return &v
}

func TestCreateReminder(t *testing.T) {
	svc, contactID := setupReminderTest(t)

	reminder, err := svc.Create(contactID, dto.CreateReminderRequest{
		Label: "Birthday",
		Day:   intPtr(15),
		Month: intPtr(6),
		Year:  intPtr(1990),
		Type:  "one_time",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if reminder.Label != "Birthday" {
		t.Errorf("Expected label 'Birthday', got '%s'", reminder.Label)
	}
	if reminder.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, reminder.ContactID)
	}
	if reminder.Type != "one_time" {
		t.Errorf("Expected type 'one_time', got '%s'", reminder.Type)
	}
	if reminder.ID == 0 {
		t.Error("Expected reminder ID to be non-zero")
	}
}

func TestListReminders(t *testing.T) {
	svc, contactID := setupReminderTest(t)

	_, err := svc.Create(contactID, dto.CreateReminderRequest{Label: "Reminder 1", Type: "one_time"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, dto.CreateReminderRequest{Label: "Reminder 2", Type: "recurring"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reminders, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 2 {
		t.Errorf("Expected 2 reminders, got %d", len(reminders))
	}
}

func TestUpdateReminder(t *testing.T) {
	svc, contactID := setupReminderTest(t)

	created, err := svc.Create(contactID, dto.CreateReminderRequest{Label: "Original", Type: "one_time"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, dto.UpdateReminderRequest{
		Label: "Updated",
		Day:   intPtr(25),
		Month: intPtr(12),
		Type:  "recurring",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "Updated" {
		t.Errorf("Expected label 'Updated', got '%s'", updated.Label)
	}
	if updated.Type != "recurring" {
		t.Errorf("Expected type 'recurring', got '%s'", updated.Type)
	}
}

func TestDeleteReminder(t *testing.T) {
	svc, contactID := setupReminderTest(t)

	created, err := svc.Create(contactID, dto.CreateReminderRequest{Label: "To delete", Type: "one_time"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	reminders, err := svc.List(contactID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(reminders) != 0 {
		t.Errorf("Expected 0 reminders after delete, got %d", len(reminders))
	}
}

func TestReminderNotFound(t *testing.T) {
	svc, contactID := setupReminderTest(t)

	_, err := svc.Update(9999, contactID, dto.UpdateReminderRequest{Label: "nope", Type: "one_time"})
	if err != ErrReminderNotFound {
		t.Errorf("Expected ErrReminderNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID)
	if err != ErrReminderNotFound {
		t.Errorf("Expected ErrReminderNotFound, got %v", err)
	}
}
