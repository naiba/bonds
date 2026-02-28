package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type relationshipTestCtx struct {
	svc              *RelationshipService
	db               *gorm.DB
	contactID        string
	relatedContactID string
	vaultID          string
	accountID        string
}

func setupRelationshipTest(t *testing.T) (*RelationshipService, string, string, string) {
	t.Helper()
	ctx := setupRelationshipTestFull(t)
	return ctx.svc, ctx.contactID, ctx.relatedContactID, ctx.vaultID
}

func setupRelationshipTestFull(t *testing.T) relationshipTestCtx {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "relationship-test@example.com",
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

	relatedContact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "Jane"})
	if err != nil {
		t.Fatalf("CreateContact (related) failed: %v", err)
	}

	return relationshipTestCtx{
		svc:              NewRelationshipService(db),
		db:               db,
		contactID:        contact.ID,
		relatedContactID: relatedContact.ID,
		vaultID:          vault.ID,
		accountID:        resp.User.AccountID,
	}
}

type graphTestEnv struct {
	svc     *RelationshipService
	db      *gorm.DB
	vaultID string
	userID  string
}

func setupGraphTest(t *testing.T) (*graphTestEnv, func(firstName string) string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "graph-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Graph Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	createContact := func(firstName string) string {
		c, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: firstName})
		if err != nil {
			t.Fatalf("CreateContact(%s) failed: %v", firstName, err)
		}
		return c.ID
	}

	env := &graphTestEnv{
		svc:     NewRelationshipService(db),
		db:      db,
		vaultID: vault.ID,
		userID:  resp.User.ID,
	}
	return env, createContact
}

func findRelTypeByDegree(t *testing.T, db *gorm.DB, degree int) uint {
	t.Helper()
	var rt models.RelationshipType
	if err := db.Where("degree = ?", degree).First(&rt).Error; err != nil {
		t.Fatalf("no relationship type with degree %d found: %v", degree, err)
	}
	return rt.ID
}

func findRelTypeNilDegree(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	var rt models.RelationshipType
	if err := db.Where("degree IS NULL").First(&rt).Error; err != nil {
		t.Fatalf("no relationship type with nil degree found: %v", err)
	}
	return rt.ID
}

func TestCreateRelationship(t *testing.T) {
	svc, contactID, relatedContactID, vaultID := setupRelationshipTest(t)

	rel, err := svc.Create(contactID, vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if rel.ContactID != contactID {
		t.Errorf("Expected contact_id '%s', got '%s'", contactID, rel.ContactID)
	}
	if rel.RelatedContactID != relatedContactID {
		t.Errorf("Expected related_contact_id '%s', got '%s'", relatedContactID, rel.RelatedContactID)
	}
	if rel.RelationshipTypeID != 1 {
		t.Errorf("Expected relationship_type_id 1, got %d", rel.RelationshipTypeID)
	}
	if rel.ID == 0 {
		t.Error("Expected relationship ID to be non-zero")
	}
}

func TestListRelationships(t *testing.T) {
	svc, contactID, relatedContactID, vaultID := setupRelationshipTest(t)

	_, err := svc.Create(contactID, vaultID, dto.CreateRelationshipRequest{RelationshipTypeID: 1, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, dto.CreateRelationshipRequest{RelationshipTypeID: 2, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// List now returns bidirectional results: 2 forward (John->Jane) + 2 reverse auto-created (Jane->John),
	// all visible from John's page because query uses contact_id=? OR related_contact_id=?.
	relationships, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(relationships) != 4 {
		t.Errorf("Expected 4 relationships (2 forward + 2 reverse), got %d", len(relationships))
	}
	// All results should show relatedContactID as the "other" person
	for _, r := range relationships {
		if r.RelatedContactID != relatedContactID {
			t.Errorf("Expected related_contact_id to be '%s' (the other person), got '%s'", relatedContactID, r.RelatedContactID)
		}
	}
}

func TestUpdateRelationship(t *testing.T) {
	svc, contactID, relatedContactID, vaultID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateRelationshipRequest{
		RelationshipTypeID: 3,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.RelationshipTypeID != 3 {
		t.Errorf("Expected relationship_type_id 3, got %d", updated.RelationshipTypeID)
	}
}

func TestDeleteRelationship(t *testing.T) {
	svc, contactID, relatedContactID, vaultID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID, contactID, vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	relationships, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(relationships) != 0 {
		t.Errorf("Expected 0 relationships after delete, got %d", len(relationships))
	}
}

func TestRelationshipNotFound(t *testing.T) {
	svc, contactID, relatedContactID, vaultID := setupRelationshipTest(t)

	_, err := svc.Update(9999, contactID, vaultID, dto.UpdateRelationshipRequest{
		RelationshipTypeID: 1,
		RelatedContactID:   relatedContactID,
	})
	if err != ErrRelationshipNotFound {
		t.Errorf("Expected ErrRelationshipNotFound, got %v", err)
	}

	err = svc.Delete(9999, contactID, vaultID)
	if err != ErrRelationshipNotFound {
		t.Errorf("Expected ErrRelationshipNotFound, got %v", err)
	}
}

func createAsymmetricTypePair(t *testing.T, db *gorm.DB, accountID string) (parentTypeID, childTypeID uint) {
	t.Helper()
	var group models.RelationshipGroupType
	if err := db.Where("account_id = ?", accountID).First(&group).Error; err != nil {
		t.Fatalf("Failed to find group: %v", err)
	}
	parentName := "test-parent"
	childName := "test-child"
	parentType := models.RelationshipType{
		RelationshipGroupTypeID: group.ID,
		Name:                    &parentName,
		NameReverseRelationship: &childName,
		CanBeDeleted:            true,
	}
	if err := db.Create(&parentType).Error; err != nil {
		t.Fatalf("Create parent type failed: %v", err)
	}
	childType := models.RelationshipType{
		RelationshipGroupTypeID: group.ID,
		Name:                    &childName,
		NameReverseRelationship: &parentName,
		CanBeDeleted:            true,
	}
	if err := db.Create(&childType).Error; err != nil {
		t.Fatalf("Create child type failed: %v", err)
	}
	return parentType.ID, childType.ID
}

func TestCreateRelationship_AutoCreatesReverse(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, childTypeID := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	found := false
	for _, r := range reverseRels {
		if r.ContactID == ctx.relatedContactID && r.RelatedContactID == ctx.contactID && r.RelationshipTypeID == childTypeID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected reverse relationship (child type %d) on related contact, not found in %d relationships", childTypeID, len(reverseRels))
	}
}

func TestCreateRelationship_SymmetricType(t *testing.T) {
	ctx := setupRelationshipTestFull(t)

	var spouseType models.RelationshipType
	if err := ctx.db.Where("name = name_reverse_relationship AND name IS NOT NULL").First(&spouseType).Error; err != nil {
		t.Fatalf("Failed to find symmetric type: %v", err)
	}

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: spouseType.ID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	found := false
	for _, r := range reverseRels {
		if r.ContactID == ctx.relatedContactID && r.RelatedContactID == ctx.contactID && r.RelationshipTypeID == spouseType.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected symmetric reverse relationship (type %d) on related contact, not found", spouseType.ID)
	}
}

func TestCreateRelationship_NoReverseType(t *testing.T) {
	ctx := setupRelationshipTestFull(t)

	var group models.RelationshipGroupType
	if err := ctx.db.Where("account_id = ?", ctx.accountID).First(&group).Error; err != nil {
		t.Fatalf("Failed to find group: %v", err)
	}
	name := "orphan-type"
	reverseName := "nonexistent-reverse"
	orphanType := models.RelationshipType{
		RelationshipGroupTypeID: group.ID,
		Name:                    &name,
		NameReverseRelationship: &reverseName,
		CanBeDeleted:            true,
	}
	if err := ctx.db.Create(&orphanType).Error; err != nil {
		t.Fatalf("Create orphan type failed: %v", err)
	}

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: orphanType.ID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// List(contactID) returns 1 forward record (no reverse was auto-created)
	forwardRels, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List forward failed: %v", err)
	}
	if len(forwardRels) != 1 {
		t.Errorf("Expected 1 forward relationship, got %d", len(forwardRels))
	}

	// List(relatedContactID) now returns 1 because the bidirectional query also
	// finds the forward record (related_contact_id = relatedContactID). No reverse
	// record was auto-created since the reverse type doesn't exist.
	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 1 {
		t.Errorf("Expected 1 relationship visible from related contact (forward record seen via bidirectional query), got %d", len(reverseRels))
	}
}

func TestDeleteRelationship_AutoDeletesReverse(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, _ := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Before delete: related contact sees 2 records (1 forward via bidirectional + 1 reverse auto-created)
	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 2 {
		t.Fatalf("Expected 2 relationships visible from related contact before delete, got %d", len(reverseRels))
	}

	if err := ctx.svc.Delete(created.ID, ctx.contactID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	forwardRels, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List forward failed: %v", err)
	}
	if len(forwardRels) != 0 {
		t.Errorf("Expected 0 forward relationships after delete, got %d", len(forwardRels))
	}

	reverseRels, err = ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse after delete failed: %v", err)
	}
	if len(reverseRels) != 0 {
		t.Errorf("Expected 0 reverse relationships after delete, got %d", len(reverseRels))
	}
}

func TestDeleteRelationship_ReverseAlreadyGone(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, _ := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Manually delete the reverse record from DB directly (not via service)
	ctx.db.Where("contact_id = ? AND related_contact_id = ?", ctx.relatedContactID, ctx.contactID).Delete(&models.Relationship{})

	// Delete the forward record should succeed even though reverse is already gone (tolerant delete)
	if err := ctx.svc.Delete(created.ID, ctx.contactID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed (reverse already gone): %v", err)
	}
}

func TestUpdateRelationship_NotBidirectional(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, childTypeID := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Related contact sees 2 records: 1 forward (via bidirectional) + 1 reverse auto-created
	reverseRelsBefore, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRelsBefore) != 2 {
		t.Fatalf("Expected 2 relationships visible from related contact, got %d", len(reverseRelsBefore))
	}

	var spouseType models.RelationshipType
	if err := ctx.db.Where("name = name_reverse_relationship AND name IS NOT NULL").First(&spouseType).Error; err != nil {
		t.Fatalf("Failed to find symmetric type: %v", err)
	}
	_, err = ctx.svc.Update(created.ID, ctx.contactID, ctx.vaultID, dto.UpdateRelationshipRequest{
		RelationshipTypeID: spouseType.ID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// After updating forward to spouse, related contact still sees 2 records:
	// 1 forward (now spouse, via bidirectional) + 1 reverse (still child, update is not bidirectional)
	reverseRelsAfter, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse after update failed: %v", err)
	}
	if len(reverseRelsAfter) != 2 {
		t.Fatalf("Expected 2 relationships after update, got %d", len(reverseRelsAfter))
	}
	// The reverse auto-created record should still be child type (update doesn't touch reverse)
	foundChild := false
	for _, r := range reverseRelsAfter {
		if r.RelationshipTypeID == childTypeID {
			foundChild = true
			break
		}
	}
	if !foundChild {
		t.Errorf("Expected reverse relationship to still be child type %d", childTypeID)
	}
}
