package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupSliceOfLifeTest(t *testing.T) (*SliceOfLifeService, uint, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "sol-test@example.com", Password: "password123",
	}, "en")
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")

	journalSvc := NewJournalService(db)
	journal, _ := journalSvc.Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})

	return NewSliceOfLifeService(db), journal.ID, vault.ID
}

func TestCreateSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	slice, err := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{
		Name: "Summer 2024", Description: "My summer adventures",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if slice.Name != "Summer 2024" {
		t.Errorf("Expected name 'Summer 2024', got '%s'", slice.Name)
	}
}

func TestListSlicesOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	_, _ = svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Slice 1"})
	_, _ = svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Slice 2"})

	slices, err := svc.List(journalID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(slices) != 2 {
		t.Errorf("Expected 2 slices, got %d", len(slices))
	}
}

func TestGetSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "My Slice"})
	got, err := svc.Get(created.ID, journalID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "My Slice" {
		t.Errorf("Expected name 'My Slice', got '%s'", got.Name)
	}
}

func TestUpdateSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "Old"})
	updated, err := svc.Update(created.ID, journalID, vaultID, dto.UpdateSliceOfLifeRequest{Name: "New"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("Expected name 'New', got '%s'", updated.Name)
	}
}

func TestDeleteSliceOfLife(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	created, _ := svc.Create(journalID, vaultID, dto.CreateSliceOfLifeRequest{Name: "ToDelete"})
	if err := svc.Delete(created.ID, journalID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	slices, _ := svc.List(journalID, vaultID)
	if len(slices) != 0 {
		t.Errorf("Expected 0 slices, got %d", len(slices))
	}
}

func TestDeleteSliceOfLifeUnlinksPosts(t *testing.T) {
	db := testutil.SetupTestDBWithFKConstraints(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "slice-delete-referenced@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	journal, err := NewJournalService(db).Create(vault.ID, dto.CreateJournalRequest{Name: "Test Journal"})
	if err != nil {
		t.Fatalf("CreateJournal failed: %v", err)
	}
	postSvc := NewPostService(db)
	post, err := postSvc.Create(journal.ID, vault.ID, dto.CreatePostRequest{Title: "Test Post", WrittenAt: time.Now()})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	sliceSvc := NewSliceOfLifeService(db)
	slice, err := sliceSvc.Create(journal.ID, vault.ID, dto.CreateSliceOfLifeRequest{Name: "Referenced"})
	if err != nil {
		t.Fatalf("Create slice failed: %v", err)
	}
	if err := postSvc.SetSliceOfLife(post.ID, journal.ID, vault.ID, slice.ID); err != nil {
		t.Fatalf("Set slice failed: %v", err)
	}

	if err := sliceSvc.Delete(slice.ID, journal.ID, vault.ID); err != nil {
		t.Fatalf("Delete referenced slice failed: %v", err)
	}
	var stored models.Post
	if err := db.First(&stored, post.ID).Error; err != nil {
		t.Fatalf("Reload post failed: %v", err)
	}
	if stored.SliceOfLifeID != nil {
		t.Fatalf("Post slice_of_life_id = %v, want nil", *stored.SliceOfLifeID)
	}
}

func TestSliceOfLifeNotFound(t *testing.T) {
	svc, journalID, vaultID := setupSliceOfLifeTest(t)

	_, err := svc.Get(9999, journalID, vaultID)
	if err != ErrSliceOfLifeNotFound {
		t.Errorf("Expected ErrSliceOfLifeNotFound, got %v", err)
	}
}
