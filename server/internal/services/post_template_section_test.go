package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupPostTemplateSectionTest(t *testing.T) (*PostTemplateSectionService, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "pts-test@example.com", Password: "password123",
	}, "en")

	var tpl models.PostTemplate
	db.Where("account_id = ?", resp.User.AccountID).First(&tpl)

	return NewPostTemplateSectionService(db), resp.User.AccountID, tpl.ID
}

func TestCreatePostTemplateSection(t *testing.T) {
	svc, accountID, templateID := setupPostTemplateSectionTest(t)

	section, err := svc.Create(accountID, templateID, dto.CreatePostTemplateSectionRequest{
		Label: "Thoughts", Position: 10,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if section.Label != "Thoughts" {
		t.Errorf("Expected label 'Thoughts', got '%s'", section.Label)
	}
	if section.Position != 10 {
		t.Errorf("Expected position 10, got %d", section.Position)
	}
}

func TestUpdatePostTemplateSection(t *testing.T) {
	svc, accountID, templateID := setupPostTemplateSectionTest(t)

	created, _ := svc.Create(accountID, templateID, dto.CreatePostTemplateSectionRequest{
		Label: "Old", Position: 1,
	})
	updated, err := svc.Update(accountID, templateID, created.ID, dto.UpdatePostTemplateSectionRequest{
		Label: "New", Position: 2,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "New" {
		t.Errorf("Expected label 'New', got '%s'", updated.Label)
	}
}

func TestDeletePostTemplateSection(t *testing.T) {
	svc, accountID, templateID := setupPostTemplateSectionTest(t)

	created, _ := svc.Create(accountID, templateID, dto.CreatePostTemplateSectionRequest{
		Label: "ToDelete", Position: 1,
	})
	if err := svc.Delete(accountID, templateID, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestPostTemplateSectionNotFound(t *testing.T) {
	svc, accountID, templateID := setupPostTemplateSectionTest(t)

	err := svc.Delete(accountID, templateID, 9999)
	if err != ErrPostTemplateSectionNotFound {
		t.Errorf("Expected ErrPostTemplateSectionNotFound, got %v", err)
	}
}

func TestUpdatePostTemplateSectionPosition(t *testing.T) {
	svc, accountID, templateID := setupPostTemplateSectionTest(t)

	created, _ := svc.Create(accountID, templateID, dto.CreatePostTemplateSectionRequest{
		Label: "Test", Position: 1,
	})
	if err := svc.UpdatePosition(accountID, templateID, created.ID, 5); err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
}
