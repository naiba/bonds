package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestRelationshipCreationRecordsReverseFeedWithForeignKey(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL foreign-key row-lock interaction")
	}
	first, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "First"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Exec("ALTER TABLE contact_feed_items ADD CONSTRAINT feed_contact_ownership FOREIGN KEY (contact_id) REFERENCES contacts(id)").Error; err != nil {
		t.Fatal(err)
	}
	var kind models.RelationshipType
	if err := svc.db.Where("name_reverse_relationship IS NOT NULL").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	if _, found := findReverseTypeID(svc.db, kind.ID); !found {
		t.Fatal("fixture does not have reverse relationship type")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	db := svc.db.WithContext(ctx)
	relations := NewRelationshipService(db)
	relations.SetFeedRecorder(NewFeedRecorder(db))
	if _, err := relations.Create(first.ID, vaultID, userID, dto.CreateRelationshipRequest{RelatedContactID: second.ID, RelationshipTypeID: kind.ID}); err != nil {
		t.Fatalf("bidirectional relationship creation blocked or failed: %v", err)
	}
	for _, id := range []string{first.ID, second.ID} {
		var rows, feeds int64
		if err := svc.db.Model(&models.Relationship{}).Where("contact_id = ?", id).Count(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if err := svc.db.Model(&models.ContactFeedItem{}).Where("contact_id = ? AND action = ?", id, ActionRelationshipAdded).Count(&feeds).Error; err != nil {
			t.Fatal(err)
		}
		if rows != 1 || feeds != 1 {
			t.Fatalf("contact %s has %d relationships and %d feed records", id, rows, feeds)
		}
	}
}

func TestMergeLocksIncomingOwnerAgainstVaultMove(t *testing.T) {
	svc, vaultID, moverID, accountID := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent ownership authorization")
	}
	other, err := NewVaultService(svc.db).CreateVault(accountID, moverID, dto.CreateVaultRequest{Name: "Private"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	actor := models.User{AccountID: accountID, Email: "merge-editor@example.test"}
	if err := svc.db.Create(&actor).Error; err != nil {
		t.Fatal(err)
	}
	for _, membership := range []models.UserVault{{UserID: actor.ID, VaultID: vaultID, Permission: models.PermissionEditor}, {UserID: actor.ID, VaultID: other.ID, Permission: models.PermissionViewer}} {
		if err := svc.db.Create(&membership).Error; err != nil {
			t.Fatal(err)
		}
	}
	target, err := svc.CreateContact(vaultID, moverID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, moverID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := svc.CreateContact(vaultID, moverID, dto.CreateContactRequest{FirstName: "Incoming owner"})
	if err != nil {
		t.Fatal(err)
	}
	var kind models.RelationshipType
	if err := svc.db.First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	relation := models.Relationship{ContactID: owner.ID, RelatedContactID: source.ID, RelationshipTypeID: kind.ID}
	if err := svc.db.Create(&relation).Error; err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, actor.ID, target.ID, source.ID)
	loaded, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_incoming_owner_read"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		owners, ok := tx.Statement.Dest.(*[]models.Contact)
		if !ok || tx.Error != nil {
			return
		}
		for _, contact := range *owners {
			if contact.ID == owner.ID && paused.CompareAndSwap(false, true) {
				close(loaded)
				select {
				case <-resume:
				case <-time.After(10 * time.Second):
					tx.AddError(errors.New("test scheduling timeout"))
				}
				return
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	merged := make(chan error, 1)
	go func() {
		_, err := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vaultID, actor.ID, req)
		merged <- err
	}()
	select {
	case <-loaded:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("incoming owner not read")
	}
	moved := make(chan error, 1)
	go func() {
		_, err := NewContactMoveService(svc.db.WithContext(ctx)).MoveMany([]string{owner.ID}, vaultID, other.ID, moverID)
		moved <- err
	}()
	moveFinished := false
	var moveErr error
	select {
	case moveErr = <-moved:
		moveFinished = true
	case <-time.After(250 * time.Millisecond):
	}
	close(resume)
	var mergeErr error
	select {
	case mergeErr = <-merged:
	case <-time.After(10 * time.Second):
		t.Fatal("merge did not finish")
	}
	if !moveFinished {
		select {
		case moveErr = <-moved:
		case <-time.After(10 * time.Second):
			t.Fatal("move did not finish")
		}
	}
	if moveErr != nil {
		t.Fatalf("move: %v", moveErr)
	}
	if moveFinished && mergeErr == nil {
		t.Fatal("merge rewrote a relationship after its owner moved into a viewer-only vault")
	}
	if !moveFinished && mergeErr != nil {
		t.Fatalf("merge should finish before the waiting move: %v", mergeErr)
	}
	var persisted models.Relationship
	if err := svc.db.First(&persisted, relation.ID).Error; err != nil {
		t.Fatal(err)
	}
	want := target.ID
	if moveFinished {
		want = source.ID
	}
	if persisted.RelatedContactID != want {
		t.Fatalf("reference=%s want=%s", persisted.RelatedContactID, want)
	}
}

func TestImportantDateReminderUpdateUsesParentLockBeforeMerge(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL contact foreign-key lock ordering")
	}
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	day, month := 1, 1
	date, err := NewImportantDateService(svc.db).Create(source.ID, vaultID, dto.CreateImportantDateRequest{Label: "Anniversary", Day: &day, Month: &month, DatePrecision: "month_day"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Exec("ALTER TABLE contact_reminders ADD CONSTRAINT reminder_contact_ownership FOREIGN KEY (contact_id) REFERENCES contacts(id)").Error; err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	updated, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_date_before_reminder"
	if err := svc.db.Callback().Update().After("gorm:update").Register(hook, func(tx *gorm.DB) {
		row, ok := tx.Statement.Dest.(*models.ContactImportantDate)
		if ok && row.ID == date.ID && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(updated)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Update().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	edited := make(chan error, 1)
	go func() {
		enabled := true
		_, err := NewImportantDateService(svc.db.WithContext(ctx)).Update(date.ID, source.ID, vaultID, dto.UpdateImportantDateRequest{Label: "Anniversary", Day: &day, Month: &month, DatePrecision: "month_day", RemindMe: &enabled})
		edited <- err
	}()
	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("date update was not observed")
	}
	merged := make(chan error, 1)
	go func() {
		_, err := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vaultID, userID, req)
		merged <- err
	}()
	// Give the competing merge a chance to acquire its parent lock. Without the
	// editor's parent lock this forms a child -> parent / parent -> child cycle.
	time.Sleep(150 * time.Millisecond)
	close(resume)
	select {
	case err := <-edited:
		if err != nil {
			t.Fatalf("date reminder edit deadlocked or failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("edit did not finish")
	}
	select {
	case err := <-merged:
		if !errors.Is(err, ErrContactMergeReviewChanged) {
			t.Fatalf("merge must re-review the new reminder: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("merge did not finish")
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	var reminder models.ContactReminder
	if err := svc.db.Where("important_date_id = ?", date.ID).First(&reminder).Error; err != nil {
		t.Fatal(err)
	}
	if reminder.ContactID != target.ID {
		t.Fatalf("reminder left behind: %+v", reminder)
	}
}

func TestImportantDateDeletionUsesParentLockBeforeMerge(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent date/reminder row locks")
	}
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	day, month := 1, 1
	enabled := true
	date, err := NewImportantDateService(svc.db).Create(source.ID, vaultID, dto.CreateImportantDateRequest{Label: "Anniversary", Day: &day, Month: &month, DatePrecision: "month_day", RemindMe: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	deleting, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "merge:pause_date_reminder_deletion"
	if err := svc.db.Callback().Delete().After("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_reminders" && tx.Error == nil && paused.CompareAndSwap(false, true) {
			close(deleting)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Delete().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	deleted := make(chan error, 1)
	go func() {
		deleted <- NewImportantDateService(svc.db.WithContext(ctx)).Delete(date.ID, source.ID, vaultID)
	}()
	select {
	case <-deleting:
	case <-time.After(10 * time.Second):
		close(resume)
		t.Fatal("reminder deletion not observed")
	}
	merged := make(chan error, 1)
	go func() {
		_, err := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vaultID, userID, req)
		merged <- err
	}()
	time.Sleep(150 * time.Millisecond)
	close(resume)
	select {
	case err := <-deleted:
		if err != nil {
			t.Fatalf("date deletion deadlocked or failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("delete did not finish")
	}
	select {
	case err := <-merged:
		if !errors.Is(err, ErrContactMergeReviewChanged) {
			t.Fatalf("merge must review deleted date: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("merge did not finish")
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	var dates, reminders int64
	if err := svc.db.Model(&models.ContactImportantDate{}).Where("id = ?", date.ID).Count(&dates).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.ContactReminder{}).Where("important_date_id = ?", date.ID).Count(&reminders).Error; err != nil {
		t.Fatal(err)
	}
	if dates != 0 || reminders != 0 {
		t.Fatalf("deleted date/reminder restored: %d/%d", dates, reminders)
	}
}
