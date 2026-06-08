package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupGroupTypeRoleTest(t *testing.T) (*GroupTypeRoleService, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "gtr-test@example.com", Password: "password123",
	}, "en")

	var gt models.GroupType
	db.Where("account_id = ?", resp.User.AccountID).First(&gt)

	return NewGroupTypeRoleService(db), resp.User.AccountID, gt.ID
}

func TestCreateGroupTypeRole(t *testing.T) {
	svc, accountID, groupTypeID := setupGroupTypeRoleTest(t)

	pos := 1
	role, err := svc.Create(accountID, groupTypeID, dto.CreateGroupTypeRoleRequest{
		Label: "Leader", Position: &pos,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if role.Label != "Leader" {
		t.Errorf("Expected label 'Leader', got '%s'", role.Label)
	}
}

func TestUpdateGroupTypeRole(t *testing.T) {
	svc, accountID, groupTypeID := setupGroupTypeRoleTest(t)

	pos := 1
	created, _ := svc.Create(accountID, groupTypeID, dto.CreateGroupTypeRoleRequest{
		Label: "Old", Position: &pos,
	})
	pos2 := 2
	updated, err := svc.Update(accountID, groupTypeID, created.ID, dto.UpdateGroupTypeRoleRequest{
		Label: "New", Position: &pos2,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Label != "New" {
		t.Errorf("Expected label 'New', got '%s'", updated.Label)
	}
}

func TestDeleteGroupTypeRole(t *testing.T) {
	svc, accountID, groupTypeID := setupGroupTypeRoleTest(t)

	pos := 1
	created, _ := svc.Create(accountID, groupTypeID, dto.CreateGroupTypeRoleRequest{
		Label: "ToDelete", Position: &pos,
	})
	if err := svc.Delete(accountID, groupTypeID, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestGroupTypeRoleNotFound(t *testing.T) {
	svc, accountID, groupTypeID := setupGroupTypeRoleTest(t)

	err := svc.Delete(accountID, groupTypeID, 9999)
	if err != ErrGroupTypeRoleNotFound {
		t.Errorf("Expected ErrGroupTypeRoleNotFound, got %v", err)
	}
}

func TestUpdateGroupTypeRolePositionResequencesSiblings(t *testing.T) {
	svc, accountID, groupTypeID := setupGroupTypeRoleTest(t)

	roles, err := svc.List(accountID, groupTypeID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(roles) < 2 {
		t.Fatalf("expected seeded group type roles, got %d", len(roles))
	}
	moved := roles[1]

	if err := svc.UpdatePosition(accountID, groupTypeID, moved.ID, 0); err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	reordered, err := svc.List(accountID, groupTypeID)
	if err != nil {
		t.Fatalf("List after UpdatePosition failed: %v", err)
	}
	if reordered[0].ID != moved.ID {
		t.Fatalf("first role id = %d, want moved id %d", reordered[0].ID, moved.ID)
	}
	for i, role := range reordered {
		if role.Position == nil {
			t.Fatalf("role %d position is nil", role.ID)
		}
		if *role.Position != i {
			t.Fatalf("role %d position = %d, want %d", role.ID, *role.Position, i)
		}
	}
}
