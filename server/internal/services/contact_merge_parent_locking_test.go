package services

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestContactDeletionLocksIntroducerOwnersBeforeMerge(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		name := "single"
		if bulk {
			name = "bulk"
		}
		t.Run(name, func(t *testing.T) {
			svc, vault, user, _ := setupContactTest(t)
			if svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL contact row lock order")
			}
			contacts := []*dto.ContactResponse{}
			for _, name := range []string{"One", "Two", "Three"} {
				c, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: name})
				if err != nil {
					t.Fatal(err)
				}
				contacts = append(contacts, c)
			}
			sort.Slice(contacts, func(i, j int) bool { return contacts[i].ID < contacts[j].ID })
			owner, target, source := contacts[0], contacts[1], contacts[2]
			if _, err := svc.UpdateContact(owner.ID, vault, user, dto.UpdateContactRequest{FirstName: owner.FirstName, FirstMetThroughContactID: &source.ID}); err != nil {
				t.Fatal(err)
			}
			req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
			locked, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "contact_delete:pause_locked_source"
			if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				rows, ok := tx.Statement.Dest.(*[]models.Contact)
				if !ok || tx.Error != nil {
					return
				}
				if _, ok := tx.Statement.Clauses["FOR"]; !ok {
					return
				}
				for _, row := range *rows {
					if row.ID == source.ID && paused.CompareAndSwap(false, true) {
						close(locked)
						select {
						case <-resume:
						case <-time.After(10 * time.Second):
							tx.AddError(errors.New("contact deletion scheduling timeout"))
						}
						return
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer svc.db.Callback().Query().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			deleted := make(chan error, 1)
			go func() {
				deleting := NewContactService(svc.db.WithContext(ctx))
				if bulk {
					_, err := deleting.DeleteContacts([]string{source.ID}, vault)
					deleted <- err
				} else {
					deleted <- deleting.DeleteContact(source.ID, vault)
				}
			}()
			awaitMergeBoundarySignal(t, locked)
			merged := make(chan error, 1)
			go func() {
				_, err := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vault, user, req)
				merged <- err
			}()
			waited := waitForContactMergeLock(svc.db)
			close(resume)
			deleteErr, mergeErr := awaitMergeBoundaryError(t, deleted), awaitMergeBoundaryError(t, merged)
			if deleteErr != nil || !errors.Is(mergeErr, ErrContactNotFound) {
				t.Fatalf("delete=%v merge=%v", deleteErr, mergeErr)
			}
			if !waited {
				t.Fatal("merge did not wait for contact deletion")
			}
			var stored models.Contact
			if err := svc.db.First(&stored, "id = ?", owner.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.FirstMetThroughContactID != nil {
				t.Fatal("deletion left an introducer reference")
			}
			var sourceCount, targetCount int64
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&sourceCount).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", target.ID).Count(&targetCount).Error; err != nil {
				t.Fatal(err)
			}
			if sourceCount != 0 || targetCount != 1 {
				t.Fatalf("unexpected contacts: source=%d target=%d", sourceCount, targetCount)
			}
		})
	}
}

func TestActivityRemovalUsesRowLockBeforeParticipants(t *testing.T) {
	for _, move := range []bool{false, true} {
		name := "delete"
		if move {
			name = "move_participant_vs_delete"
		}
		t.Run(name, func(t *testing.T) {
			svc, vault, user, account := setupContactTest(t)
			if svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL activity row lock order")
			}
			target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Survivor"})
			if err != nil {
				t.Fatal(err)
			}
			source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Payer"})
			if err != nil {
				t.Fatal(err)
			}
			participant := source
			if move {
				participant, err = svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Participant"})
				if err != nil {
					t.Fatal(err)
				}
			}
			// The migration preserves historical payer IDs separately from participants.
			event := models.Activity{VaultID: vault, Title: "Legacy paid activity", PaidByContactID: &source.ID}
			if err := svc.db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Create(&models.ActivityParticipant{ActivityID: event.ID, ContactID: participant.ID}).Error; err != nil {
				t.Fatal(err)
			}
			destination, err := NewVaultService(svc.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Destination"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
			removed, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "activity:pause_participant_removal"
			if err := svc.db.Callback().Delete().After("gorm:delete").Register(hook, func(tx *gorm.DB) {
				if tx.Statement.Table == "activity_participants" && tx.Error == nil && paused.CompareAndSwap(false, true) {
					close(removed)
					select {
					case <-resume:
					case <-time.After(10 * time.Second):
						tx.AddError(errors.New("activity deletion scheduling timeout"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer svc.db.Callback().Delete().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			deleted := make(chan error, 1)
			go func() {
				db := svc.db.WithContext(ctx)
				if move {
					_, err := NewContactMoveService(db).MoveMany([]string{participant.ID}, vault, destination.ID, user)
					deleted <- err
				} else {
					deleted <- NewActivityService(db).Delete(vault, event.ID)
				}
			}()
			awaitMergeBoundarySignal(t, removed)
			merged := make(chan error, 1)
			go func() {
				if move {
					merged <- NewActivityService(svc.db.WithContext(ctx)).Delete(vault, event.ID)
					return
				}
				_, err := NewContactService(svc.db.WithContext(ctx)).MergeContacts(vault, user, req)
				merged <- err
			}()
			waited := false
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				var count int64
				if err := svc.db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid <> pg_backend_pid() AND a.wait_event_type = 'Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation IN ('activities'::regclass,'activity_participants'::regclass))`).Scan(&count).Error; err != nil {
					close(resume)
					t.Fatal(err)
				}
				if count > 0 {
					waited = true
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			close(resume)
			deleteErr, mergeErr := awaitMergeBoundaryError(t, deleted), awaitMergeBoundaryError(t, merged)
			wantError := ErrContactMergeReviewChanged
			if move {
				wantError = ErrActivityNotFound
			}
			if deleteErr != nil || !errors.Is(mergeErr, wantError) {
				t.Fatalf("remove=%v competing operation=%v", deleteErr, mergeErr)
			}
			if !waited {
				t.Fatal("merge did not wait on activity removal")
			}
			if _, err := svc.MergeContacts(vault, user, reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)); err != nil {
				t.Fatal(err)
			}
			var rows, pivots int64
			if err := svc.db.Model(&models.Activity{}).Where("id = ?", event.ID).Count(&rows).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.ActivityParticipant{}).Where("activity_id = ?", event.ID).Count(&pivots).Error; err != nil {
				t.Fatal(err)
			}
			if rows != 0 || pivots != 0 {
				t.Fatalf("removed activity restored: %d/%d", rows, pivots)
			}
			if move {
				var stored models.Contact
				if err := svc.db.First(&stored, "id = ?", participant.ID).Error; err != nil {
					t.Fatal(err)
				}
				if stored.VaultID != destination.ID {
					t.Fatal("participant move lost")
				}
			}
		})
	}
}
