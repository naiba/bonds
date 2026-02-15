package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactInformationTest(t *testing.T) (*ContactInformationService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-info-test@example.com",
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

	return NewContactInformationService(db), contact.ID, vault.ID
}

func TestCreateContactInformation(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	info, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1,
		Data:   "john@example.com",
		Kind:   "personal",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if info.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, info.ContactID)
	}
	if info.TypeID != 1 {
		t.Errorf("Expected type_id 1, got %d", info.TypeID)
	}
	if info.Data != "john@example.com" {
		t.Errorf("Expected data 'john@example.com', got '%s'", info.Data)
	}
	if info.Kind != "personal" {
		t.Errorf("Expected kind 'personal', got '%s'", info.Kind)
	}
	if !info.Pref {
		t.Error("Expected pref to be true by default")
	}
	if info.ID == 0 {
		t.Error("Expected contact information ID to be non-zero")
	}
}

func TestCreateContactInformationWithKind(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	info, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1,
		Data:   "john@work.com",
		Kind:   "work",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if info.Kind != "work" {
		t.Errorf("Expected kind 'work', got '%s'", info.Kind)
	}
	if info.Data != "john@work.com" {
		t.Errorf("Expected data 'john@work.com', got '%s'", info.Data)
	}
}

func TestListContactInformation(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{TypeID: 1, Data: "email@test.com"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{TypeID: 2, Data: "+1234567890"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	items, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 contact information items, got %d", len(items))
	}
}

func TestUpdateContactInformation(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1,
		Data:   "old@example.com",
		Kind:   "personal",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	pref := false
	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateContactInformationRequest{
		TypeID: 2,
		Data:   "new@example.com",
		Kind:   "work",
		Pref:   &pref,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.TypeID != 2 {
		t.Errorf("Expected type_id 2, got %d", updated.TypeID)
	}
	if updated.Data != "new@example.com" {
		t.Errorf("Expected data 'new@example.com', got '%s'", updated.Data)
	}
	if updated.Kind != "work" {
		t.Errorf("Expected kind 'work', got '%s'", updated.Kind)
	}
	if updated.Pref {
		t.Error("Expected pref to be false after update")
	}
}

func TestDeleteContactInformation(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1,
		Data:   "to-delete@example.com",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	items, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 contact information after delete, got %d", len(items))
	}
}

func TestDeleteContactInformationNotFound(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	err := svc.Delete(9999, contactID, vaultID)
	if err != ErrContactInformationNotFound {
		t.Errorf("Expected ErrContactInformationNotFound, got %v", err)
	}
}
