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

func TestFindByIdentityExactMatch(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	if _, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1, Data: "alice@example.com",
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	matches, err := svc.FindByIdentity(vaultID, "alice@example.com", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].ContactID != contactID {
		t.Errorf("expected contactID %s, got %s", contactID, matches[0].ContactID)
	}
	if matches[0].ContactInformation.Data != "alice@example.com" {
		t.Errorf("data mismatch: %q", matches[0].ContactInformation.Data)
	}
	if matches[0].ContactFirstName != "John" {
		t.Errorf("expected first name John, got %q", matches[0].ContactFirstName)
	}
}

func TestFindByIdentityCaseInsensitive(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{
		TypeID: 1, Data: "Alice@Example.COM",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	matches, err := svc.FindByIdentity(vaultID, "alice@example.com", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected case-insensitive match, got %d", len(matches))
	}
}

func TestFindByIdentityNoMatch(t *testing.T) {
	svc, _, vaultID := setupContactInformationTest(t)

	matches, err := svc.FindByIdentity(vaultID, "nobody@example.com", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestFindByIdentityFiltersByType(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)

	_, _ = svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{TypeID: 1, Data: "shared"})
	_, _ = svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{TypeID: 2, Data: "shared"})

	all, err := svc.FindByIdentity(vaultID, "shared", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("without type filter expected 2, got %d", len(all))
	}

	onlyOne, err := svc.FindByIdentity(vaultID, "shared", 1)
	if err != nil {
		t.Fatalf("FindByIdentity typeID=1: %v", err)
	}
	if len(onlyOne) != 1 {
		t.Fatalf("with type filter expected 1, got %d", len(onlyOne))
	}
	if onlyOne[0].ContactInformation.TypeID != 1 {
		t.Errorf("type filter did not apply, got TypeID=%d", onlyOne[0].ContactInformation.TypeID)
	}
}

func TestFindByIdentityScopedToVault(t *testing.T) {
	svc, contactID, vaultID := setupContactInformationTest(t)
	_, _ = svc.Create(contactID, vaultID, dto.CreateContactInformationRequest{TypeID: 1, Data: "vault-1@example.com"})

	matches, err := svc.FindByIdentity("non-existent-vault", "vault-1@example.com", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("must not leak across vaults, got %d matches", len(matches))
	}
}

func TestFindByIdentityEmptyParams(t *testing.T) {
	svc, _, vaultID := setupContactInformationTest(t)

	if got, _ := svc.FindByIdentity("", "x", 0); len(got) != 0 {
		t.Error("empty vault must yield no matches")
	}
	if got, _ := svc.FindByIdentity(vaultID, "", 0); len(got) != 0 {
		t.Error("empty data must yield no matches")
	}
}

func TestFindByIdentityMultipleContacts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email:    "find-by-id-multi@example.com",
		Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "v"}, "en")
	contactSvc := NewContactService(db)
	c1, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Alice"})
	c2, _ := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Bob"})

	svc := NewContactInformationService(db)
	_, _ = svc.Create(c1.ID, vault.ID, dto.CreateContactInformationRequest{TypeID: 1, Data: "shared@example.com"})
	_, _ = svc.Create(c2.ID, vault.ID, dto.CreateContactInformationRequest{TypeID: 1, Data: "shared@example.com"})

	matches, err := svc.FindByIdentity(vault.ID, "shared@example.com", 0)
	if err != nil {
		t.Fatalf("FindByIdentity: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected matches across both contacts, got %d", len(matches))
	}
}
