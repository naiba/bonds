package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupQuickFactTest(t *testing.T) (*QuickFactService, string) {
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

	return NewQuickFactService(db), contact.ID
}

func TestCreateQuickFact(t *testing.T) {
	svc, contactID := setupQuickFactTest(t)

	fact, err := svc.Create(contactID, 1, dto.CreateQuickFactRequest{
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
	svc, contactID := setupQuickFactTest(t)

	_, err := svc.Create(contactID, 1, dto.CreateQuickFactRequest{Content: "Fact 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, 1, dto.CreateQuickFactRequest{Content: "Fact 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	facts, err := svc.List(contactID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("Expected 2 quick facts, got %d", len(facts))
	}
}

func TestUpdateQuickFact(t *testing.T) {
	svc, contactID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, 1, dto.CreateQuickFactRequest{Content: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, dto.UpdateQuickFactRequest{
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
	svc, contactID := setupQuickFactTest(t)

	created, err := svc.Create(contactID, 1, dto.CreateQuickFactRequest{Content: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	facts, err := svc.List(contactID, 1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 quick facts after delete, got %d", len(facts))
	}
}

func TestDeleteQuickFactNotFound(t *testing.T) {
	svc, _ := setupQuickFactTest(t)

	err := svc.Delete(9999)
	if err != ErrQuickFactNotFound {
		t.Errorf("Expected ErrQuickFactNotFound, got %v", err)
	}
}
