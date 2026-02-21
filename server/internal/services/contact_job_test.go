package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactJobTest(t *testing.T) (*ContactJobService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "job-test@example.com",
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

	return NewContactJobService(db), contact.ID, vault.ID
}

func TestContactJobUpdate(t *testing.T) {
	svc, contactID, vaultID := setupContactJobTest(t)

	companyID := uint(1)
	resp, err := svc.Update(contactID, vaultID, dto.UpdateJobInfoRequest{
		CompanyID:   &companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactJobDelete(t *testing.T) {
	svc, contactID, vaultID := setupContactJobTest(t)

	companyID := uint(1)
	_, err := svc.Update(contactID, vaultID, dto.UpdateJobInfoRequest{
		CompanyID:   &companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resp, err := svc.Delete(contactID, vaultID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if resp.ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, resp.ID)
	}
}

func TestContactJobUpdateNotFound(t *testing.T) {
	svc, _, vaultID := setupContactJobTest(t)

	_, err := svc.Update("nonexistent-id", vaultID, dto.UpdateJobInfoRequest{JobPosition: "Engineer"})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJobDeleteNotFound(t *testing.T) {
	svc, _, vaultID := setupContactJobTest(t)

	_, err := svc.Delete("nonexistent-id", vaultID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}
