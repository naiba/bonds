package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type groupTestCtx struct {
	svc       *GroupService
	vaultID   string
	accountID string
	db        *gorm.DB
}

func setupGroupTest(t *testing.T) groupTestCtx {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "groups-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	return groupTestCtx{
		svc:       NewGroupService(db),
		vaultID:   vault.ID,
		accountID: resp.User.AccountID,
		db:        db,
	}
}

func TestCreateGroup(t *testing.T) {
	ctx := setupGroupTest(t)

	resp, err := ctx.svc.Create(ctx.vaultID, dto.CreateGroupRequest{Name: "Close Friends"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Name != "Close Friends" {
		t.Errorf("Expected name 'Close Friends', got '%s'", resp.Name)
	}
	if resp.VaultID != ctx.vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", ctx.vaultID, resp.VaultID)
	}
	if resp.ID == 0 {
		t.Error("Expected non-zero ID")
	}
}

func TestCreateGroup_AppearsInList(t *testing.T) {
	ctx := setupGroupTest(t)

	_, err := ctx.svc.Create(ctx.vaultID, dto.CreateGroupRequest{Name: "Group A"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = ctx.svc.Create(ctx.vaultID, dto.CreateGroupRequest{Name: "Group B"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	groups, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestListGroups(t *testing.T) {
	ctx := setupGroupTest(t)

	group1 := models.Group{VaultID: ctx.vaultID, Name: "Family"}
	if err := ctx.db.Create(&group1).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}
	group2 := models.Group{VaultID: ctx.vaultID, Name: "Friends"}
	if err := ctx.db.Create(&group2).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	groups, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestGetGroup(t *testing.T) {
	ctx := setupGroupTest(t)

	group := models.Group{VaultID: ctx.vaultID, Name: "Work Team"}
	if err := ctx.db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	got, err := ctx.svc.Get(group.ID, ctx.vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Work Team" {
		t.Errorf("Expected name 'Work Team', got '%s'", got.Name)
	}
	if got.VaultID != ctx.vaultID {
		t.Errorf("Expected vault_id '%s', got '%s'", ctx.vaultID, got.VaultID)
	}
	if got.ID != group.ID {
		t.Errorf("Expected ID %d, got %d", group.ID, got.ID)
	}
	if len(got.Contacts) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(got.Contacts))
	}
}

func TestUpdateGroup(t *testing.T) {
	ctx := setupGroupTest(t)

	group := models.Group{VaultID: ctx.vaultID, Name: "Original"}
	if err := ctx.db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	updated, err := ctx.svc.Update(group.ID, ctx.vaultID, dto.UpdateGroupRequest{
		Name: "Updated Group",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "Updated Group" {
		t.Errorf("Expected name 'Updated Group', got '%s'", updated.Name)
	}
}

func TestDeleteGroup(t *testing.T) {
	ctx := setupGroupTest(t)

	group := models.Group{VaultID: ctx.vaultID, Name: "To delete"}
	if err := ctx.db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	if err := ctx.svc.Delete(group.ID, ctx.vaultID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	groups, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups after delete, got %d", len(groups))
	}
}

func TestDeleteGroupNotFound(t *testing.T) {
	ctx := setupGroupTest(t)

	err := ctx.svc.Delete(9999, ctx.vaultID)
	if err != ErrGroupNotFound {
		t.Errorf("Expected ErrGroupNotFound, got %v", err)
	}
}

func TestGetGroupNotFound(t *testing.T) {
	ctx := setupGroupTest(t)

	_, err := ctx.svc.Get(9999, ctx.vaultID)
	if err != ErrGroupNotFound {
		t.Errorf("Expected ErrGroupNotFound, got %v", err)
	}
}

func TestListGroupsReturnsContacts(t *testing.T) {
	ctx := setupGroupTest(t)

	contact := models.Contact{VaultID: ctx.vaultID, FirstName: strPtrOrNil("Alice"), LastName: strPtrOrNil("Doe")}
	if err := ctx.db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	group := models.Group{VaultID: ctx.vaultID, Name: "Family"}
	if err := ctx.db.Create(&group).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	cg := models.ContactGroup{GroupID: group.ID, ContactID: contact.ID}
	if err := ctx.db.Create(&cg).Error; err != nil {
		t.Fatalf("Create ContactGroup failed: %v", err)
	}

	groups, err := ctx.svc.List(ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Contacts) != 1 {
		t.Errorf("Expected 1 contact in group, got %d", len(groups[0].Contacts))
	}
	if groups[0].Contacts[0].FirstName != "Alice" {
		t.Errorf("Expected contact name 'Alice', got '%s'", groups[0].Contacts[0].FirstName)
	}
}

func TestCreateGroupWithGroupType(t *testing.T) {
	ctx := setupGroupTest(t)

	gt := models.GroupType{AccountID: ctx.accountID, Label: strPtrOrNil("Social")}
	if err := ctx.db.Create(&gt).Error; err != nil {
		t.Fatalf("Create GroupType failed: %v", err)
	}

	resp, err := ctx.svc.Create(ctx.vaultID, dto.CreateGroupRequest{
		Name:        "Book Club",
		GroupTypeID: &gt.ID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Name != "Book Club" {
		t.Errorf("Expected name 'Book Club', got '%s'", resp.Name)
	}
	if resp.GroupTypeID == nil || *resp.GroupTypeID != gt.ID {
		t.Errorf("Expected GroupTypeID %d, got %v", gt.ID, resp.GroupTypeID)
	}
}

func TestListByContact(t *testing.T) {
	ctx := setupGroupTest(t)

	contact := models.Contact{VaultID: ctx.vaultID, FirstName: strPtrOrNil("Bob"), LastName: strPtrOrNil("Smith")}
	if err := ctx.db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	g1 := models.Group{VaultID: ctx.vaultID, Name: "Family"}
	g2 := models.Group{VaultID: ctx.vaultID, Name: "Work"}
	g3 := models.Group{VaultID: ctx.vaultID, Name: "Gym"}
	if err := ctx.db.Create(&g1).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}
	if err := ctx.db.Create(&g2).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}
	if err := ctx.db.Create(&g3).Error; err != nil {
		t.Fatalf("Create group failed: %v", err)
	}

	ctx.db.Create(&models.ContactGroup{GroupID: g1.ID, ContactID: contact.ID})
	ctx.db.Create(&models.ContactGroup{GroupID: g3.ID, ContactID: contact.ID})

	groups, err := ctx.svc.ListByContact(contact.ID, ctx.vaultID)
	if err != nil {
		t.Fatalf("ListByContact failed: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	names := map[string]bool{}
	for _, g := range groups {
		names[g.Name] = true
	}
	if !names["Family"] || !names["Gym"] {
		t.Errorf("Expected groups Family and Gym, got %v", names)
	}
}
