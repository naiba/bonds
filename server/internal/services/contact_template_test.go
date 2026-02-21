package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactTemplateTest(t *testing.T) (*ContactTemplateService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "template-test@example.com",
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

	return NewContactTemplateService(db), contact.ID, vault.ID
}

func TestContactTemplateUpdate(t *testing.T) {
	svc, contactID, vaultID := setupContactTemplateTest(t)

	templateID := uint(1)
	resp, err := svc.UpdateTemplate(contactID, vaultID, dto.UpdateContactTemplateRequest{TemplateID: &templateID})
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactTemplateUpdateClear(t *testing.T) {
	svc, contactID, vaultID := setupContactTemplateTest(t)

	templateID := uint(1)
	_, err := svc.UpdateTemplate(contactID, vaultID, dto.UpdateContactTemplateRequest{TemplateID: &templateID})
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}

	resp, err := svc.UpdateTemplate(contactID, vaultID, dto.UpdateContactTemplateRequest{TemplateID: nil})
	if err != nil {
		t.Fatalf("UpdateTemplate to nil failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactTemplateUpdateNotFound(t *testing.T) {
	svc, _, vaultID := setupContactTemplateTest(t)

	templateID := uint(1)
	_, err := svc.UpdateTemplate("nonexistent-id", vaultID, dto.UpdateContactTemplateRequest{TemplateID: &templateID})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}
