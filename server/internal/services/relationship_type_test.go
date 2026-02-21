package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupRelationshipTypeTest(t *testing.T) (*RelationshipTypeService, *gorm.DB, string, uint) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "rt-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get an existing relationship group type (seeded by SeedAccountDefaults)
	var gt models.RelationshipGroupType
	if err := db.Where("account_id = ?", resp.User.AccountID).First(&gt).Error; err != nil {
		t.Fatalf("Failed to find seeded RelationshipGroupType: %v", err)
	}

	return NewRelationshipTypeService(db), db, resp.User.AccountID, gt.ID
}

func TestCreateRelationshipType_Success(t *testing.T) {
	svc, _, accountID, groupTypeID := setupRelationshipTypeTest(t)

	created, err := svc.Create(accountID, groupTypeID, dto.CreateRelationshipTypeRequest{
		Name:                    "mentor",
		NameReverseRelationship: "mentee",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Name != "mentor" {
		t.Errorf("Expected name 'mentor', got '%s'", created.Name)
	}
	if created.NameReverseRelationship != "mentee" {
		t.Errorf("Expected reverse name 'mentee', got '%s'", created.NameReverseRelationship)
	}
	if created.RelationshipGroupTypeID != groupTypeID {
		t.Errorf("Expected group_type_id %d, got %d", groupTypeID, created.RelationshipGroupTypeID)
	}
	if !created.CanBeDeleted {
		t.Error("Expected can_be_deleted to be true for custom type")
	}
}

func TestCreateRelationshipType_GroupNotFound(t *testing.T) {
	svc, _, accountID, _ := setupRelationshipTypeTest(t)

	_, err := svc.Create(accountID, 9999, dto.CreateRelationshipTypeRequest{
		Name:                    "test",
		NameReverseRelationship: "test-reverse",
	})
	if err != ErrRelationshipGroupTypeNotFound {
		t.Errorf("Expected ErrRelationshipGroupTypeNotFound, got %v", err)
	}
}

func TestUpdateRelationshipType_Success(t *testing.T) {
	svc, _, accountID, groupTypeID := setupRelationshipTypeTest(t)

	created, err := svc.Create(accountID, groupTypeID, dto.CreateRelationshipTypeRequest{
		Name:                    "old-name",
		NameReverseRelationship: "old-reverse",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(accountID, groupTypeID, created.ID, dto.UpdateRelationshipTypeRequest{
		Name:                    "new-name",
		NameReverseRelationship: "new-reverse",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("Expected name 'new-name', got '%s'", updated.Name)
	}
	if updated.NameReverseRelationship != "new-reverse" {
		t.Errorf("Expected reverse name 'new-reverse', got '%s'", updated.NameReverseRelationship)
	}
}

func TestUpdateRelationshipType_NotFound(t *testing.T) {
	svc, _, accountID, groupTypeID := setupRelationshipTypeTest(t)

	_, err := svc.Update(accountID, groupTypeID, 9999, dto.UpdateRelationshipTypeRequest{
		Name:                    "nope",
		NameReverseRelationship: "nope-reverse",
	})
	if err != ErrRelationshipTypeNotFound {
		t.Errorf("Expected ErrRelationshipTypeNotFound, got %v", err)
	}
}

func TestDeleteRelationshipType_Success(t *testing.T) {
	svc, _, accountID, groupTypeID := setupRelationshipTypeTest(t)

	created, err := svc.Create(accountID, groupTypeID, dto.CreateRelationshipTypeRequest{
		Name:                    "to-delete",
		NameReverseRelationship: "to-delete-reverse",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(accountID, groupTypeID, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestDeleteRelationshipType_CannotDelete(t *testing.T) {
	svc, db, accountID, groupTypeID := setupRelationshipTypeTest(t)

	// Seeded types have can_be_deleted=false. Find one under this group.
	var rt models.RelationshipType
	if err := db.Where("relationship_group_type_id = ? AND can_be_deleted = ?", groupTypeID, false).First(&rt).Error; err != nil {
		t.Fatalf("Failed to find undeletable seeded RelationshipType: %v", err)
	}

	err := svc.Delete(accountID, groupTypeID, rt.ID)
	if err != ErrRelationshipTypeCannotBeDeleted {
		t.Errorf("Expected ErrRelationshipTypeCannotBeDeleted, got %v", err)
	}
}

func TestDeleteRelationshipType_NotFound(t *testing.T) {
	svc, _, accountID, groupTypeID := setupRelationshipTypeTest(t)

	err := svc.Delete(accountID, groupTypeID, 9999)
	if err != ErrRelationshipTypeNotFound {
		t.Errorf("Expected ErrRelationshipTypeNotFound, got %v", err)
	}
}
