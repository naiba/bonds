package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactTabsTest(t *testing.T) (*ContactTabService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "tabs-test@example.com",
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

	return NewContactTabService(db), contact.ID, vault.ID, resp.User.AccountID
}

func TestGetTabs_WithTemplate(t *testing.T) {
	svc, contactID, vaultID, accountID := setupContactTabsTest(t)

	var tmpl models.Template
	svc.db.Where("account_id = ? AND can_be_deleted = ?", accountID, false).First(&tmpl)

	svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("template_id", tmpl.ID)

	tabs, err := svc.GetTabs(contactID, vaultID)
	if err != nil {
		t.Fatalf("GetTabs failed: %v", err)
	}

	if tabs.TemplateID != tmpl.ID {
		t.Errorf("Expected template ID %d, got %d", tmpl.ID, tabs.TemplateID)
	}
	if tabs.TemplateName != "Default template" {
		t.Errorf("Expected template name 'Default template', got '%s'", tabs.TemplateName)
	}
	if len(tabs.Pages) != 5 {
		t.Fatalf("Expected 5 pages, got %d", len(tabs.Pages))
	}

	contactPage := tabs.Pages[0]
	if contactPage.Slug != "contact" {
		t.Errorf("Expected first page slug 'contact', got '%s'", contactPage.Slug)
	}
	if len(contactPage.Modules) != 10 {
		t.Errorf("Expected 10 modules on contact page, got %d", len(contactPage.Modules))
	}

	feedPage := tabs.Pages[1]
	if feedPage.Slug != "feed" {
		t.Errorf("Expected second page slug 'feed', got '%s'", feedPage.Slug)
	}
	if len(feedPage.Modules) != 1 {
		t.Errorf("Expected 1 module on feed page, got %d", len(feedPage.Modules))
	}

	socialPage := tabs.Pages[2]
	if socialPage.Slug != "social" {
		t.Errorf("Expected third page slug 'social', got '%s'", socialPage.Slug)
	}
	if len(socialPage.Modules) != 3 {
		t.Errorf("Expected 3 modules on social page, got %d", len(socialPage.Modules))
	}

	lifeGoalsPage := tabs.Pages[3]
	if lifeGoalsPage.Slug != "life-goals" {
		t.Errorf("Expected fourth page slug 'life-goals', got '%s'", lifeGoalsPage.Slug)
	}
	if len(lifeGoalsPage.Modules) != 2 {
		t.Errorf("Expected 2 modules on life-goals page, got %d", len(lifeGoalsPage.Modules))
	}

	infoPage := tabs.Pages[4]
	if infoPage.Slug != "information" {
		t.Errorf("Expected fifth page slug 'information', got '%s'", infoPage.Slug)
	}
	if len(infoPage.Modules) != 8 {
		t.Errorf("Expected 8 modules on information page, got %d", len(infoPage.Modules))
	}
}

func TestGetTabs_WithoutTemplate(t *testing.T) {
	svc, contactID, vaultID, _ := setupContactTabsTest(t)

	tabs, err := svc.GetTabs(contactID, vaultID)
	if err != nil {
		t.Fatalf("GetTabs failed: %v", err)
	}

	if tabs.TemplateName != "Default template" {
		t.Errorf("Expected fallback to 'Default template', got '%s'", tabs.TemplateName)
	}
	if len(tabs.Pages) != 5 {
		t.Fatalf("Expected 5 pages, got %d", len(tabs.Pages))
	}
}

func TestGetTabs_ContactNotFound(t *testing.T) {
	svc, _, vaultID, _ := setupContactTabsTest(t)

	_, err := svc.GetTabs("nonexistent-id", vaultID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestGetTabs_ModuleOrdering(t *testing.T) {
	svc, contactID, vaultID, _ := setupContactTabsTest(t)

	tabs, err := svc.GetTabs(contactID, vaultID)
	if err != nil {
		t.Fatalf("GetTabs failed: %v", err)
	}

	contactPage := tabs.Pages[0]
	for i, mod := range contactPage.Modules {
		expectedPos := i + 1
		if mod.Position != expectedPos {
			t.Errorf("Module %d: expected position %d, got %d", i, expectedPos, mod.Position)
		}
	}

	expectedTypes := []string{"avatar", "contact_names", "family_summary", "important_dates", "gender_pronoun", "labels", "company", "religions", "addresses", "contact_information"}
	for i, mod := range contactPage.Modules {
		if mod.Type != expectedTypes[i] {
			t.Errorf("Module %d: expected type '%s', got '%s'", i, expectedTypes[i], mod.Type)
		}
	}
}
