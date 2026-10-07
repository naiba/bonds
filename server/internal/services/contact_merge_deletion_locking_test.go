package services

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func awaitMergeBoundarySignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("production operation did not reach scheduling boundary")
	}
}
func awaitMergeBoundaryError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("operation did not finish")
		return nil
	}
}

func TestAvatarDeletionUsesParentLockBeforeMerge(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row lock order")
	}
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	storage := t.TempDir()
	files := NewVaultFileService(svc.db, storage)
	uploaded, err := files.Upload(vault, source.ID, user, "avatar", "synthetic.png", "image/png", 8, bytes.NewReader([]byte("synthetic")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactAvatarService(svc.db).UpdateAvatar(source.ID, vault, user, uploaded.ID); err != nil {
		t.Fatal(err)
	}
	var file models.File
	if err := svc.db.First(&file, uploaded.ID).Error; err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
	locked, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const queryHook = "acceptance:avatar_file_locked"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(queryHook, func(tx *gorm.DB) {
		if f, ok := tx.Statement.Dest.(*models.File); ok && f.ID == file.ID && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(locked)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(queryHook)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	deleted := make(chan error, 1)
	go func() {
		deleted <- NewVaultFileService(svc.db.WithContext(ctx), storage).DeleteContactPhoto(file.ID, source.ID, vault)
	}()
	awaitMergeBoundarySignal(t, locked)
	merged := make(chan error, 1)
	go func() {
		_, e := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vault, user, req)
		merged <- e
	}()
	waiting := waitForContactMergeLock(svc.db)
	close(resume)
	deleteErr, mergeErr := awaitMergeBoundaryError(t, deleted), awaitMergeBoundaryError(t, merged)
	var rows int64
	if err := svc.db.Model(&models.File{}).Where("id = ?", file.ID).Count(&rows).Error; err != nil {
		t.Fatal(err)
	}
	_, statErr := os.Stat(files.localPath(&file))
	bytesExist := statErr == nil
	t.Logf("file consistency after operations: rows=%d bytes_exist=%v", rows, bytesExist)
	if rows == 1 && !bytesExist {
		t.Fatal("database file survived but physical bytes lost")
	}
	if deleteErr != nil || !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
		t.Fatalf("delete=%v merge=%v", deleteErr, mergeErr)
	}
	if !waiting {
		t.Fatal("merge did not wait on the concurrent deletion")
	}
	if rows != 0 || !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("deleted avatar survived: rows=%d stat=%v", rows, statErr)
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	var survivor models.Contact
	if err := svc.db.First(&survivor, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if survivor.FileID != nil {
		t.Fatal("merge restored deleted avatar")
	}
	if err := files.DeleteContactPhoto(file.ID, source.ID, vault); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("deleted source accepted mutation: %v", err)
	}
}

func TestRelationshipDeletionUsesOrderedLocksBeforeMerge(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row lock order")
	}
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	neighbor, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Neighbor"})
	if err != nil {
		t.Fatal(err)
	}
	var kind models.RelationshipType
	if err := svc.db.Where("reverse_relationship_type_id IS NOT NULL").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewRelationshipService(svc.db).Create(source.ID, vault, user, dto.CreateRelationshipRequest{RelatedContactID: neighbor.ID, RelationshipTypeID: kind.ID}); err != nil {
		t.Fatal(err)
	}
	var relations []models.Relationship
	if err := svc.db.Order("id ASC").Find(&relations).Error; err != nil {
		t.Fatal(err)
	}
	if len(relations) != 2 {
		t.Fatalf("expected bidirectional pair, got %d", len(relations))
	}
	higher := relations[1]
	req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
	deletedFirst, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "acceptance:reverse_cleanup_paused"
	if err := svc.db.Callback().Delete().After("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.Relationship); ok && row.ID == higher.ID && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(deletedFirst)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Delete().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	deleted := make(chan error, 1)
	go func() {
		deleted <- NewRelationshipService(svc.db.WithContext(ctx)).Delete(higher.ID, higher.ContactID, vault)
	}()
	awaitMergeBoundarySignal(t, deletedFirst)
	merged := make(chan error, 1)
	go func() {
		_, e := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vault, user, req)
		merged <- e
	}()
	waiting := waitForContactMergeLock(svc.db)
	close(resume)
	deleteErr, mergeErr := awaitMergeBoundaryError(t, deleted), awaitMergeBoundaryError(t, merged)
	if deleteErr != nil || !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
		t.Fatalf("delete=%v merge=%v", deleteErr, mergeErr)
	}
	if !waiting {
		t.Fatal("merge did not wait on concurrent deletion")
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := svc.db.Model(&models.Relationship{}).Where("id IN ?", []uint{relations[0].ID, relations[1].ID}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted relationship pair restored: %d", count)
	}
}

// Wait for an actual competing row lock, whether the old implementation reaches
// a child first or the corrected implementation waits on the parent. Restrict
// observations to this test schema so concurrent packages cannot satisfy it.
func waitForContactMergeLock(db *gorm.DB) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := db.Raw(`SELECT COUNT(*) FROM pg_stat_activity activity
 WHERE activity.pid <> pg_backend_pid() AND activity.wait_event_type = 'Lock'
 AND (activity.query LIKE ? OR activity.query LIKE ? OR activity.query LIKE ?)
 AND EXISTS (SELECT 1 FROM pg_locks held WHERE held.pid = activity.pid
  AND held.relation IN ('contacts'::regclass, 'relationships'::regclass, 'files'::regclass))`, "%contacts%FOR UPDATE%", "%relationships%FOR UPDATE%", "%UPDATE%files%").Scan(&count).Error; err != nil {
			return false
		}
		if count > 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestRelationshipDeletionRollsBackBothDirectionsOnCleanupFailure(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	parentTypeID, _ := createAsymmetricTypePair(t, ctx.db, ctx.accountID)
	created, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{RelationshipTypeID: parentTypeID, RelatedContactID: ctx.relatedContactID})
	if err != nil {
		t.Fatal(err)
	}
	var pair []models.Relationship
	if err := ctx.db.Order("id ASC").Find(&pair).Error; err != nil {
		t.Fatal(err)
	}
	if len(pair) != 2 {
		t.Fatalf("fixture expected two directions: %d", len(pair))
	}
	hook := "relationship:reject_reverse_cleanup"
	injected := errors.New("reverse cleanup unavailable")
	if err := ctx.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*models.Relationship); ok && row.ID == 0 {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer ctx.db.Callback().Delete().Remove(hook)
	if err := ctx.svc.Delete(created.ID, ctx.contactID, ctx.vaultID); !errors.Is(err, injected) {
		t.Fatalf("cleanup error not returned: %v", err)
	}
	for _, original := range pair {
		var current models.Relationship
		if err := ctx.db.First(&current, original.ID).Error; err != nil {
			t.Fatalf("direction lost: %v", err)
		}
		if current.ContactID != original.ContactID || current.RelatedContactID != original.RelatedContactID || current.RelationshipTypeID != original.RelationshipTypeID {
			t.Fatal("failed deletion changed pair")
		}
	}
}

func TestContactPhotoDeletionRejectsMergedOwnerWithoutRemovingFile(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	files := NewVaultFileService(svc.db, t.TempDir())
	uploaded, err := files.Upload(vault, source.ID, user, "photo", "synthetic.png", "image/png", 9, bytes.NewReader([]byte("synthetic")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	if err := files.DeleteContactPhoto(uploaded.ID, source.ID, vault); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("stale photo path must retain its 404 domain error: %v", err)
	}
	var file models.File
	if err := svc.db.First(&file, uploaded.ID).Error; err != nil {
		t.Fatal(err)
	}
	if file.UfileableID == nil || *file.UfileableID != target.ID {
		t.Fatal("file ownership changed")
	}
	content, err := os.ReadFile(files.localPath(&file))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "synthetic" {
		t.Fatal("file bytes changed")
	}
}

func TestMergedAvatarCanBeDeletedWithEnforcedForeignKey(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Migrator().CreateConstraint(&models.Contact{}, "File"); err != nil {
		t.Fatal(err)
	}
	if svc.db.Dialector.Name() == "sqlite" {
		if err := svc.db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", target.ID).Update("file_id", 999999).Error; err == nil {
		t.Fatal("avatar FK was not enforced")
	}
	files := NewVaultFileService(svc.db, t.TempDir())
	uploaded, err := files.Upload(vault, source.ID, user, "avatar", "synthetic.png", "image/png", 9, bytes.NewReader([]byte("synthetic")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactAvatarService(svc.db).UpdateAvatar(source.ID, vault, user, uploaded.ID); err != nil {
		t.Fatal(err)
	}
	var file models.File
	if err := svc.db.First(&file, uploaded.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	if err := files.DeleteContactPhoto(file.ID, target.ID, vault); err != nil {
		t.Fatalf("merged avatar cannot be deleted: %v", err)
	}
	var count int64
	if err := svc.db.Model(&models.File{}).Where("id = ?", file.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("deleted file row remains")
	}
	if _, err := os.Stat(files.localPath(&file)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file bytes remain: %v", err)
	}
	var contacts []models.Contact
	if err := svc.db.Unscoped().Where("id IN ?", []string{source.ID, target.ID}).Find(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 2 {
		t.Fatal("contact identity history lost")
	}
	for _, contact := range contacts {
		if contact.FileID != nil {
			t.Fatal("avatar reference remains")
		}
		if contact.ID == source.ID && !contact.DeletedAt.Valid {
			t.Fatal("source resurrected")
		}
	}
}

func TestAvatarUploadRollsBackFileWhenProfileWriteFails(t *testing.T) {
	svc, vault, user, _ := setupContactTest(t)
	contact, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Avatar owner"})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	files := NewVaultFileService(svc.db, directory)
	files.SetFeedRecorder(NewFeedRecorder(svc.db))
	injected := errors.New("avatar profile write unavailable")
	hook := "avatar:reject_profile_reference"
	if err := svc.db.Callback().Update().Before("gorm:update").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contacts" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Update().Remove(hook)
	if _, err := files.UploadContactAvatar(contact.ID, vault, user, "synthetic.png", "image/png", 9, bytes.NewReader([]byte("synthetic"))); !errors.Is(err, injected) {
		t.Fatalf("profile failure not returned: %v", err)
	}
	var count int64
	if err := svc.db.Model(&models.File{}).Where("ufileable_id = ?", contact.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed avatar left a file row")
	}
	if err := svc.db.Model(&models.ContactFeedItem{}).Where("contact_id = ? AND action = ?", contact.ID, ActionFileUploaded).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed avatar recorded successful upload")
	}
	var stored models.Contact
	if err := svc.db.First(&stored, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.FileID != nil {
		t.Fatal("failed avatar changed profile")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("failed avatar left file bytes")
	}
}
