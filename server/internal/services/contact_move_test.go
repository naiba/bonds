package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactMoveTest(t *testing.T) (*ContactMoveService, string, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "move-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault1, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 1"})
	if err != nil {
		t.Fatalf("CreateVault 1 failed: %v", err)
	}

	vault2, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 2"})
	if err != nil {
		t.Fatalf("CreateVault 2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault1.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewContactMoveService(db), contact.ID, vault1.ID, vault2.ID, resp.User.ID
}

func TestMoveContact(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)

	resp, err := svc.Move(contactID, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if resp.VaultID != vault2ID {
		t.Errorf("Expected vault_id '%s', got '%s'", vault2ID, resp.VaultID)
	}
}

func TestMoveContactNotFound(t *testing.T) {
	svc, _, vault1ID, vault2ID, userID := setupContactMoveTest(t)

	_, err := svc.Move("nonexistent-id", vault1ID, vault2ID, userID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestMoveContactTargetVaultNotFound(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)

	_, err := svc.Move(contactID, vault1ID, "nonexistent-vault", userID)
	if err != ErrTargetVaultNotFound {
		t.Errorf("Expected ErrTargetVaultNotFound, got %v", err)
	}
}
