package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactLabelFilterTest(t *testing.T) (*ContactService, string, string, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	contactSvc := NewContactService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "clf-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	label := models.Label{VaultID: vault.ID, Name: "Family", Slug: "family"}
	if err := db.Create(&label).Error; err != nil {
		t.Fatalf("Create label failed: %v", err)
	}

	cl := models.ContactLabel{LabelID: label.ID, ContactID: contact.ID}
	if err := db.Create(&cl).Error; err != nil {
		t.Fatalf("Create contact_label failed: %v", err)
	}

	return contactSvc, vault.ID, resp.User.ID, contact.ID, label.ID
}

func TestListContactsByLabel_Success(t *testing.T) {
	svc, vaultID, userID, contactID, labelID := setupContactLabelFilterTest(t)

	contacts, meta, err := svc.ListContactsByLabel(vaultID, userID, labelID, 1, 15, "", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel failed: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].ID != contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", contactID, contacts[0].ID)
	}
	if meta.Total != 1 {
		t.Errorf("Expected total 1, got %d", meta.Total)
	}
}

func TestListContactsByLabel_NoResults(t *testing.T) {
	svc, vaultID, userID, _, _ := setupContactLabelFilterTest(t)

	contacts, meta, err := svc.ListContactsByLabel(vaultID, userID, 9999, 1, 15, "", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(contacts))
	}
	if meta.Total != 0 {
		t.Errorf("Expected total 0, got %d", meta.Total)
	}
}

func TestListContactsByLabel_Pagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)
	contactSvc := NewContactService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "clf-pag@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	label := models.Label{VaultID: vault.ID, Name: "Friends", Slug: "friends"}
	if err := db.Create(&label).Error; err != nil {
		t.Fatalf("Create label failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		names := []string{"Alice", "Bob", "Charlie"}
		c, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: names[i]})
		if err != nil {
			t.Fatalf("CreateContact failed: %v", err)
		}
		cl := models.ContactLabel{LabelID: label.ID, ContactID: c.ID}
		if err := db.Create(&cl).Error; err != nil {
			t.Fatalf("Create contact_label failed: %v", err)
		}
	}

	page1, meta1, err := contactSvc.ListContactsByLabel(vault.ID, resp.User.ID, label.ID, 1, 2, "", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel page 1 failed: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Expected 2 contacts on page 1, got %d", len(page1))
	}
	if meta1.Total != 3 {
		t.Errorf("Expected total 3, got %d", meta1.Total)
	}
	if meta1.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", meta1.TotalPages)
	}

	page2, _, err := contactSvc.ListContactsByLabel(vault.ID, resp.User.ID, label.ID, 2, 2, "", "")
	if err != nil {
		t.Fatalf("ListContactsByLabel page 2 failed: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("Expected 1 contact on page 2, got %d", len(page2))
	}
}
