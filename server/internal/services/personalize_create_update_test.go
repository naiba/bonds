package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
)

func TestPersonalizeCreate_WorksOnSQLite(t *testing.T) {
	svc, accountID := setupPersonalizeTest(t)

	resp, err := svc.Create(accountID, "genders", dto.PersonalizeEntityRequest{Name: "Custom"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Name != "Custom" {
		t.Errorf("expected name 'Custom', got %q", resp.Name)
	}
	if resp.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be set")
	}
}

func TestPersonalizeUpdate_WorksOnSQLite(t *testing.T) {
	svc, accountID := setupPersonalizeTest(t)

	created, err := svc.Create(accountID, "genders", dto.PersonalizeEntityRequest{Name: "Before"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(accountID, "genders", created.ID, dto.PersonalizeEntityRequest{Name: "After"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "After" {
		t.Errorf("expected name 'After', got %q", updated.Name)
	}
}
