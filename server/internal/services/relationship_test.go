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
	userID           string
}

func setupRelationshipTest(t *testing.T) (*RelationshipService, string, string, string, string) {
	t.Helper()
	ctx := setupRelationshipTestFull(t)
	return ctx.svc, ctx.contactID, ctx.relatedContactID, ctx.vaultID, ctx.userID
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
		userID:           resp.User.ID,
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
	svc, contactID, relatedContactID, vaultID, userID := setupRelationshipTest(t)

	rel, err := svc.Create(contactID, vaultID, userID, dto.CreateRelationshipRequest{
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
	svc, contactID, relatedContactID, vaultID, userID := setupRelationshipTest(t)

	_, err := svc.Create(contactID, vaultID, userID, dto.CreateRelationshipRequest{RelationshipTypeID: 1, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = svc.Create(contactID, vaultID, userID, dto.CreateRelationshipRequest{RelationshipTypeID: 2, RelatedContactID: relatedContactID})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// BUG FIX: List now uses simple contact_id query (no bidirectional).
	// Each contact only sees records they "own" (contact_id = their ID).
	// John created 2 forward records; auto-reverse created 2 records for Jane.
	// From John's page, we see exactly 2 (his owned records).
	relationships, err := svc.List(contactID, vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(relationships) != 2 {
		t.Errorf("Expected 2 relationships (John's owned records), got %d", len(relationships))
	}
	for _, r := range relationships {
		if r.RelatedContactID != relatedContactID {
			t.Errorf("Expected related_contact_id to be '%s' (the other person), got '%s'", relatedContactID, r.RelatedContactID)
		}
	}
}

func TestUpdateRelationship(t *testing.T) {
	svc, contactID, relatedContactID, vaultID, userID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateRelationshipRequest{
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
	svc, contactID, relatedContactID, vaultID, userID := setupRelationshipTest(t)

	created, err := svc.Create(contactID, vaultID, userID, dto.CreateRelationshipRequest{
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
	svc, contactID, relatedContactID, vaultID, _ := setupRelationshipTest(t)

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
		RelationshipGroupTypeID:  group.ID,
		Name:                     &childName,
		NameReverseRelationship:  &parentName,
		CanBeDeleted:             true,
		ReverseRelationshipTypeID: &parentType.ID,
	}
	if err := db.Create(&childType).Error; err != nil {
		t.Fatalf("Create child type failed: %v", err)
	}
	// Link parent → child bidirectionally
	if err := db.Model(&parentType).Update("reverse_relationship_type_id", childType.ID).Error; err != nil {
		t.Fatalf("Link parent to child failed: %v", err)
	}
	return parentType.ID, childType.ID
}

func TestCreateRelationship_AutoCreatesReverse(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, childTypeID := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
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

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
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

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
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

	// BUG FIX: List now uses simple contact_id query. The orphan type has no
	// matching reverse, so no auto-reverse record was created for relatedContact.
	// From relatedContact's page, no records have contact_id = relatedContactID.
	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 0 {
		t.Errorf("Expected 0 relationships from related contact (no auto-reverse created for orphan type), got %d", len(reverseRels))
	}
}

func TestDeleteRelationship_AutoDeletesReverse(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, _ := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Before delete: related contact has 1 auto-reverse record (contact_id = relatedContact)
	reverseRels, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 1 {
		t.Fatalf("Expected 1 relationship visible from related contact before delete, got %d", len(reverseRels))
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

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
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

	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   ctx.relatedContactID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Related contact has 1 auto-reverse record (the child type)
	reverseRelsBefore, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRelsBefore) != 1 {
		t.Fatalf("Expected 1 relationship visible from related contact, got %d", len(reverseRelsBefore))
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

	// After updating forward to spouse, related contact still sees 1 record:
	// the original auto-reverse (child type, update is not bidirectional)
	reverseRelsAfter, err := ctx.svc.List(ctx.relatedContactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List reverse after update failed: %v", err)
	}
	if len(reverseRelsAfter) != 1 {
		t.Fatalf("Expected 1 relationship after update, got %d", len(reverseRelsAfter))
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

// TestCrossVaultRelationship_WithEditorPermission tests that auto-reverse IS created
// when the user has Editor permission on the related contact's vault.
func TestCrossVaultRelationship_WithEditorPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Cross",
		LastName:  "Vault",
		Email:     "crossvault-editor@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	userID := resp.User.ID
	accountID := resp.User.AccountID

	vault1, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault One"}, "en")
	if err != nil {
		t.Fatalf("CreateVault1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault Two"}, "en")
	if err != nil {
		t.Fatalf("CreateVault2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact1, err := contactSvc.CreateContact(vault1.ID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact1 failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault2.ID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact2 failed: %v", err)
	}

	// User is Manager (100) on both vaults via CreateVault, so hasEditorPermission = true for both.
	parentTypeID, childTypeID := createAsymmetricTypePair(t, db, accountID)

	relSvc := NewRelationshipService(db)
	_, err = relSvc.Create(contact1.ID, vault1.ID, userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   contact2.ID,
	})
	if err != nil {
		t.Fatalf("Create cross-vault relationship failed: %v", err)
	}

	// Verify forward exists on contact1
	forwardRels, err := relSvc.List(contact1.ID, vault1.ID)
	if err != nil {
		t.Fatalf("List forward failed: %v", err)
	}
	if len(forwardRels) != 1 {
		t.Errorf("Expected 1 forward relationship, got %d", len(forwardRels))
	}

	// Verify auto-reverse exists on contact2 (user has Editor on vault2)
	reverseRels, err := relSvc.List(contact2.ID, vault2.ID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 1 {
		t.Fatalf("Expected 1 auto-reverse relationship on cross-vault contact, got %d", len(reverseRels))
	}
	if reverseRels[0].RelationshipTypeID != childTypeID {
		t.Errorf("Expected reverse type %d, got %d", childTypeID, reverseRels[0].RelationshipTypeID)
	}
	if reverseRels[0].RelatedContactID != contact1.ID {
		t.Errorf("Expected reverse to point back to contact1, got %s", reverseRels[0].RelatedContactID)
	}
}

// TestCrossVaultRelationship_WithoutEditorPermission tests that auto-reverse is NOT created
// when the user lacks Editor permission on the related contact's vault.
func TestCrossVaultRelationship_WithoutEditorPermission(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	// user1 = vault owner (Manager on both vaults)
	resp1, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Owner",
		LastName:  "User",
		Email:     "crossvault-owner@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user1 failed: %v", err)
	}
	user1ID := resp1.User.ID
	account1ID := resp1.User.AccountID

	vault1, err := vaultSvc.CreateVault(account1ID, user1ID, dto.CreateVaultRequest{Name: "Vault One"}, "en")
	if err != nil {
		t.Fatalf("CreateVault1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(account1ID, user1ID, dto.CreateVaultRequest{Name: "Vault Two"}, "en")
	if err != nil {
		t.Fatalf("CreateVault2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact1, err := contactSvc.CreateContact(vault1.ID, user1ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact1 failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault2.ID, user1ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact2 failed: %v", err)
	}

	// user2 = Editor on vault1, Viewer on vault2
	resp2, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Limited",
		LastName:  "User",
		Email:     "crossvault-limited@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user2 failed: %v", err)
	}
	user2ID := resp2.User.ID

	// Add user2 to vault1 as Editor (200)
	if err := db.Create(&models.UserVault{
		VaultID:    vault1.ID,
		UserID:     user2ID,
		ContactID:  "",
		Permission: models.PermissionEditor,
	}).Error; err != nil {
		t.Fatalf("Add user2 to vault1 failed: %v", err)
	}
	// Add user2 to vault2 as Viewer (300)
	if err := db.Create(&models.UserVault{
		VaultID:    vault2.ID,
		UserID:     user2ID,
		ContactID:  "",
		Permission: models.PermissionViewer,
	}).Error; err != nil {
		t.Fatalf("Add user2 to vault2 failed: %v", err)
	}

	parentTypeID, _ := createAsymmetricTypePair(t, db, account1ID)

	relSvc := NewRelationshipService(db)
	// user2 creates relationship from vault1.contact1 → vault2.contact2
	// user2 has Editor on vault1 (can create forward), but only Viewer on vault2 (no auto-reverse)
	_, err = relSvc.Create(contact1.ID, vault1.ID, user2ID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   contact2.ID,
	})
	if err != nil {
		t.Fatalf("Create cross-vault relationship failed: %v", err)
	}

	// Forward relationship should exist on contact1
	forwardRels, err := relSvc.List(contact1.ID, vault1.ID)
	if err != nil {
		t.Fatalf("List forward failed: %v", err)
	}
	if len(forwardRels) != 1 {
		t.Errorf("Expected 1 forward relationship, got %d", len(forwardRels))
	}

	// Reverse relationship should NOT exist on contact2 (user2 only has Viewer on vault2)
	reverseRels, err := relSvc.List(contact2.ID, vault2.ID)
	if err != nil {
		t.Fatalf("List reverse failed: %v", err)
	}
	if len(reverseRels) != 0 {
		t.Errorf("Expected 0 reverse relationships (user lacks Editor on vault2), got %d", len(reverseRels))
	}
}

// TestCrossVaultRelationship_List tests that List correctly returns cross-vault
// relationship details (related contact name, vault ID, vault name).
func TestCrossVaultRelationship_List(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "List",
		LastName:  "User",
		Email:     "crossvault-list@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	userID := resp.User.ID
	accountID := resp.User.AccountID

	vault1, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Home"}, "en")
	if err != nil {
		t.Fatalf("CreateVault1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Work"}, "en")
	if err != nil {
		t.Fatalf("CreateVault2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact1, err := contactSvc.CreateContact(vault1.ID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact1 failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault2.ID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact2 failed: %v", err)
	}

	parentTypeID, _ := createAsymmetricTypePair(t, db, accountID)

	relSvc := NewRelationshipService(db)
	_, err = relSvc.Create(contact1.ID, vault1.ID, userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   contact2.ID,
	})
	if err != nil {
		t.Fatalf("Create cross-vault relationship failed: %v", err)
	}

	// List relationships for contact1
	rels, err := relSvc.List(contact1.ID, vault1.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(rels))
	}

	// Verify cross-vault details are populated
	rel := rels[0]
	if rel.RelatedContactName != "Bob" {
		t.Errorf("Expected RelatedContactName 'Bob', got '%s'", rel.RelatedContactName)
	}
	if rel.RelatedVaultID != vault2.ID {
		t.Errorf("Expected RelatedVaultID '%s', got '%s'", vault2.ID, rel.RelatedVaultID)
	}
	if rel.RelatedVaultName != "Work" {
		t.Errorf("Expected RelatedVaultName 'Work', got '%s'", rel.RelatedVaultName)
	}
}

// TestListContactsAcrossVaults tests the cross-vault contact listing with permission annotations.
func TestListContactsAcrossVaults(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	// user1 creates vault1 (Manager) and vault2 (Manager)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Across",
		LastName:  "User",
		Email:     "across-vaults@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	user1ID := resp.User.ID
	accountID := resp.User.AccountID

	vault1, err := vaultSvc.CreateVault(accountID, user1ID, dto.CreateVaultRequest{Name: "Vault A"}, "en")
	if err != nil {
		t.Fatalf("CreateVault1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(accountID, user1ID, dto.CreateVaultRequest{Name: "Vault B"}, "en")
	if err != nil {
		t.Fatalf("CreateVault2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	_, err = contactSvc.CreateContact(vault1.ID, user1ID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact in vault1 failed: %v", err)
	}
	_, err = contactSvc.CreateContact(vault2.ID, user1ID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact in vault2 failed: %v", err)
	}

	// user2 has Editor on vault1, Viewer on vault2
	resp2, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Limited",
		LastName:  "Viewer",
		Email:     "across-limited@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register user2 failed: %v", err)
	}
	user2ID := resp2.User.ID

	if err := db.Create(&models.UserVault{
		VaultID:    vault1.ID,
		UserID:     user2ID,
		ContactID:  "",
		Permission: models.PermissionEditor,
	}).Error; err != nil {
		t.Fatalf("Add user2 to vault1 failed: %v", err)
	}
	if err := db.Create(&models.UserVault{
		VaultID:    vault2.ID,
		UserID:     user2ID,
		ContactID:  "",
		Permission: models.PermissionViewer,
	}).Error; err != nil {
		t.Fatalf("Add user2 to vault2 failed: %v", err)
	}

	relSvc := NewRelationshipService(db)
	result, err := relSvc.ListContactsAcrossVaults(user2ID)
	if err != nil {
		t.Fatalf("ListContactsAcrossVaults failed: %v", err)
	}

	// user2 has access to 2 vaults, each with 1 contact
	if len(result) != 2 {
		t.Fatalf("Expected 2 contacts across vaults, got %d", len(result))
	}

	// Build map for easier assertions
	contactsByVault := make(map[string]dto.CrossVaultContactItem)
	for _, c := range result {
		contactsByVault[c.VaultID] = c
	}

	// vault1 contact should have HasEditor=true (Editor permission)
	v1Contact, ok := contactsByVault[vault1.ID]
	if !ok {
		t.Fatal("Expected contact from vault1 in results")
	}
	if !v1Contact.HasEditor {
		t.Error("Expected HasEditor=true for vault1 contact (user2 is Editor)")
	}
	if v1Contact.ContactName != "Alice" {
		t.Errorf("Expected vault1 contact name 'Alice', got '%s'", v1Contact.ContactName)
	}

	// vault2 contact should have HasEditor=false (Viewer permission)
	v2Contact, ok := contactsByVault[vault2.ID]
	if !ok {
		t.Fatal("Expected contact from vault2 in results")
	}
	if v2Contact.HasEditor {
		t.Error("Expected HasEditor=false for vault2 contact (user2 is Viewer)")
	}
	if v2Contact.ContactName != "Bob" {
		t.Errorf("Expected vault2 contact name 'Bob', got '%s'", v2Contact.ContactName)
	}
}

// TestCrossVaultRelationship_DeleteRemovesCrossVaultReverse tests that
// deleting a relationship also deletes the reverse even if cross-vault.
func TestCrossVaultRelationship_DeleteRemovesCrossVaultReverse(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Delete",
		LastName:  "Cross",
		Email:     "crossvault-delete@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	userID := resp.User.ID
	accountID := resp.User.AccountID

	vault1, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault One"}, "en")
	if err != nil {
		t.Fatalf("CreateVault1 failed: %v", err)
	}
	vault2, err := vaultSvc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault Two"}, "en")
	if err != nil {
		t.Fatalf("CreateVault2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact1, err := contactSvc.CreateContact(vault1.ID, userID, dto.CreateContactRequest{FirstName: "Alice"})
	if err != nil {
		t.Fatalf("CreateContact1 failed: %v", err)
	}
	contact2, err := contactSvc.CreateContact(vault2.ID, userID, dto.CreateContactRequest{FirstName: "Bob"})
	if err != nil {
		t.Fatalf("CreateContact2 failed: %v", err)
	}

	parentTypeID, _ := createAsymmetricTypePair(t, db, accountID)

	relSvc := NewRelationshipService(db)
	// User has Editor on both vaults, so auto-reverse will be created
	created, err := relSvc.Create(contact1.ID, vault1.ID, userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: parentTypeID,
		RelatedContactID:   contact2.ID,
	})
	if err != nil {
		t.Fatalf("Create cross-vault relationship failed: %v", err)
	}

	// Verify reverse exists before delete
	reverseRels, err := relSvc.List(contact2.ID, vault2.ID)
	if err != nil {
		t.Fatalf("List reverse before delete failed: %v", err)
	}
	if len(reverseRels) != 1 {
		t.Fatalf("Expected 1 auto-reverse before delete, got %d", len(reverseRels))
	}

	// Delete forward relationship
	if err := relSvc.Delete(created.ID, contact1.ID, vault1.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify forward is gone
	forwardRels, err := relSvc.List(contact1.ID, vault1.ID)
	if err != nil {
		t.Fatalf("List forward after delete failed: %v", err)
	}
	if len(forwardRels) != 0 {
		t.Errorf("Expected 0 forward relationships after delete, got %d", len(forwardRels))
	}

	// Verify reverse is also gone (tolerant delete works cross-vault)
	reverseRels, err = relSvc.List(contact2.ID, vault2.ID)
	if err != nil {
		t.Fatalf("List reverse after delete failed: %v", err)
	}
	if len(reverseRels) != 0 {
		t.Errorf("Expected 0 reverse relationships after delete (cross-vault tolerant delete), got %d", len(reverseRels))
	}
}
