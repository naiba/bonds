package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupContactLabelTest(t *testing.T) (*ContactLabelService, string, string, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "contact-label-test@example.com",
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

	return NewContactLabelService(db), contact.ID, vault.ID, db
}

func createTestLabel(t *testing.T, db *gorm.DB, vaultID string) models.Label {
	t.Helper()
	label := models.Label{
		VaultID:   vaultID,
		Name:      "Test Label",
		Slug:      "test-label",
		BgColor:   "bg-blue-200",
		TextColor: "text-blue-700",
	}
	if err := db.Create(&label).Error; err != nil {
		t.Fatalf("Create label failed: %v", err)
	}
	return label
}

func TestContactLabelAdd(t *testing.T) {
	svc, contactID, vaultID, db := setupContactLabelTest(t)
	label := createTestLabel(t, db, vaultID)

	resp, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: label.ID})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if resp.LabelID != label.ID {
		t.Errorf("Expected label_id %d, got %d", label.ID, resp.LabelID)
	}
	if resp.Name != "Test Label" {
		t.Errorf("Expected name 'Test Label', got '%s'", resp.Name)
	}
	if resp.ID == 0 {
		t.Error("Expected contact label ID to be non-zero")
	}
}

func TestContactLabelList(t *testing.T) {
	svc, contactID, vaultID, db := setupContactLabelTest(t)
	label := createTestLabel(t, db, vaultID)

	_, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: label.ID})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	labels, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}
}

func TestContactLabelRemove(t *testing.T) {
	svc, contactID, vaultID, db := setupContactLabelTest(t)
	label := createTestLabel(t, db, vaultID)

	added, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: label.ID})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := svc.Remove(contactID, vaultID, added.ID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	labels, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("Expected 0 labels after remove, got %d", len(labels))
	}
}

func TestContactLabelRemoveNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupContactLabelTest(t)

	err := svc.Remove(contactID, vaultID, 9999)
	if err != ErrContactLabelNotFound {
		t.Errorf("Expected ErrContactLabelNotFound, got %v", err)
	}
}

func TestContactLabelAddLabelNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupContactLabelTest(t)

	_, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: 9999})
	if err != ErrLabelNotFound {
		t.Errorf("Expected ErrLabelNotFound, got %v", err)
	}
}

func TestContactLabelUpdate(t *testing.T) {
	svc, contactID, vaultID, db := setupContactLabelTest(t)
	label1 := createTestLabel(t, db, vaultID)
	label2 := models.Label{
		VaultID:   vaultID,
		Name:      "Updated Label",
		Slug:      "updated-label",
		BgColor:   "bg-red-200",
		TextColor: "text-red-700",
	}
	if err := db.Create(&label2).Error; err != nil {
		t.Fatalf("Create label2 failed: %v", err)
	}

	added, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: label1.ID})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	updated, err := svc.Update(contactID, vaultID, added.ID, dto.UpdateContactLabelRequest{LabelID: label2.ID})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.LabelID != label2.ID {
		t.Errorf("Expected label_id %d, got %d", label2.ID, updated.LabelID)
	}
	if updated.Name != "Updated Label" {
		t.Errorf("Expected name 'Updated Label', got '%s'", updated.Name)
	}
}

func TestContactLabelUpdateNotFound(t *testing.T) {
	svc, contactID, vaultID, _ := setupContactLabelTest(t)

	_, err := svc.Update(contactID, vaultID, 9999, dto.UpdateContactLabelRequest{LabelID: 1})
	if err != ErrContactLabelNotFound {
		t.Errorf("Expected ErrContactLabelNotFound, got %v", err)
	}
}

func TestContactLabelUpdateLabelNotFound(t *testing.T) {
	svc, contactID, vaultID, db := setupContactLabelTest(t)
	label := createTestLabel(t, db, vaultID)

	added, err := svc.Add(contactID, vaultID, dto.AddContactLabelRequest{LabelID: label.ID})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	_, err = svc.Update(contactID, vaultID, added.ID, dto.UpdateContactLabelRequest{LabelID: 9999})
	if err != ErrLabelNotFound {
		t.Errorf("Expected ErrLabelNotFound, got %v", err)
	}
}
