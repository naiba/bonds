package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupJournalTest(t *testing.T) (*JournalService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "journal-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return NewJournalService(db), vault.ID
}

func TestCreateJournal(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	journal, err := svc.Create(vaultID, dto.CreateJournalRequest{
		Name:        "My Journal",
		Description: "A daily journal",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if journal.Name != "My Journal" {
		t.Errorf("Expected name 'My Journal', got '%s'", journal.Name)
	}
	if journal.Description != "A daily journal" {
		t.Errorf("Expected description 'A daily journal', got '%s'", journal.Description)
	}
	if journal.VaultID != vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", vaultID, journal.VaultID)
	}
	if journal.PostCount != 0 {
		t.Errorf("Expected post_count 0, got %d", journal.PostCount)
	}
	if journal.ID == 0 {
		t.Error("Expected journal ID to be non-zero")
	}
}

func TestListJournals(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	_, err := svc.Create(vaultID, dto.CreateJournalRequest{Name: "Journal 1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(vaultID, dto.CreateJournalRequest{Name: "Journal 2"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	journals, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(journals) != 2 {
		t.Errorf("Expected 2 journals, got %d", len(journals))
	}
}

func TestGetJournal(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	created, err := svc.Create(vaultID, dto.CreateJournalRequest{Name: "Get Me"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.Get(created.ID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Get Me" {
		t.Errorf("Expected name 'Get Me', got '%s'", got.Name)
	}
	if got.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestUpdateJournal(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	created, err := svc.Create(vaultID, dto.CreateJournalRequest{Name: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, vaultID, dto.UpdateJournalRequest{
		Name:        "Updated",
		Description: "New description",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", updated.Name)
	}
	if updated.Description != "New description" {
		t.Errorf("Expected description 'New description', got '%s'", updated.Description)
	}
}

func TestDeleteJournal(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	created, err := svc.Create(vaultID, dto.CreateJournalRequest{Name: "To delete"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	journals, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(journals) != 0 {
		t.Errorf("Expected 0 journals after delete, got %d", len(journals))
	}
}

func TestJournalNotFound(t *testing.T) {
	svc, vaultID := setupJournalTest(t)

	_, err := svc.Get(9999, vaultID)
	if err != ErrJournalNotFound {
		t.Errorf("Expected ErrJournalNotFound, got %v", err)
	}

	_, err = svc.Update(9999, vaultID, dto.UpdateJournalRequest{Name: "nope"})
	if err != ErrJournalNotFound {
		t.Errorf("Expected ErrJournalNotFound, got %v", err)
	}

	err = svc.Delete(9999, vaultID)
	if err != ErrJournalNotFound {
		t.Errorf("Expected ErrJournalNotFound, got %v", err)
	}
}
