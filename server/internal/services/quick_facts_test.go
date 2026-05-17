package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupQuickFactTest(t *testing.T) (*QuickFactService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "quick-facts-test@example.com",
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

	return NewQuickFactService(db), contact.ID, vault.ID
}

func TestCreateQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	fact, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{
		Content: "Loves coffee",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if fact.Content != "Loves coffee" {
		t.Errorf("Expected content 'Loves coffee', got '%s'", fact.Content)
	}
	if fact.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, fact.ContactID)
	}
	if fact.VaultQuickFactsTemplateID != 1 {
		t.Errorf("Expected vault_quick_facts_template_id 1, got %d", fact.VaultQuickFactsTemplateID)
	}
	if fact.ID == 0 {
		t.Error("Expected quick fact ID to be non-zero")
	}
}

func TestListQuickFacts(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	_, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Fact 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Fact 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	facts, err := svc.List(contactID, vaultID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("Expected 2 quick facts, got %d", len(facts))
	}
}

func TestListAllQuickFactsGroupsFactsByVaultTemplate(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	var templates []models.VaultQuickFactsTemplate
	if err := svc.db.Where("vault_id = ?", vaultID).Order("position ASC").Find(&templates).Error; err != nil {
		t.Fatalf("List templates failed: %v", err)
	}
	if len(templates) < 2 {
		t.Fatalf("Expected at least 2 quick fact templates, got %d", len(templates))
	}

	_, err := svc.Create(contactID, vaultID, templates[0].ID, dto.CreateQuickFactRequest{Content: "Enjoys hiking"})
	if err != nil {
		t.Fatalf("Create first template fact failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, templates[1].ID, dto.CreateQuickFactRequest{Content: "Avoids caffeine"})
	if err != nil {
		t.Fatalf("Create second template fact failed: %v", err)
	}

	groups, err := svc.ListAll(contactID, vaultID)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(groups) != len(templates) {
		t.Fatalf("Expected %d template groups, got %d", len(templates), len(groups))
	}
	if groups[0].TemplateID != templates[0].ID {
		t.Errorf("Expected first template ID %d, got %d", templates[0].ID, groups[0].TemplateID)
	}
	if groups[0].TemplateLabel == "" {
		t.Error("Expected first template label to be populated")
	}
	if len(groups[0].Facts) != 1 || groups[0].Facts[0].Content != "Enjoys hiking" {
		t.Fatalf("Expected first group to contain hiking fact, got %+v", groups[0].Facts)
	}
	if len(groups[1].Facts) != 1 || groups[1].Facts[0].Content != "Avoids caffeine" {
		t.Fatalf("Expected second group to contain caffeine fact, got %+v", groups[1].Facts)
	}
}

func TestUpdateQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateQuickFactRequest{
		Content: "Updated content",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", updated.Content)
	}
}

func TestDeleteQuickFact(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, vaultID, 1, dto.CreateQuickFactRequest{Content: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	facts, err := svc.List(contactID, vaultID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 quick facts after delete, got %d", len(facts))
	}
}

func TestDeleteQuickFactNotFound(t *testing.T) {
	svc, contactID, vaultID := setupQuickFactTest(t)

	err := svc.Delete(9999, contactID, vaultID)
	if err != ErrQuickFactNotFound {
		t.Errorf("Expected ErrQuickFactNotFound, got %v", err)
	}
}
