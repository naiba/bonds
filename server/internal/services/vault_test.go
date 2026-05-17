package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupVaultTest(t *testing.T) (*VaultService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	regReq := dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "vault-test@example.com",
		Password:  "password123",
	}
	resp, err := authSvc.Register(regReq, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewVaultService(db), resp.User.AccountID, resp.User.ID
}

func TestCreateVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	req := dto.CreateVaultRequest{
		Name:        "My Vault",
		Description: "Test vault",
	}

	vault, err := svc.CreateVault(accountID, userID, req, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	if vault.Name != "My Vault" {
		t.Errorf("Expected name 'My Vault', got '%s'", vault.Name)
	}
	if vault.Description != "Test vault" {
		t.Errorf("Expected description 'Test vault', got '%s'", vault.Description)
	}
	if vault.ID == "" {
		t.Error("Expected vault ID to be non-empty")
	}

	db := svc.db

	var dateTypeCount int64
	db.Model(&models.ContactImportantDateType{}).Where("vault_id = ?", vault.ID).Count(&dateTypeCount)
	if dateTypeCount != 5 {
		t.Errorf("expected 5 ContactImportantDateTypes, got %d", dateTypeCount)
	}

	var moodCount int64
	db.Model(&models.MoodTrackingParameter{}).Where("vault_id = ?", vault.ID).Count(&moodCount)
	if moodCount != 5 {
		t.Errorf("expected 5 MoodTrackingParameters, got %d", moodCount)
	}

	var catCount int64
	db.Model(&models.LifeEventCategory{}).Where("vault_id = ?", vault.ID).Count(&catCount)
	if catCount != 4 {
		t.Errorf("expected 4 LifeEventCategories, got %d", catCount)
	}

	var qfCount int64
	db.Model(&models.VaultQuickFactsTemplate{}).Where("vault_id = ?", vault.ID).Count(&qfCount)
	if qfCount != 2 {
		t.Errorf("expected 2 VaultQuickFactsTemplates, got %d", qfCount)
	}
}

func TestCreateVault_UserContactAutoCreated(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{
		Name: "Contact Auto Test",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	if vault.UserContactID == "" {
		t.Fatal("Expected UserContactID to be populated after vault creation")
	}

	var uv models.UserVault
	if err := svc.db.Where("user_id = ? AND vault_id = ?", userID, vault.ID).First(&uv).Error; err != nil {
		t.Fatalf("UserVault lookup failed: %v", err)
	}
	if uv.ContactID == "" {
		t.Fatal("Expected UserVault.ContactID to be set")
	}
	if uv.ContactID != vault.UserContactID {
		t.Errorf("UserVault.ContactID (%s) != VaultResponse.UserContactID (%s)", uv.ContactID, vault.UserContactID)
	}

	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", uv.ContactID).Error; err != nil {
		t.Fatalf("Self-contact lookup failed: %v", err)
	}
	if contact.CanBeDeleted {
		t.Error("Self-contact should have CanBeDeleted=false")
	}
	if contact.Listed {
		t.Error("Self-contact should have Listed=false")
	}
	if contact.VaultID != vault.ID {
		t.Errorf("Self-contact VaultID = %s, want %s", contact.VaultID, vault.ID)
	}
}

func TestListVaults(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	_, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault 1"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	_, err = svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Vault 2"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	vaults, err := svc.ListVaults(userID)
	if err != nil {
		t.Fatalf("ListVaults failed: %v", err)
	}
	if len(vaults) != 2 {
		t.Errorf("Expected 2 vaults, got %d", len(vaults))
	}
}

func TestUpdateVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Before"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	updated, err := svc.UpdateVault(created.ID, userID, dto.UpdateVaultRequest{Name: "After", Description: "Updated"})
	if err != nil {
		t.Fatalf("UpdateVault failed: %v", err)
	}
	if updated.Name != "After" {
		t.Errorf("Expected name 'After', got '%s'", updated.Name)
	}
	if updated.Description != "Updated" {
		t.Errorf("Expected description 'Updated', got '%s'", updated.Description)
	}
}

func TestDeleteVault(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "ToDelete"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	if err := svc.DeleteVault(created.ID); err != nil {
		t.Fatalf("DeleteVault failed: %v", err)
	}

	_, err = svc.GetVault(created.ID, userID)
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}
}

func TestCheckUserVaultAccess(t *testing.T) {
	t.Run("creator_manager_permission", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Access Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(userID, created.ID, models.PermissionManager); err != nil {
			t.Errorf("Expected access, got: %v", err)
		}
	})

	t.Run("nonexistent_user", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		created, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Access Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess("nonexistent", created.ID, models.PermissionViewer); err != ErrVaultForbidden {
			t.Errorf("Expected ErrVaultForbidden, got %v", err)
		}
	})

	t.Run("editor_access_manager_only_fails", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		editorUser := models.User{
			ID:        "test-editor-1",
			AccountID: accountID,
			Email:     "editor-user@example.com",
			FirstName: strPtrOrNil("Editor"),
		}
		if err := db.Create(&editorUser).Error; err != nil {
			t.Fatalf("Create editor user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     editorUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionEditor,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(editorUser.ID, vault.ID, models.PermissionManager); err != ErrInsufficientPerm {
			t.Errorf("Expected ErrInsufficientPerm, got %v", err)
		}
	})

	t.Run("viewer_access_editor_only_fails", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		viewerUser := models.User{
			ID:        "test-viewer-1",
			AccountID: accountID,
			Email:     "viewer-user@example.com",
			FirstName: strPtrOrNil("Viewer"),
		}
		if err := db.Create(&viewerUser).Error; err != nil {
			t.Fatalf("Create viewer user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     viewerUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionViewer,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(viewerUser.ID, vault.ID, models.PermissionEditor); err != ErrInsufficientPerm {
			t.Errorf("Expected ErrInsufficientPerm, got %v", err)
		}
	})

	t.Run("editor_access_editor_required_succeeds", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)
		db := svc.db

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		editorUser := models.User{
			ID:        "test-editor-2",
			AccountID: accountID,
			Email:     "editor-user2@example.com",
			FirstName: strPtrOrNil("Editor"),
		}
		if err := db.Create(&editorUser).Error; err != nil {
			t.Fatalf("Create editor user failed: %v", err)
		}

		uv := models.UserVault{
			UserID:     editorUser.ID,
			VaultID:    vault.ID,
			Permission: models.PermissionEditor,
		}
		if err := db.Create(&uv).Error; err != nil {
			t.Fatalf("Create UserVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(editorUser.ID, vault.ID, models.PermissionEditor); err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})

	t.Run("manager_access_viewer_required_succeeds", func(t *testing.T) {
		svc, accountID, userID := setupVaultTest(t)

		vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Permission Test"}, "en")
		if err != nil {
			t.Fatalf("CreateVault failed: %v", err)
		}

		if err := svc.CheckUserVaultAccess(userID, vault.ID, models.PermissionViewer); err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})
}

func TestDeleteVault_CleanupCompleteness(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)

	vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{
		Name:        "Cleanup Test",
		Description: "vault to be fully cleaned",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	vaultID := vault.ID
	db := svc.db

	if err := svc.DeleteVault(vaultID); err != nil {
		t.Fatalf("DeleteVault failed: %v", err)
	}

	type tableCheck struct {
		name  string
		model interface{}
	}
	vaultTables := []tableCheck{
		{"ContactImportantDateType", &models.ContactImportantDateType{}},
		{"MoodTrackingParameter", &models.MoodTrackingParameter{}},
		{"LifeEventCategory", &models.LifeEventCategory{}},
		{"VaultQuickFactsTemplate", &models.VaultQuickFactsTemplate{}},
		{"Label", &models.Label{}},
		{"Company", &models.Company{}},
		{"Group", &models.Group{}},
		{"Tag", &models.Tag{}},
		{"Loan", &models.Loan{}},
		{"File", &models.File{}},
		{"Address", &models.Address{}},
		{"LifeMetric", &models.LifeMetric{}},
		{"Note", &models.Note{}},
		{"ContactTask", &models.ContactTask{}},
		{"Journal", &models.Journal{}},
		{"TimelineEvent", &models.TimelineEvent{}},
		{"ContactVaultUser", &models.ContactVaultUser{}},
		{"UserVault", &models.UserVault{}},
	}
	for _, tc := range vaultTables {
		var count int64
		// Unscoped() so soft-deletable models (Group, ContactTask) don't hide
		// orphan rows whose deleted_at was merely set — those would still
		// violate FK constraints in production. A regression of #122 left
		// these tables with soft-delete remnants but the previous scoped
		// Count() reported 0 anyway.
		db.Unscoped().Model(tc.model).Where("vault_id = ?", vaultID).Count(&count)
		if count != 0 {
			t.Errorf("%s: expected 0 records (including soft-deleted) for vault_id=%s, got %d", tc.name, vaultID, count)
		}
	}

	var contactCount int64
	db.Model(&models.Contact{}).Unscoped().Where("vault_id = ?", vaultID).Count(&contactCount)
	if contactCount != 0 {
		t.Errorf("Contact: expected 0 records for vault_id=%s, got %d", vaultID, contactCount)
	}

	var vaultCount int64
	db.Model(&models.Vault{}).Where("id = ?", vaultID).Count(&vaultCount)
	if vaultCount != 0 {
		t.Errorf("Vault: expected 0 records for id=%s, got %d", vaultID, vaultCount)
	}
}

// Real-FK regression for #122: under SetupTestDBWithFKConstraints the schema
// has actual FOREIGN KEY constraints (mirroring Postgres), so any cascade
// step that leaves orphan rows trips a "FOREIGN KEY constraint failed" at
// delete time. The plain SetupTestDB helper strips FK constraints during
// migration, so the previous test passed even on the unfixed cascade — this
// version actually reproduces the production failure mode.
func TestDeleteVault_WithForeignKeysEnabled(t *testing.T) {
	db := testutil.SetupTestDBWithFKConstraints(t)

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	regReq := dto.RegisterRequest{
		FirstName: "FK",
		LastName:  "Test",
		Email:     "fk-test@example.com",
		Password:  "password123",
	}
	resp, err := authSvc.Register(regReq, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vaultSvc := NewVaultService(db)
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{
		Name: "FK Vault",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	// Standalone vault task — covers vaultChildModels path (ContactTask has DeletedAt).
	taskSvc := NewVaultTaskService(db)
	if _, err := taskSvc.Create(vault.ID, resp.User.ID, dto.CreateVaultTaskRequest{
		Label: "Standalone vault task",
	}); err != nil {
		t.Fatalf("Create standalone vault task failed: %v", err)
	}

	// Contact + ContactImportantDate — the exact #122 repro: child rows with
	// gorm.DeletedAt whose FK points at a vault-scoped catalog row. Without
	// Unscoped() in the cascade, the row is soft-deleted (deleted_at set, row
	// still present) and its FK keeps the parent ContactImportantDateType
	// pinned, so the Step 3 catalog delete fails with FK violation.
	contact := models.Contact{
		VaultID:   vault.ID,
		FirstName: strPtrOrNil("Alice"),
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}
	var dateType models.ContactImportantDateType
	if err := db.Where("vault_id = ?", vault.ID).First(&dateType).Error; err != nil {
		t.Fatalf("Find seeded ContactImportantDateType failed: %v", err)
	}
	if err := db.Create(&models.ContactImportantDate{
		ContactID:                  contact.ID,
		ContactImportantDateTypeID: &dateType.ID,
		Label:                      "Birthday",
	}).Error; err != nil {
		t.Fatalf("Create ContactImportantDate failed: %v", err)
	}

	if err := vaultSvc.DeleteVault(vault.ID); err != nil {
		t.Fatalf("DeleteVault with foreign_keys=ON failed: %v", err)
	}

	var taskCount int64
	db.Unscoped().Model(&models.ContactTask{}).Where("vault_id = ?", vault.ID).Count(&taskCount)
	if taskCount != 0 {
		t.Errorf("Expected standalone vault tasks to be hard-deleted, got %d", taskCount)
	}

	_, err = vaultSvc.GetVault(vault.ID, resp.User.ID)
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound after deletion, got %v", err)
	}
}

// Cross-vault regression for the de47a12 follow-up to #122: when a contact
// in vault A holds a child row whose FK points at vault B's catalog (e.g.
// vault A's contact has a QuickFact filed under vault B's template), deleting
// vault B must clean up that cross-vault reference before deleting the
// catalog row — otherwise Postgres rejects the catalog delete with FK
// violation. Nullable FKs are NULL'd to preserve the child row in vault A;
// NOT-NULL FKs hard-delete the cross-vault child row.
func TestDeleteVault_CrossVaultFKCleanup(t *testing.T) {
	db := testutil.SetupTestDBWithFKConstraints(t)

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "X",
		LastName:  "Vault",
		Email:     "x-vault@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vaultSvc := NewVaultService(db)
	vaultA, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault A"}, "en")
	if err != nil {
		t.Fatalf("CreateVault A failed: %v", err)
	}
	vaultB, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault B"}, "en")
	if err != nil {
		t.Fatalf("CreateVault B failed: %v", err)
	}

	// Contact lives in vault A.
	contactA := models.Contact{
		VaultID:   vaultA.ID,
		FirstName: strPtrOrNil("CrossVault"),
	}
	if err := db.Create(&contactA).Error; err != nil {
		t.Fatalf("Create contact in vault A failed: %v", err)
	}

	// Nullable cross-vault FK: vault A's ContactImportantDate refers to
	// vault B's ContactImportantDateType.
	var bDateType models.ContactImportantDateType
	if err := db.Where("vault_id = ?", vaultB.ID).First(&bDateType).Error; err != nil {
		t.Fatalf("Find vault B ContactImportantDateType failed: %v", err)
	}
	dateInA := models.ContactImportantDate{
		ContactID:                  contactA.ID,
		ContactImportantDateTypeID: &bDateType.ID,
		Label:                      "Cross-vault birthday",
	}
	if err := db.Create(&dateInA).Error; err != nil {
		t.Fatalf("Create cross-vault ContactImportantDate failed: %v", err)
	}

	// NOT NULL cross-vault FK: vault A's QuickFact refers to vault B's
	// VaultQuickFactsTemplate. QuickFact.vault_quick_facts_template_id is
	// NOT NULL so cleanup hard-deletes the row.
	var bQFTemplate models.VaultQuickFactsTemplate
	if err := db.Where("vault_id = ?", vaultB.ID).First(&bQFTemplate).Error; err != nil {
		t.Fatalf("Find vault B VaultQuickFactsTemplate failed: %v", err)
	}
	qfInA := models.QuickFact{
		VaultQuickFactsTemplateID: bQFTemplate.ID,
		ContactID:                 contactA.ID,
		Content:                   "Cross-vault fact",
	}
	if err := db.Create(&qfInA).Error; err != nil {
		t.Fatalf("Create cross-vault QuickFact failed: %v", err)
	}

	// Delete vault B — must succeed under FK constraints despite the
	// dangling references from vault A.
	if err := vaultSvc.DeleteVault(vaultB.ID); err != nil {
		t.Fatalf("DeleteVault B failed: %v", err)
	}

	// Vault A's ContactImportantDate survives, but with type_id NULL'd.
	var dateAfter models.ContactImportantDate
	if err := db.First(&dateAfter, dateInA.ID).Error; err != nil {
		t.Fatalf("ContactImportantDate in vault A should survive cross-vault cleanup, got: %v", err)
	}
	if dateAfter.ContactImportantDateTypeID != nil {
		t.Errorf("ContactImportantDateTypeID should be NULL'd after cross-vault cleanup, got %v", *dateAfter.ContactImportantDateTypeID)
	}

	// Vault A's QuickFact must be hard-deleted (NOT NULL FK cleanup).
	var qfCount int64
	db.Model(&models.QuickFact{}).Where("id = ?", qfInA.ID).Count(&qfCount)
	if qfCount != 0 {
		t.Errorf("Cross-vault QuickFact should be hard-deleted, got %d remaining", qfCount)
	}

	// Vault A itself is untouched.
	if _, err := vaultSvc.GetVault(vaultA.ID, resp.User.ID); err != nil {
		t.Errorf("Vault A should still be readable, got: %v", err)
	}
}

// Regression for #122 (focused unit-level check): even without FK
// enforcement, deleting a vault must HARD-delete ContactImportantDate rows
// belonging to its contacts. The broken cascade only soft-deleted them
// (deleted_at set, row still in table), which on Postgres trips an FK
// violation when the parent ContactImportantDateType is deleted. This test
// asserts the post-state regardless of FK enforcement so the regression is
// caught even under SetupTestDB (no FK constraints in schema).
func TestDeleteVault_HardDeletesContactImportantDates(t *testing.T) {
	svc, accountID, userID := setupVaultTest(t)
	db := svc.db

	vault, err := svc.CreateVault(accountID, userID, dto.CreateVaultRequest{
		Name: "Important Date Vault",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contact := models.Contact{
		VaultID:   vault.ID,
		FirstName: strPtrOrNil("Alice"),
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("Create contact failed: %v", err)
	}

	var dateType models.ContactImportantDateType
	if err := db.Where("vault_id = ?", vault.ID).First(&dateType).Error; err != nil {
		t.Fatalf("Find seeded ContactImportantDateType failed: %v", err)
	}

	importantDate := models.ContactImportantDate{
		ContactID:                  contact.ID,
		ContactImportantDateTypeID: &dateType.ID,
		Label:                      "Birthday",
	}
	if err := db.Create(&importantDate).Error; err != nil {
		t.Fatalf("Create ContactImportantDate failed: %v", err)
	}

	if err := svc.DeleteVault(vault.ID); err != nil {
		t.Fatalf("DeleteVault failed: %v", err)
	}

	// Unscoped() counts soft-deleted rows too. The broken cascade soft-deletes
	// the ContactImportantDate (count > 0); the fixed cascade hard-deletes it.
	var importantDateCount int64
	db.Unscoped().Model(&models.ContactImportantDate{}).
		Where("contact_id = ?", contact.ID).
		Count(&importantDateCount)
	if importantDateCount != 0 {
		t.Errorf("Expected ContactImportantDate to be hard-deleted, got %d remaining (likely soft-deleted via gorm.DeletedAt)", importantDateCount)
	}
}

// Regression for the #122/#107 merge interaction: ContactTask lost its
// contact_id column in #107 (assignees moved to the task_contacts pivot), so
// any cascade that still tried to delete ContactTask by contact_id — or that
// dropped ContactTask rows before their task_contacts pivot rows were gone —
// would either error with "no such column: contact_id" or trip a FK
// violation on task_contacts.contact_task_id. Both failure modes are
// reproduced here under real FK constraints with a task that has two
// assignees, so the dedicated ContactTask cascade must pluck task ids, drop
// task_contacts first, then hard-delete the tasks.
func TestDeleteVault_MultiContactTaskCleansPivot(t *testing.T) {
	db := testutil.SetupTestDBWithFKConstraints(t)

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Multi",
		LastName:  "Task",
		Email:     "multi-task@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vaultSvc := NewVaultService(db)
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{
		Name: "Multi-assignee vault",
	}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactA := models.Contact{VaultID: vault.ID, FirstName: strPtrOrNil("Alice")}
	if err := db.Create(&contactA).Error; err != nil {
		t.Fatalf("Create contact A failed: %v", err)
	}
	contactB := models.Contact{VaultID: vault.ID, FirstName: strPtrOrNil("Bob")}
	if err := db.Create(&contactB).Error; err != nil {
		t.Fatalf("Create contact B failed: %v", err)
	}

	taskSvc := NewVaultTaskService(db)
	taskResp, err := taskSvc.Create(vault.ID, resp.User.ID, dto.CreateVaultTaskRequest{
		Label:      "Shared task",
		ContactIDs: []string{contactA.ID, contactB.ID},
	})
	if err != nil {
		t.Fatalf("Create multi-contact task failed: %v", err)
	}

	// Sanity: pivot rows actually exist before the cascade runs — otherwise
	// the test would pass for the wrong reason (no FK to violate).
	var pivotCount int64
	db.Model(&models.TaskContact{}).Where("contact_task_id = ?", taskResp.ID).Count(&pivotCount)
	if pivotCount != 2 {
		t.Fatalf("expected 2 TaskContact pivot rows for the new task, got %d", pivotCount)
	}

	if err := vaultSvc.DeleteVault(vault.ID); err != nil {
		t.Fatalf("DeleteVault with multi-contact task failed under FK constraints: %v", err)
	}

	var remainingTasks int64
	db.Unscoped().Model(&models.ContactTask{}).Where("vault_id = ?", vault.ID).Count(&remainingTasks)
	if remainingTasks != 0 {
		t.Errorf("expected all ContactTask rows hard-deleted, got %d", remainingTasks)
	}

	var remainingPivots int64
	db.Model(&models.TaskContact{}).Where("contact_task_id = ?", taskResp.ID).Count(&remainingPivots)
	if remainingPivots != 0 {
		t.Errorf("expected all TaskContact pivot rows deleted, got %d", remainingPivots)
	}
}
