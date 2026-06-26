package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/testutil"
)

func setupContactMoveTest(t *testing.T) (*ContactMoveService, string, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "move-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault1, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 1"}, "en")
	if err != nil {
		t.Fatalf("CreateVault 1 failed: %v", err)
	}

	vault2, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Vault 2"}, "en")
	if err != nil {
		t.Fatalf("CreateVault 2 failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault1.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	return NewContactMoveService(db), contact.ID, vault1.ID, vault2.ID, resp.User.ID
}

func TestMoveContact(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)

	resp, err := svc.Move(contactID, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if resp.VaultID != vault2ID {
		t.Errorf("Expected vault_id '%s', got '%s'", vault2ID, resp.VaultID)
	}
}

func TestMoveContactClearsCrossVaultFirstMetThrough(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	introducedBy, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{
		FirstName: "Source",
		LastName:  "Introducer",
	})
	if err != nil {
		t.Fatalf("Create introducer failed: %v", err)
	}
	if _, err := contactSvc.UpdateContact(contactID, vault1ID, userID, dto.UpdateContactRequest{
		FirstName:                "John",
		FirstMetThroughContactID: &introducedBy.ID,
	}); err != nil {
		t.Fatalf("Set first_met_through_contact_id failed: %v", err)
	}

	resp, err := svc.Move(contactID, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if resp.FirstMetThroughContactID != nil {
		t.Fatalf("expected moved response to clear source-vault met-through id, got %v", *resp.FirstMetThroughContactID)
	}
	if resp.FirstMetThroughContact != nil {
		t.Fatalf("expected moved response not to leak source-vault met-through contact, got %+v", resp.FirstMetThroughContact)
	}

	reloaded, err := contactSvc.GetContact(contactID, userID, vault2ID)
	if err != nil {
		t.Fatalf("Get moved contact failed: %v", err)
	}
	if reloaded.FirstMetThroughContactID != nil {
		t.Fatalf("expected moved contact to persist cleared met-through id, got %v", *reloaded.FirstMetThroughContactID)
	}
	if reloaded.FirstMetThroughContact != nil {
		t.Fatalf("expected moved contact not to return source-vault met-through contact, got %+v", reloaded.FirstMetThroughContact)
	}

	contacts, _, err := contactSvc.ListContacts(vault2ID, userID, 1, 15, "", "first_name", "")
	if err != nil {
		t.Fatalf("List moved contacts failed: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ID != contactID {
		t.Fatalf("expected moved contact in target vault list, got %+v", contacts)
	}
	if contacts[0].FirstMetThroughContactID != nil {
		t.Fatalf("expected target vault list to hide source-vault met-through id, got %v", *contacts[0].FirstMetThroughContactID)
	}
	if contacts[0].FirstMetThroughContact != nil {
		t.Fatalf("expected target vault list not to leak source-vault met-through contact, got %+v", contacts[0].FirstMetThroughContact)
	}
}

func TestMoveContactTargetVaultForbidden(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)
	targetVaultID := createMoveAuthTargetVault(t, svc, "move-target-forbidden@example.com")

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrVaultForbidden) {
		t.Fatalf("expected ErrVaultForbidden, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
}

func TestMoveContactTargetVaultInsufficientPermission(t *testing.T) {
	svc, contactID, vault1ID, targetVaultID, userID := setupContactMoveTest(t)
	if err := svc.db.Model(&models.UserVault{}).
		Where("vault_id = ? AND user_id = ?", targetVaultID, userID).
		Update("permission", models.PermissionViewer).Error; err != nil {
		t.Fatalf("Set viewer target vault access failed: %v", err)
	}

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrInsufficientPerm) {
		t.Fatalf("expected ErrInsufficientPerm, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
}

func TestMoveContactCrossAccountTargetForbidden(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)
	targetVaultID := createMoveAuthTargetVault(t, svc, "move-cross-account-target@example.com")
	sourceVault := getMoveTestVault(t, svc, vault1ID)
	targetVault := getMoveTestVault(t, svc, targetVaultID)
	if sourceVault.AccountID == targetVault.AccountID {
		t.Fatal("expected source and target vaults to belong to different accounts")
	}
	if err := svc.db.Create(&models.UserVault{
		VaultID:    targetVaultID,
		UserID:     userID,
		Permission: models.PermissionEditor,
	}).Error; err != nil {
		t.Fatalf("Add editor target vault access failed: %v", err)
	}

	_, err := svc.Move(contactID, vault1ID, targetVaultID, userID)
	if !errors.Is(err, ErrVaultForbidden) {
		t.Fatalf("expected ErrVaultForbidden, got %v", err)
	}
	if errors.Is(err, ErrTargetVaultNotFound) {
		t.Fatal("expected cross-account target vault to be forbidden, not reported missing")
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
	assertMoveContactNotInVault(t, svc, contactID, targetVaultID)
}

func TestMoveContactShadowSelfContactNotFound(t *testing.T) {
	svc, _, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	sourceUserVault := getMoveTestUserVault(t, svc, userID, vault1ID)
	targetUserVault := getMoveTestUserVault(t, svc, userID, vault2ID)
	shadowContactID := sourceUserVault.ContactID
	if shadowContactID == "" {
		t.Fatal("expected source user vault to have a shadow contact")
	}
	if shadowContactID == targetUserVault.ContactID {
		t.Fatal("expected each vault to have a distinct shadow contact")
	}

	_, err := svc.Move(shadowContactID, vault1ID, vault2ID, userID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, shadowContactID, vault1ID)
	shadowContact := getMoveTestContact(t, svc, shadowContactID)
	if shadowContact.CanBeDeleted || shadowContact.Listed {
		t.Fatalf("expected shadow contact to remain hidden and undeletable, got can_be_deleted=%v listed=%v", shadowContact.CanBeDeleted, shadowContact.Listed)
	}
	reloadedSourceUserVault := getMoveTestUserVault(t, svc, userID, vault1ID)
	reloadedTargetUserVault := getMoveTestUserVault(t, svc, userID, vault2ID)
	if reloadedSourceUserVault.ContactID != shadowContactID {
		t.Fatalf("expected source UserVault.ContactID to remain %s, got %s", shadowContactID, reloadedSourceUserVault.ContactID)
	}
	if reloadedTargetUserVault.ContactID != targetUserVault.ContactID {
		t.Fatalf("expected target UserVault.ContactID to remain %s, got %s", targetUserVault.ContactID, reloadedTargetUserVault.ContactID)
	}
	if reloadedTargetUserVault.ContactID == shadowContactID {
		t.Fatal("expected target UserVault.ContactID not to point at source shadow contact")
	}
}

func TestMoveContactNotFound(t *testing.T) {
	svc, _, vault1ID, vault2ID, userID := setupContactMoveTest(t)

	_, err := svc.Move("nonexistent-id", vault1ID, vault2ID, userID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestMoveContactTargetVaultNotFound(t *testing.T) {
	svc, contactID, vault1ID, _, userID := setupContactMoveTest(t)

	_, err := svc.Move(contactID, vault1ID, "nonexistent-vault", userID)
	if err != ErrTargetVaultNotFound {
		t.Errorf("Expected ErrTargetVaultNotFound, got %v", err)
	}
}

func TestMoveManyRejectsInvalidContactAndRollsBack(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	second, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "Second"})
	if err != nil {
		t.Fatalf("Create second contact failed: %v", err)
	}

	_, err = svc.MoveMany([]string{contactID, second.ID, "missing-contact"}, vault1ID, vault2ID, userID)
	if !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("expected ErrContactNotFound, got %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault1ID)
	assertMoveContactRemainsInVault(t, svc, second.ID, vault1ID)
}

func TestMoveManyKeepsBatchFirstMetThroughAndClearsOutsideIntroducer(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	batchIntroducer, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "BatchIntroducer"})
	if err != nil {
		t.Fatalf("Create batch introducer failed: %v", err)
	}
	outsideIntroducer, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "OutsideIntroducer"})
	if err != nil {
		t.Fatalf("Create outside introducer failed: %v", err)
	}
	if _, err := contactSvc.UpdateContact(contactID, vault1ID, userID, dto.UpdateContactRequest{FirstName: "John", FirstMetThroughContactID: &batchIntroducer.ID}); err != nil {
		t.Fatalf("Set batch introducer failed: %v", err)
	}
	if _, err := contactSvc.UpdateContact(batchIntroducer.ID, vault1ID, userID, dto.UpdateContactRequest{FirstName: "BatchIntroducer", FirstMetThroughContactID: &outsideIntroducer.ID}); err != nil {
		t.Fatalf("Set outside introducer failed: %v", err)
	}

	resp, err := svc.MoveMany([]string{contactID, batchIntroducer.ID}, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	if resp.MovedCount != 2 {
		t.Fatalf("expected 2 moved contacts, got %+v", resp)
	}
	moved := getMoveTestContact(t, svc, contactID)
	if moved.FirstMetThroughContactID == nil || *moved.FirstMetThroughContactID != batchIntroducer.ID {
		t.Fatalf("expected moved contact to keep in-batch introducer, got %v", moved.FirstMetThroughContactID)
	}
	introduced := getMoveTestContact(t, svc, batchIntroducer.ID)
	if introduced.FirstMetThroughContactID != nil {
		t.Fatalf("expected in-batch contact to clear outside introducer, got %v", *introduced.FirstMetThroughContactID)
	}
}

func TestMoveManyUpdatesOwnedRowsAndCleansSourceScopedPivots(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	second, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "Second"})
	if err != nil {
		t.Fatalf("Create second contact failed: %v", err)
	}
	label := models.Label{VaultID: vault1ID, Name: "Source Label", Slug: "source-label"}
	group := models.Group{VaultID: vault1ID, Name: "Source Group"}
	company := models.Company{VaultID: vault1ID, Name: "Source Company"}
	if err := svc.db.Create(&label).Error; err != nil {
		t.Fatalf("create label failed: %v", err)
	}
	if err := svc.db.Create(&group).Error; err != nil {
		t.Fatalf("create group failed: %v", err)
	}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatalf("create company failed: %v", err)
	}
	fileType := "contact_photo"
	fileContactID := contactID
	rows := []interface{}{
		&models.Note{ContactID: contactID, VaultID: vault1ID, Body: "note"},
		&models.ContactVaultUser{ContactID: contactID, VaultID: vault1ID, UserID: userID, NumberOfViews: 3},
		&models.File{VaultID: vault1ID, UfileableID: &fileContactID, UUID: "file-uuid", Name: "file", Type: fileType, MimeType: "image/png", Size: 10},
		&models.ContactLabel{ContactID: contactID, LabelID: label.ID},
		&models.ContactGroup{ContactID: contactID, GroupID: group.ID},
		&models.ContactCompany{ContactID: contactID, CompanyID: company.ID},
	}
	for _, row := range rows {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create move related row failed: %v", err)
		}
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("company_id", company.ID).Error; err != nil {
		t.Fatalf("set company id failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID, second.ID, contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault2ID)
	assertMoveContactRemainsInVault(t, svc, second.ID, vault2ID)
	assertMoveCount(t, svc, &models.Note{}, "contact_id = ? AND vault_id = ?", 1, contactID, vault2ID)
	assertMoveCount(t, svc, &models.ContactVaultUser{}, "contact_id = ? AND vault_id = ?", 2, contactID, vault2ID)
	assertMoveCount(t, svc, &models.File{}, "ufileable_id = ? AND vault_id = ?", 1, contactID, vault2ID)
	assertMoveCount(t, svc, &models.ContactLabel{}, "contact_id = ?", 0, contactID)
	assertMoveCount(t, svc, &models.ContactGroup{}, "contact_id = ?", 0, contactID)
	assertMoveCount(t, svc, &models.ContactCompany{}, "contact_id = ?", 0, contactID)
	moved := getMoveTestContact(t, svc, contactID)
	if moved.CompanyID != nil {
		t.Fatalf("expected source company_id cleared, got %v", *moved.CompanyID)
	}
}

func TestMoveManyMovesAvatarFileAndKeepsContactFileID(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	fileContactID := contactID
	fileableType := "Contact"
	avatarFile := models.File{
		VaultID:      vault1ID,
		UfileableID:  &fileContactID,
		FileableType: &fileableType,
		UUID:         "avatar-file-uuid",
		Name:         "avatar.png",
		Type:         "avatar",
		MimeType:     "image/png",
		Size:         100,
	}
	if err := svc.db.Create(&avatarFile).Error; err != nil {
		t.Fatalf("create avatar file failed: %v", err)
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("file_id", avatarFile.ID).Error; err != nil {
		t.Fatalf("set avatar file id failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}

	moved := getMoveTestContact(t, svc, contactID)
	if moved.FileID == nil || *moved.FileID != avatarFile.ID {
		t.Fatalf("expected moved contact to keep avatar file_id %d, got %v", avatarFile.ID, moved.FileID)
	}
	var movedAvatar models.File
	if err := svc.db.First(&movedAvatar, avatarFile.ID).Error; err != nil {
		t.Fatalf("reload moved avatar file failed: %v", err)
	}
	if movedAvatar.VaultID != vault2ID {
		t.Fatalf("expected avatar file vault %s after move, got %s", vault2ID, movedAvatar.VaultID)
	}
	if movedAvatar.UfileableID == nil || *movedAvatar.UfileableID != contactID {
		t.Fatalf("expected avatar ufileable_id to stay %s, got %v", contactID, movedAvatar.UfileableID)
	}
	assertMoveCount(t, svc, &models.File{}, "id = ? AND vault_id = ?", 0, avatarFile.ID, vault1ID)
	assertMoveCount(t, svc, &models.File{}, "id = ? AND vault_id = ? AND type = ?", 1, avatarFile.ID, vault2ID, "avatar")
}

func TestMoveManyMovesFullTasksLoansAndStripsMixedPivots(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	second, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "Second"})
	if err != nil {
		t.Fatalf("Create second failed: %v", err)
	}
	outside, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "Outside"})
	if err != nil {
		t.Fatalf("Create outside failed: %v", err)
	}
	fullTask := models.ContactTask{VaultID: vault1ID, AuthorName: "Tester", Label: "Full task"}
	mixedTask := models.ContactTask{VaultID: vault1ID, AuthorName: "Tester", Label: "Mixed task"}
	fullLoan := models.Loan{VaultID: vault1ID, Type: "lend", Name: "Full loan"}
	mixedLoan := models.Loan{VaultID: vault1ID, Type: "lend", Name: "Mixed loan"}
	for _, row := range []interface{}{&fullTask, &mixedTask, &fullLoan, &mixedLoan} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create task/loan failed: %v", err)
		}
	}
	pivots := []interface{}{
		&models.TaskContact{ContactTaskID: fullTask.ID, ContactID: contactID},
		&models.TaskContact{ContactTaskID: fullTask.ID, ContactID: second.ID},
		&models.TaskContact{ContactTaskID: mixedTask.ID, ContactID: contactID},
		&models.TaskContact{ContactTaskID: mixedTask.ID, ContactID: outside.ID},
		&models.ContactLoan{LoanID: fullLoan.ID, LoanerID: contactID, LoaneeID: second.ID},
		&models.ContactLoan{LoanID: mixedLoan.ID, LoanerID: contactID, LoaneeID: outside.ID},
	}
	for _, row := range pivots {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create pivot failed: %v", err)
		}
	}

	if _, err := svc.MoveMany([]string{contactID, second.ID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveCount(t, svc, &models.ContactTask{}, "id = ? AND vault_id = ?", 1, fullTask.ID, vault2ID)
	assertMoveCount(t, svc, &models.ContactTask{}, "id = ? AND vault_id = ?", 1, mixedTask.ID, vault1ID)
	assertMoveCount(t, svc, &models.TaskContact{}, "contact_task_id = ?", 2, fullTask.ID)
	assertMoveCount(t, svc, &models.TaskContact{}, "contact_task_id = ? AND contact_id = ?", 0, mixedTask.ID, contactID)
	assertMoveCount(t, svc, &models.TaskContact{}, "contact_task_id = ? AND contact_id = ?", 1, mixedTask.ID, outside.ID)
	assertMoveCount(t, svc, &models.Loan{}, "id = ? AND vault_id = ?", 1, fullLoan.ID, vault2ID)
	assertMoveCount(t, svc, &models.Loan{}, "id = ? AND vault_id = ?", 1, mixedLoan.ID, vault1ID)
	assertMoveCount(t, svc, &models.ContactLoan{}, "loan_id = ?", 1, fullLoan.ID)
	assertMoveCount(t, svc, &models.ContactLoan{}, "loan_id = ?", 0, mixedLoan.ID)
}

func TestMoveManyCleansLifeEventPivotsAndOrphanTimeline(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	lifeSvc := NewLifeEventService(svc.db)
	typeID := getLifeEventTypeIDForMoveVault(t, svc, vault1ID)
	created, err := lifeSvc.CreateDashboardLifeEvent(vault1ID, dto.CreateLifeEventRequest{
		LifeEventTypeID: typeID,
		HappenedAt:      time.Now(),
		Summary:         "Move cleanup",
		Participants:    []string{contactID},
	})
	if err != nil {
		t.Fatalf("CreateDashboardLifeEvent failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveCount(t, svc, &models.LifeEventParticipant{}, "contact_id = ?", 0, contactID)
	assertMoveCount(t, svc, &models.TimelineEventParticipant{}, "contact_id = ?", 0, contactID)
	assertMoveCount(t, svc, &models.LifeEvent{}, "id = ?", 0, created.LifeEvents[0].ID)
	assertMoveCount(t, svc, &models.TimelineEvent{}, "id = ?", 0, created.ID)
}

func TestMoveManyAllowsArchivedContacts(t *testing.T) {
	svc, _, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	listed := false
	contactSvc := NewContactService(svc.db)
	archived, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "Archived", Listed: &listed})
	if err != nil {
		t.Fatalf("Create archived contact failed: %v", err)
	}

	resp, err := svc.MoveMany([]string{archived.ID}, vault1ID, vault2ID, userID)
	if err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	if resp.MovedCount != 1 || resp.Contacts[0].VaultID != vault2ID {
		t.Fatalf("expected archived contact moved to target vault, got %+v", resp)
	}
	if resp.Contacts[0].Listed {
		t.Fatal("expected archived contact response to remain unlisted")
	}
	reloaded := getMoveTestContact(t, svc, archived.ID)
	if reloaded.Listed {
		t.Fatal("expected archived contact to stay archived after move")
	}
}

func TestMoveManySameVaultDoesNotCleanRelationships(t *testing.T) {
	svc, contactID, vaultID, _, userID := setupContactMoveTest(t)
	label := models.Label{VaultID: vaultID, Name: "Keep Label", Slug: "keep-label"}
	group := models.Group{VaultID: vaultID, Name: "Keep Group"}
	company := models.Company{VaultID: vaultID, Name: "Keep Company"}
	journal := models.Journal{VaultID: vaultID, Name: "Keep Journal"}
	lifeMetric := models.LifeMetric{VaultID: vaultID, Label: "Keep Metric"}
	subscription := models.AddressBookSubscription{VaultID: vaultID, UserID: userID, URI: "https://dav.example.com/source/", Username: "user", Password: "encrypted", Active: true, SyncWay: SyncWayPush, Capabilities: "{}"}
	for _, row := range []interface{}{&label, &group, &company, &journal, &lifeMetric, &subscription} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create same-vault related row failed: %v", err)
		}
	}
	post := models.Post{JournalID: journal.ID, WrittenAt: time.Now()}
	if err := svc.db.Create(&post).Error; err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	for _, row := range []interface{}{
		&models.ContactLabel{ContactID: contactID, LabelID: label.ID},
		&models.ContactGroup{ContactID: contactID, GroupID: group.ID},
		&models.ContactCompany{ContactID: contactID, CompanyID: company.ID},
		&models.ContactPost{ContactID: contactID, PostID: post.ID},
		&models.ContactLifeMetric{ContactID: contactID, LifeMetricID: lifeMetric.ID, UserID: userID},
		&models.ContactSubscriptionState{ContactID: contactID, AddressBookSubscriptionID: subscription.ID, DistantURI: "https://dav.example.com/source/contact.vcf", DistantEtag: "etag"},
		&models.DavSyncLog{ContactID: &contactID, AddressBookSubscriptionID: subscription.ID, DistantURI: "https://dav.example.com/source/contact.vcf", Action: "pushed"},
	} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create same-vault pivot failed: %v", err)
		}
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", contactID).Update("company_id", company.ID).Error; err != nil {
		t.Fatalf("set company id failed: %v", err)
	}

	resp, err := svc.MoveMany([]string{contactID}, vaultID, vaultID, userID)
	if err != nil {
		t.Fatalf("same-vault MoveMany failed: %v", err)
	}
	if resp.MovedCount != 1 || resp.Contacts[0].VaultID != vaultID {
		t.Fatalf("expected same-vault response to keep vault, got %+v", resp)
	}
	if resp.Contacts[0].CompanyID == nil || *resp.Contacts[0].CompanyID != company.ID {
		t.Fatalf("expected same-vault response to preserve company_id %d, got %v", company.ID, resp.Contacts[0].CompanyID)
	}
	assertMoveCount(t, svc, &models.ContactLabel{}, "contact_id = ? AND label_id = ?", 1, contactID, label.ID)
	assertMoveCount(t, svc, &models.ContactGroup{}, "contact_id = ? AND group_id = ?", 1, contactID, group.ID)
	assertMoveCount(t, svc, &models.ContactCompany{}, "contact_id = ? AND company_id = ?", 1, contactID, company.ID)
	assertMoveCount(t, svc, &models.ContactPost{}, "contact_id = ? AND post_id = ?", 1, contactID, post.ID)
	assertMoveCount(t, svc, &models.ContactLifeMetric{}, "contact_id = ? AND life_metric_id = ?", 1, contactID, lifeMetric.ID)
	assertMoveCount(t, svc, &models.ContactSubscriptionState{}, "contact_id = ? AND address_book_subscription_id = ?", 1, contactID, subscription.ID)
	assertMoveCount(t, svc, &models.DavSyncLog{}, "contact_id = ? AND address_book_subscription_id = ?", 1, contactID, subscription.ID)
}

func TestMoveManyClonesSharedAddressesAndMovesOwnedAddresses(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	contactSvc := NewContactService(svc.db)
	outside, err := contactSvc.CreateContact(vault1ID, userID, dto.CreateContactRequest{FirstName: "AddressOutside"})
	if err != nil {
		t.Fatalf("Create outside contact failed: %v", err)
	}
	sharedLine := "Shared street"
	ownedLine := "Owned street"
	sharedAddress := models.Address{VaultID: vault1ID, Line1: &sharedLine}
	ownedAddress := models.Address{VaultID: vault1ID, Line1: &ownedLine}
	if err := svc.db.Create(&sharedAddress).Error; err != nil {
		t.Fatalf("create shared address failed: %v", err)
	}
	if err := svc.db.Create(&ownedAddress).Error; err != nil {
		t.Fatalf("create owned address failed: %v", err)
	}
	for _, row := range []interface{}{
		&models.ContactAddress{ContactID: contactID, AddressID: sharedAddress.ID},
		&models.ContactAddress{ContactID: outside.ID, AddressID: sharedAddress.ID},
		&models.ContactAddress{ContactID: contactID, AddressID: ownedAddress.ID},
	} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create contact address failed: %v", err)
		}
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	var sharedReloaded models.Address
	if err := svc.db.First(&sharedReloaded, sharedAddress.ID).Error; err != nil {
		t.Fatalf("reload shared address failed: %v", err)
	}
	if sharedReloaded.VaultID != vault1ID {
		t.Fatalf("expected shared source address to remain in source vault, got %s", sharedReloaded.VaultID)
	}
	assertMoveCount(t, svc, &models.ContactAddress{}, "contact_id = ? AND address_id = ?", 0, contactID, sharedAddress.ID)
	assertMoveCount(t, svc, &models.ContactAddress{}, "contact_id = ? AND address_id = ?", 1, outside.ID, sharedAddress.ID)
	var copiedShared models.Address
	if err := svc.db.Where("vault_id = ? AND line1 = ?", vault2ID, sharedLine).First(&copiedShared).Error; err != nil {
		t.Fatalf("expected copied shared address in target vault: %v", err)
	}
	assertMoveCount(t, svc, &models.ContactAddress{}, "contact_id = ? AND address_id = ?", 1, contactID, copiedShared.ID)
	var ownedReloaded models.Address
	if err := svc.db.First(&ownedReloaded, ownedAddress.ID).Error; err != nil {
		t.Fatalf("reload owned address failed: %v", err)
	}
	if ownedReloaded.VaultID != vault2ID {
		t.Fatalf("expected fully-owned address to move to target vault, got %s", ownedReloaded.VaultID)
	}
}

func TestMoveManyRemapsVaultScopedContactDetails(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	svc.SetFileService(NewVaultFileService(svc.db, t.TempDir()))
	dateUUID := "date-uuid"
	dateEtag := "date-etag"
	dateURI := "https://dav.example.com/date.ics"
	sourceType := models.ContactImportantDateType{VaultID: vault1ID, Label: "Custom migration date"}
	targetType := models.ContactImportantDateType{VaultID: vault2ID, Label: "Custom migration date"}
	missingType := models.ContactImportantDateType{VaultID: vault1ID, Label: "Source only date"}
	quickTranslationKey := "quick_fact.favorite_place"
	quickLabel := "Favorite place"
	sourceQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault1ID, Label: &quickLabel, LabelTranslationKey: &quickTranslationKey, FieldType: "text"}
	targetQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault2ID, Label: &quickLabel, LabelTranslationKey: &quickTranslationKey, FieldType: "text"}
	missingQuickLabel := "Source-only quick fact"
	missingQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault1ID, Label: &missingQuickLabel, FieldType: "text"}
	moodTranslationKey := "mood.good"
	moodLabel := "Good"
	sourceMood := models.MoodTrackingParameter{VaultID: vault1ID, Label: &moodLabel, LabelTranslationKey: &moodTranslationKey, HexColor: "#00ff00"}
	targetMood := models.MoodTrackingParameter{VaultID: vault2ID, Label: &moodLabel, LabelTranslationKey: &moodTranslationKey, HexColor: "#00ff00"}
	missingMoodLabel := "Source-only mood"
	missingMood := models.MoodTrackingParameter{VaultID: vault1ID, Label: &missingMoodLabel, HexColor: "#ff0000"}
	for _, row := range []interface{}{&sourceType, &targetType, &missingType, &sourceQuickTemplate, &targetQuickTemplate, &missingQuickTemplate, &sourceMood, &targetMood, &missingMood} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create vault-scoped row failed: %v", err)
		}
	}
	dateDay := 1
	matchedDate := models.ContactImportantDate{ContactID: contactID, ContactImportantDateTypeID: &sourceType.ID, Label: "Custom migration date", Day: &dateDay, DistantUUID: &dateUUID, DistantEtag: &dateEtag, DistantURI: &dateURI}
	missingDate := models.ContactImportantDate{ContactID: contactID, ContactImportantDateTypeID: &missingType.ID, Label: "Source only", Day: &dateDay, DistantUUID: &dateUUID, DistantEtag: &dateEtag, DistantURI: &dateURI}
	untypedDate := models.ContactImportantDate{ContactID: contactID, Label: "Untyped", Day: &dateDay, DistantUUID: &dateUUID, DistantEtag: &dateEtag, DistantURI: &dateURI}
	quickFact := models.QuickFact{ContactID: contactID, VaultQuickFactsTemplateID: sourceQuickTemplate.ID, Content: "Park"}
	fileContactID := contactID
	quickFile := models.File{VaultID: vault1ID, UfileableID: &fileContactID, UUID: "quick-file", Name: "quick-file", Type: "quick_fact", MimeType: "text/plain", Size: 1}
	if err := svc.db.Create(&quickFile).Error; err != nil {
		t.Fatalf("create quick fact file failed: %v", err)
	}
	missingQuickFact := models.QuickFact{ContactID: contactID, VaultQuickFactsTemplateID: missingQuickTemplate.ID, Content: "Delete me", FileID: &quickFile.ID}
	matchedMood := models.MoodTrackingEvent{ContactID: contactID, MoodTrackingParameterID: sourceMood.ID, RatedAt: time.Now()}
	missingMoodEvent := models.MoodTrackingEvent{ContactID: contactID, MoodTrackingParameterID: missingMood.ID, RatedAt: time.Now()}
	for _, row := range []interface{}{&matchedDate, &missingDate, &untypedDate, &quickFact, &missingQuickFact, &matchedMood, &missingMoodEvent} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create contact detail row failed: %v", err)
		}
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	var reloadedDate models.ContactImportantDate
	if err := svc.db.First(&reloadedDate, matchedDate.ID).Error; err != nil {
		t.Fatalf("reload matched date failed: %v", err)
	}
	if reloadedDate.ContactImportantDateTypeID == nil || *reloadedDate.ContactImportantDateTypeID != targetType.ID {
		var got uint
		if reloadedDate.ContactImportantDateTypeID != nil {
			got = *reloadedDate.ContactImportantDateTypeID
		}
		t.Fatalf("expected matched date type %d, got %d (source type %d)", targetType.ID, got, sourceType.ID)
	}
	if reloadedDate.DistantUUID != nil || reloadedDate.DistantEtag != nil || reloadedDate.DistantURI != nil {
		t.Fatal("expected remapped important date remote fields to be cleared")
	}
	for _, id := range []uint{missingDate.ID, untypedDate.ID} {
		var date models.ContactImportantDate
		if err := svc.db.First(&date, id).Error; err != nil {
			t.Fatalf("reload date %d failed: %v", id, err)
		}
		if date.ContactImportantDateTypeID != nil || date.DistantUUID != nil || date.DistantEtag != nil || date.DistantURI != nil {
			t.Fatalf("expected date %d to clear type/remote fields, got type=%v uuid=%v etag=%v uri=%v", id, date.ContactImportantDateTypeID, date.DistantUUID, date.DistantEtag, date.DistantURI)
		}
	}
	var reloadedQuickFact models.QuickFact
	if err := svc.db.First(&reloadedQuickFact, quickFact.ID).Error; err != nil {
		t.Fatalf("reload quick fact failed: %v", err)
	}
	if reloadedQuickFact.VaultQuickFactsTemplateID != targetQuickTemplate.ID {
		t.Fatalf("expected quick fact template %d, got %d", targetQuickTemplate.ID, reloadedQuickFact.VaultQuickFactsTemplateID)
	}
	assertMoveCount(t, svc, &models.QuickFact{}, "id = ?", 0, missingQuickFact.ID)
	assertMoveCount(t, svc, &models.File{}, "id = ?", 0, quickFile.ID)
	var reloadedMood models.MoodTrackingEvent
	if err := svc.db.First(&reloadedMood, matchedMood.ID).Error; err != nil {
		t.Fatalf("reload mood event failed: %v", err)
	}
	if reloadedMood.MoodTrackingParameterID != targetMood.ID {
		t.Fatalf("expected mood parameter %d, got %d", targetMood.ID, reloadedMood.MoodTrackingParameterID)
	}
	assertMoveCount(t, svc, &models.MoodTrackingEvent{}, "id = ?", 0, missingMoodEvent.ID)
}

func TestMoveManyReschedulesRemindersAndCleansSourceDavState(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	now := time.Now()
	if err := svc.db.Model(&models.UserNotificationChannel{}).Where("user_id = ?", userID).Update("active", false).Error; err != nil {
		t.Fatalf("deactivate seeded channels failed: %v", err)
	}
	channel := models.UserNotificationChannel{UserID: &userID, Type: "email", Content: "move-reminder@example.com", Active: true, VerifiedAt: &now}
	if err := svc.db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}
	day, month, year := now.Day(), int(now.Month()), now.Year()+1
	reminder := models.ContactReminder{ContactID: contactID, Label: "Move reminder", Day: &day, Month: &month, Year: &year, Type: "one_time"}
	if err := svc.db.Create(&reminder).Error; err != nil {
		t.Fatalf("create reminder failed: %v", err)
	}
	oldScheduled := models.ContactReminderScheduled{ContactReminderID: reminder.ID, UserNotificationChannelID: channel.ID, ScheduledAt: now.Add(-time.Hour)}
	if err := svc.db.Create(&oldScheduled).Error; err != nil {
		t.Fatalf("create old schedule failed: %v", err)
	}
	sourceSub := models.AddressBookSubscription{VaultID: vault1ID, UserID: userID, URI: "https://dav.example.com/source/", Username: "user", Password: "encrypted", Active: true, SyncWay: SyncWayPush, Capabilities: "{}"}
	targetSub := models.AddressBookSubscription{VaultID: vault2ID, UserID: userID, URI: "https://dav.example.com/target/", Username: "user", Password: "encrypted", Active: true, SyncWay: SyncWayPush, Capabilities: "{}"}
	if err := svc.db.Create(&sourceSub).Error; err != nil {
		t.Fatalf("create source subscription failed: %v", err)
	}
	if err := svc.db.Create(&targetSub).Error; err != nil {
		t.Fatalf("create target subscription failed: %v", err)
	}
	for _, row := range []interface{}{
		&models.ContactSubscriptionState{ContactID: contactID, AddressBookSubscriptionID: sourceSub.ID, DistantURI: "https://dav.example.com/source/contact.vcf", DistantEtag: "source-etag"},
		&models.ContactSubscriptionState{ContactID: contactID, AddressBookSubscriptionID: targetSub.ID, DistantURI: "https://dav.example.com/target/contact.vcf", DistantEtag: "target-etag"},
		&models.DavSyncLog{ContactID: &contactID, AddressBookSubscriptionID: sourceSub.ID, DistantURI: "https://dav.example.com/source/contact.vcf", Action: "pushed"},
		&models.DavSyncLog{ContactID: &contactID, AddressBookSubscriptionID: targetSub.ID, DistantURI: "https://dav.example.com/target/contact.vcf", Action: "pushed"},
	} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create DAV row failed: %v", err)
		}
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveCount(t, svc, &models.ContactReminderScheduled{}, "id = ?", 0, oldScheduled.ID)
	assertMoveCount(t, svc, &models.ContactReminderScheduled{}, "contact_reminder_id = ? AND triggered_at IS NULL", 1, reminder.ID)
	assertMoveCount(t, svc, &models.ContactSubscriptionState{}, "contact_id = ? AND address_book_subscription_id = ?", 0, contactID, sourceSub.ID)
	assertMoveCount(t, svc, &models.DavSyncLog{}, "contact_id = ? AND address_book_subscription_id = ?", 0, contactID, sourceSub.ID)
	assertMoveCount(t, svc, &models.ContactSubscriptionState{}, "contact_id = ? AND address_book_subscription_id = ?", 1, contactID, targetSub.ID)
	assertMoveCount(t, svc, &models.DavSyncLog{}, "contact_id = ? AND address_book_subscription_id = ?", 1, contactID, targetSub.ID)
}

func TestMoveManyReindexesMovedContactsAndNotes(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	engine := &contactMoveRecordingSearchEngine{contactVaults: map[string]string{}, noteVaults: map[string]string{}}
	svc.SetSearchService(NewSearchService(engine))
	title := "Moved note"
	note := models.Note{ContactID: contactID, VaultID: vault1ID, Title: &title, Body: "Note body"}
	if err := svc.db.Create(&note).Error; err != nil {
		t.Fatalf("create note failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	if got := engine.contactVaults[contactID]; got != vault2ID {
		t.Fatalf("expected contact indexed in vault %s, got %s", vault2ID, got)
	}
	noteID := fmt.Sprintf("%d", note.ID)
	if got := engine.noteVaults[noteID]; got != vault2ID {
		t.Fatalf("expected note indexed in vault %s, got %s", vault2ID, got)
	}
}

func TestMoveManyReturnsSearchReindexContactError(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	indexErr := errors.New("contact index failed")
	engine := &contactMoveRecordingSearchEngine{contactVaults: map[string]string{}, noteVaults: map[string]string{}, indexContactErr: indexErr}
	svc.SetSearchService(NewSearchService(engine))

	_, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID)
	if !errors.Is(err, indexErr) {
		t.Fatalf("expected contact index error, got %v", err)
	}
	if len(engine.deletedDocuments) != 1 || engine.deletedDocuments[0] != "contact:"+contactID {
		t.Fatalf("expected failed contact index to delete stale document, got %#v", engine.deletedDocuments)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault2ID)
}

func TestMoveManyReturnsSearchReindexNoteError(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	indexErr := errors.New("note index failed")
	engine := &contactMoveRecordingSearchEngine{contactVaults: map[string]string{}, noteVaults: map[string]string{}, indexNoteErr: indexErr}
	svc.SetSearchService(NewSearchService(engine))
	title := "Moved note"
	note := models.Note{ContactID: contactID, VaultID: vault1ID, Title: &title, Body: "Note body"}
	if err := svc.db.Create(&note).Error; err != nil {
		t.Fatalf("create note failed: %v", err)
	}

	_, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID)
	if !errors.Is(err, indexErr) {
		t.Fatalf("expected note index error, got %v", err)
	}
	noteDocumentID := fmt.Sprintf("note:%d", note.ID)
	if len(engine.deletedDocuments) != 1 || engine.deletedDocuments[0] != noteDocumentID {
		t.Fatalf("expected failed note index to delete stale document %s, got %#v", noteDocumentID, engine.deletedDocuments)
	}
	assertMoveContactRemainsInVault(t, svc, contactID, vault2ID)
}

func TestMoveManyDeletesMissingQuickFactFileFromDiskAndDB(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	uploadDir := t.TempDir()
	svc.SetFileService(NewVaultFileService(svc.db, uploadDir))
	missingQuickLabel := "Source-only file quick fact"
	missingQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault1ID, Label: &missingQuickLabel, FieldType: "document"}
	if err := svc.db.Create(&missingQuickTemplate).Error; err != nil {
		t.Fatalf("create missing quick fact template failed: %v", err)
	}
	quickFile := models.File{VaultID: vault1ID, UUID: "quick-fact-file", Name: "quick-fact-file", Type: "quick_fact", MimeType: "text/plain", Size: 1}
	if err := svc.db.Create(&quickFile).Error; err != nil {
		t.Fatalf("create quick fact file failed: %v", err)
	}
	quickFact := models.QuickFact{ContactID: contactID, VaultQuickFactsTemplateID: missingQuickTemplate.ID, Content: "Delete me", FileID: &quickFile.ID}
	if err := svc.db.Create(&quickFact).Error; err != nil {
		t.Fatalf("create quick fact failed: %v", err)
	}
	quickFactType := "QuickFact"
	if err := svc.db.Model(&quickFile).Updates(map[string]interface{}{"fileable_type": quickFactType, "fileable_id": quickFact.ID}).Error; err != nil {
		t.Fatalf("link quick fact file failed: %v", err)
	}
	filePath := filepath.Join(uploadDir, quickFile.UUID)
	if err := os.WriteFile(filePath, []byte("quick fact file"), 0o644); err != nil {
		t.Fatalf("write quick fact file failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveCount(t, svc, &models.QuickFact{}, "id = ?", 0, quickFact.ID)
	assertMoveCount(t, svc, &models.File{}, "id = ?", 0, quickFile.ID)
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected quick fact disk file to be removed, stat err=%v", err)
	}
}

func TestMoveManyDoesNotMoveFilesLinkedOnlyByQuickFactFileable(t *testing.T) {
	svc, contactID, vault1ID, vault2ID, userID := setupContactMoveTest(t)
	quickLabel := "Matched file quick fact"
	sourceQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault1ID, Label: &quickLabel, FieldType: "document"}
	targetQuickTemplate := models.VaultQuickFactsTemplate{VaultID: vault2ID, Label: &quickLabel, FieldType: "document"}
	for _, row := range []interface{}{&sourceQuickTemplate, &targetQuickTemplate} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatalf("create quick fact template failed: %v", err)
		}
	}
	quickFact := models.QuickFact{ContactID: contactID, VaultQuickFactsTemplateID: sourceQuickTemplate.ID, Content: "Keep me"}
	if err := svc.db.Create(&quickFact).Error; err != nil {
		t.Fatalf("create quick fact failed: %v", err)
	}
	quickFactType := "QuickFact"
	quickFile := models.File{VaultID: vault1ID, FileableID: &quickFact.ID, FileableType: &quickFactType, UUID: "quick-fact-fileable", Name: "quick-fact-fileable", Type: "quick_fact", MimeType: "text/plain", Size: 1}
	if err := svc.db.Create(&quickFile).Error; err != nil {
		t.Fatalf("create quick fact file failed: %v", err)
	}

	if _, err := svc.MoveMany([]string{contactID}, vault1ID, vault2ID, userID); err != nil {
		t.Fatalf("MoveMany failed: %v", err)
	}
	assertMoveCount(t, svc, &models.QuickFact{}, "id = ? AND vault_quick_facts_template_id = ?", 1, quickFact.ID, targetQuickTemplate.ID)
	assertMoveCount(t, svc, &models.File{}, "id = ? AND vault_id = ?", 1, quickFile.ID, vault1ID)
	assertMoveCount(t, svc, &models.File{}, "id = ? AND vault_id = ?", 0, quickFile.ID, vault2ID)
}

func assertMoveCount(t *testing.T, svc *ContactMoveService, model interface{}, query string, expected int64, args ...interface{}) {
	t.Helper()
	var count int64
	if err := svc.db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T failed: %v", model, err)
	}
	if count != expected {
		t.Fatalf("expected count %d for %T where %s, got %d", expected, model, query, count)
	}
}

func getLifeEventTypeIDForMoveVault(t *testing.T, svc *ContactMoveService, vaultID string) uint {
	t.Helper()
	var typeID uint
	if err := svc.db.Model(&models.LifeEventType{}).
		Joins("JOIN life_event_categories ON life_event_categories.id = life_event_types.life_event_category_id").
		Where("life_event_categories.vault_id = ?", vaultID).
		Select("life_event_types.id").Limit(1).Scan(&typeID).Error; err != nil {
		t.Fatalf("load life event type failed: %v", err)
	}
	if typeID == 0 {
		t.Fatal("expected seeded life event type")
	}
	return typeID
}

func createMoveAuthTargetVault(t *testing.T, svc *ContactMoveService, email string) string {
	t.Helper()
	authSvc := NewAuthService(svc.db, testutil.TestJWTConfig())
	vaultSvc := NewVaultService(svc.db)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Target",
		LastName:  "Owner",
		Email:     email,
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register target owner failed: %v", err)
	}
	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Target Vault"}, "en")
	if err != nil {
		t.Fatalf("Create target vault failed: %v", err)
	}
	return vault.ID
}

func assertMoveContactRemainsInVault(t *testing.T, svc *ContactMoveService, contactID, vaultID string) {
	t.Helper()
	contact := getMoveTestContact(t, svc, contactID)
	if contact.VaultID != vaultID {
		t.Fatalf("expected contact to remain in source vault %s, got %s", vaultID, contact.VaultID)
	}
}

func assertMoveContactNotInVault(t *testing.T, svc *ContactMoveService, contactID, vaultID string) {
	t.Helper()
	var count int64
	if err := svc.db.Model(&models.Contact{}).Where("id = ? AND vault_id = ?", contactID, vaultID).Count(&count).Error; err != nil {
		t.Fatalf("Count contact in vault failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no contact %s in vault %s, got %d", contactID, vaultID, count)
	}
}

func getMoveTestContact(t *testing.T, svc *ContactMoveService, contactID string) models.Contact {
	t.Helper()
	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", contactID).Error; err != nil {
		t.Fatalf("Reload contact failed: %v", err)
	}
	return contact
}

func getMoveTestVault(t *testing.T, svc *ContactMoveService, vaultID string) models.Vault {
	t.Helper()
	var vault models.Vault
	if err := svc.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("Load vault failed: %v", err)
	}
	return vault
}

func getMoveTestUserVault(t *testing.T, svc *ContactMoveService, userID, vaultID string) models.UserVault {
	t.Helper()
	var userVault models.UserVault
	if err := svc.db.First(&userVault, "user_id = ? AND vault_id = ?", userID, vaultID).Error; err != nil {
		t.Fatalf("Load user vault failed: %v", err)
	}
	return userVault
}

type contactMoveRecordingSearchEngine struct {
	contactVaults    map[string]string
	noteVaults       map[string]string
	deletedDocuments []string
	indexContactErr  error
	indexNoteErr     error
}

func (e *contactMoveRecordingSearchEngine) IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error {
	if e.indexContactErr != nil {
		return e.indexContactErr
	}
	e.contactVaults[id] = vaultID
	return nil
}

func (e *contactMoveRecordingSearchEngine) IndexNote(id string, vaultID, contactID, title, body string) error {
	if e.indexNoteErr != nil {
		return e.indexNoteErr
	}
	e.noteVaults[id] = vaultID
	return nil
}

func (e *contactMoveRecordingSearchEngine) DeleteDocument(id string) error {
	e.deletedDocuments = append(e.deletedDocuments, id)
	return nil
}

func (e *contactMoveRecordingSearchEngine) Search(vaultID, query string, limit, offset int) (*search.SearchResponse, error) {
	return &search.SearchResponse{}, nil
}

func (e *contactMoveRecordingSearchEngine) Rebuild() error {
	return nil
}

func (e *contactMoveRecordingSearchEngine) Close() error {
	return nil
}
