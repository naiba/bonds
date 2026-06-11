package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
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
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault1, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 1"}, "en")
	if err != nil {
		t.Fatalf("CreateVault 1 failed: %v", err)
	}

	vault2, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 2"}, "en")
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

func TestMoveContactClearsCrossVaultFirstMetThrough(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	introducedBy, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{
		FirstName: "Source",
		LastName:  "Introducer",
	})
	if err != nil {
		t.Fatalf("Create introducer failed: %v", err)
	}
	if _, err := contactSvc.UpdateContact(contactID, vault1ID, userID, dto.UpdateContactRequest{
		FirstName:                "John",
		FirstMetThroughContactID: &introducedBy.ID,
	}); err != nil {
		t.Fatalf("Set first_met_through_contact_id failed: %v", err)
	}

	resp, err := svc.Move(contactID, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if resp.FirstMetThroughContactID != nil {
		t.Fatalf("expected moved response to clear source-vault met-through id, got %v", *resp.FirstMetThroughContactID)
	}
	if resp.FirstMetThroughContact != nil {
		t.Fatalf("expected moved response not to leak source-vault met-through contact, got %+v", resp.FirstMetThroughContact)
	}

	reloaded, err := contactSvc.GetContact(contactID, userID, vault2ID)
	if err != nil {
		t.Fatalf("Get moved contact failed: %v", err)
	}
	if reloaded.FirstMetThroughContactID != nil {
		t.Fatalf("expected moved contact to persist cleared met-through id, got %v", *reloaded.FirstMetThroughContactID)
	}
	if reloaded.FirstMetThroughContact != nil {
		t.Fatalf("expected moved contact not to return source-vault met-through contact, got %+v", reloaded.FirstMetThroughContact)
	}

	contacts, _, err := contactSvc.ListContacts(vault2ID, userID, 1, 15, "", "first_name", "")
	if err != nil {
		t.Fatalf("List moved contacts failed: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != contactID {
		t.Fatalf("expected moved contact in target vault list, got %+v", contacts)
	}
	if contacts[0].FirstMetThroughContactID != nil {
		t.Fatalf("expected target vault list to hide source-vault met-through id, got %v", *contacts[0].FirstMetThroughContactID)
	}
	if contacts[0].FirstMetThroughContact != nil {
		t.Fatalf("expected target vault list not to leak source-vault met-through contact, got %+v", contacts[0].FirstMetThroughContact)
	}
}

func TestMoveContactTargetVaultForbidden(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)
	targetVaultID := createMoveAuthTargetVault(t, svc, "move-target-forbidden@example.com")

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrVaultForbidden) {
		t.Fatalf("expected ErrVaultForbidden, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
}

func TestMoveContactTargetVaultInsufficientPermission(t *testing.T) {
	svc, contactID, vault1ID, targetVaultID, userID := setupContactMoveTest(t)
	if err := svc.db.Model(&models.UserVault{}).
		Where("vault_id = ? AND user_id = ?", targetVaultID, userID).
		Update("permission", models.PermissionViewer).Error; err != nil {
		t.Fatalf("Set viewer target vault access failed: %v", err)
	}

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrInsufficientPerm) {
		t.Fatalf("expected ErrInsufficientPerm, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
}

func TestMoveContactCrossAccountTargetForbidden(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)
	targetVaultID := createMoveAuthTargetVault(t, svc, "move-cross-account-target@example.com")
	sourceVault := getMoveTestVault(t, svc, vault1ID)
	targetVault := getMoveTestVault(t, svc, targetVaultID)
	if sourceVault.AccountID == targetVault.AccountID {
		t.Fatal("expected source and target vaults to belong to different accounts")
	}
	if err := svc.db.Create(&models.UserVault{
		VaultID:    targetVaultID,
		UserID:     userID,
		Permission: models.PermissionEditor,
	}).Error; err != nil {
		t.Fatalf("Add editor target vault access failed: %v", err)
	}

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrVaultForbidden) {
		t.Fatalf("expected ErrVaultForbidden, got %v", err)
	}
	if errors.Is(err, ErrTargetVaultNotFound) {
		t.Fatal("expected cross-account target vault to be forbidden, not reported missing")
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
	assertMoveContactNotInVault(t, svc, contactID, targetVaultID)
}

func TestMoveContactShadowSelfContactNotFound(t *testing.T) {
	svc, _, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	sourceUserVault := getMoveTestUserVault(t, svc, userID, vault1ID)
	targetUserVault := getMoveTestUserVault(t, svc, userID, vault2ID)
	shadowContactID := sourceUserVault.ContactID
	if shadowContactID == "" {
		t.Fatal("expected source user vault to have a shadow contact")
	}
	if shadowContactID == targetUserVault.ContactID {
		t.Fatal("expected each vault to have a distinct shadow contact")
	}

	_, err := svc.Move(shadowContactID, vault1ID, vault2ID, userID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, shadowContactID, vault1ID)
	shadowContact := getMoveTestContact(t, svc, shadowContactID)
	if shadowContact.CanBeDeleted || shadowContact.Listed {
		t.Fatalf("expected shadow contact to remain hidden and undeletable, got can_be_deleted=%v listed=%v", shadowContact.CanBeDeleted, shadowContact.Listed)
	}
	reloadedSourceUserVault := getMoveTestUserVault(t, svc, userID, vault1ID)
	reloadedTargetUserVault := getMoveTestUserVault(t, svc, userID, vault2ID)
	if reloadedSourceUserVault.ContactID != shadowContactID {
		t.Fatalf("expected source UserVault.ContactID to remain %s, got %s", shadowContactID, reloadedSourceUserVault.ContactID)
	}
	if reloadedTargetUserVault.ContactID != targetUserVault.ContactID {
		t.Fatalf("expected target UserVault.ContactID to remain %s, got %s", targetUserVault.ContactID, reloadedTargetUserVault.ContactID)
	}
	if reloadedTargetUserVault.ContactID == shadowContactID {
		t.Fatal("expected target UserVault.ContactID not to point at source shadow contact")
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

func createMoveAuthTargetVault(t *testing.T, svc *ContactMoveService, email string) string {
	t.Helper()
	authSvc := NewAuthService(svc.db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(svc.db)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Target",
		LastName:  "Owner",
		Email:     email,
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register target owner failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Target Vault"}, "en")
	if err != nil {
		t.Fatalf("Create target vault failed: %v", err)
	}
	return vault.ID
}

func assertMoveContactRemainsInVault(t *testing.T, svc *ContactMoveService, contactID, vaultID string) {
	t.Helper()
	contact := getMoveTestContact(t, svc, contactID)
	if contact.VaultID != vaultID {
		t.Fatalf("expected contact to remain in source vault %s, got %s", vaultID, contact.VaultID)
	}
}

func assertMoveContactNotInVault(t *testing.T, svc *ContactMoveService, contactID, vaultID string) {
	t.Helper()
	var count int64
	if err := svc.db.Model(&models.Contact{}).Where("id = ? AND vault_id = ?", contactID, vaultID).Count(&count).Error; err != nil {
		t.Fatalf("Count contact in vault failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no contact %s in vault %s, got %d", contactID, vaultID, count)
	}
}

func getMoveTestContact(t *testing.T, svc *ContactMoveService, contactID string) models.Contact {
	t.Helper()
	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", contactID).Error; err != nil {
		t.Fatalf("Reload contact failed: %v", err)
	}
	return contact
}

func getMoveTestVault(t *testing.T, svc *ContactMoveService, vaultID string) models.Vault {
	t.Helper()
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("Load vault failed: %v", err)
	}
	return vault
}

func getMoveTestUserVault(t *testing.T, svc *ContactMoveService, userID, vaultID string) models.UserVault {
	t.Helper()
	var userVault models.UserVault
	if err := svc.db.First(&userVault, "user_id = ? AND vault_id = ?", userID, vaultID).Error; err != nil {
		t.Fatalf("Load user vault failed: %v", err)
	}
	return userVault
}
