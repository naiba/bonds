package services

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type contentMutationContextKey struct{}

func waitForContentMergeLock(db *gorm.DB) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := db.Raw(`SELECT COUNT(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%FOR UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation IN ('contacts'::regclass,'activities'::regclass))`).Scan(&count).Error; err != nil {
			return false
		}
		if count > 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestContactMergeContentWaitsForMutation(t *testing.T) {
	for _, operation := range []string{"note_edit", "note_delete", "note_create", "activity_edit", "activity_delete", "activity_create"} {
		t.Run(operation, func(t *testing.T) {
			f := setupContactMergeContent(t)
			if f.svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL content/parent row locking")
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			locked, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "content_merge:pause_writer"
			if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Error != nil || tx.Statement.Context.Value(contentMutationContextKey{}) != true {
					return
				}
				if _, ok := tx.Statement.Clauses["FOR"]; !ok {
					return
				}
				if paused.CompareAndSwap(false, true) {
					close(locked)
					select {
					case <-resume:
					case <-time.After(15 * time.Second):
						tx.AddError(errors.New("content writer barrier timed out"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.svc.db.Callback().Query().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			written := make(chan error, 1)
			go func() {
				db := f.svc.db.WithContext(context.WithValue(ctx, contentMutationContextKey{}, true))
				notes, activities := NewNoteService(db), NewActivityService(db)
				var err error
				switch operation {
				case "note_edit":
					_, err = notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: "A memory alone"})
				case "note_delete":
					err = notes.Delete(f.note.ID, f.owner, f.vault)
				case "note_create":
					_, err = notes.Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: "New " + f.body})
				case "activity_edit":
					_, err = activities.UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest("A memory alone"))
				case "activity_delete":
					err = activities.Delete(f.vault, f.activity.ID)
				case "activity_create":
					_, err = activities.CreateForUser(f.vault, f.user, f.activityRequest("New "+f.body))
				}
				written <- err
			}()
			awaitMergeBoundarySignal(t, locked)
			merged := make(chan error, 1)
			go func() {
				_, err := NewContactService(f.svc.db.WithContext(ctx)).MergeContacts(f.vault, f.user, request)
				merged <- err
			}()
			waited := waitForContentMergeLock(f.svc.db)
			close(resume)
			writeErr, mergeErr := awaitMergeBoundaryError(t, written), awaitMergeBoundaryError(t, merged)
			if writeErr != nil || !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
				t.Fatalf("write=%v merge=%v", writeErr, mergeErr)
			}
			if !waited {
				t.Fatal("merge did not wait for the real writer lock")
			}
			var source models.Contact
			if err := f.svc.db.First(&source, "id = ?", f.source).Error; err != nil {
				t.Fatal("rejected merge removed source", err)
			}
			if _, err := f.svc.MergeContacts(f.vault, f.user, reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)); err != nil {
				t.Fatal(err)
			}
			var notes []models.Note
			var activities []models.Activity
			if err := f.svc.db.Where("vault_id = ?", f.vault).Find(&notes).Error; err != nil {
				t.Fatal(err)
			}
			if err := f.svc.db.Where("vault_id = ?", f.vault).Find(&activities).Error; err != nil {
				t.Fatal(err)
			}
			wantNotes, wantActivities := 1, 1
			switch operation {
			case "note_delete":
				wantNotes = 0
			case "note_create":
				wantNotes = 2
			case "activity_delete":
				wantActivities = 0
			case "activity_create":
				wantActivities = 2
			}
			if len(notes) != wantNotes || len(activities) != wantActivities {
				t.Fatalf("lost/resurrected records: notes=%d activities=%d", len(notes), len(activities))
			}
			for _, note := range notes {
				want := strings.ReplaceAll(f.body, f.source, f.target)
				if operation == "note_edit" {
					want = "A memory alone"
				}
				if operation == "note_create" && note.ID != f.note.ID {
					want = "New " + want
				}
				if note.Body != want || note.ContactID != f.owner {
					t.Fatalf("note mutation lost: %+v", note)
				}
			}
			for _, activity := range activities {
				want := strings.ReplaceAll(f.body, f.source, f.target)
				if operation == "activity_edit" {
					want = "A memory alone"
				}
				if operation == "activity_create" && activity.ID != f.activity.ID {
					want = "New " + want
				}
				if ptrToStr(activity.Description) != want {
					t.Fatalf("activity mutation lost: %+v", activity)
				}
			}
		})
	}
}

func TestContentWriterRejectsConcurrentMergedMention(t *testing.T) {
	for _, operation := range []string{"note_edit", "note_create", "activity_edit", "activity_create"} {
		t.Run(operation, func(t *testing.T) {
			f := setupContactMergeContent(t)
			if f.svc.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL content/parent row locking")
			}
			request := reviewedContactMerge(t, f.svc, f.vault, f.user, f.target, f.source)
			locked, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			hook := "content_merge:pause_merge"
			if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Error != nil || tx.Statement.Context.Value(contentMutationContextKey{}) != true {
					return
				}
				if _, ok := tx.Statement.Dest.(*[]models.Contact); !ok {
					return
				}
				if _, ok := tx.Statement.Clauses["FOR"]; !ok {
					return
				}
				if paused.CompareAndSwap(false, true) {
					close(locked)
					select {
					case <-resume:
					case <-time.After(15 * time.Second):
						tx.AddError(errors.New("content merge barrier timed out"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.svc.db.Callback().Query().Remove(hook)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			merged := make(chan error, 1)
			go func() {
				_, err := NewContactService(f.svc.db.WithContext(context.WithValue(ctx, contentMutationContextKey{}, true))).MergeContacts(f.vault, f.user, request)
				merged <- err
			}()
			awaitMergeBoundarySignal(t, locked)
			written := make(chan error, 1)
			go func() {
				notes, activities := NewNoteService(f.svc.db.WithContext(ctx)), NewActivityService(f.svc.db.WithContext(ctx))
				var err error
				switch operation {
				case "note_edit":
					_, err = notes.Update(f.note.ID, f.owner, f.vault, dto.UpdateNoteRequest{Body: f.body})
				case "note_create":
					_, err = notes.Create(f.owner, f.vault, f.user, dto.CreateNoteRequest{Body: f.body})
				case "activity_edit":
					_, err = activities.UpdateForUser(f.vault, f.user, f.activity.ID, f.activityRequest(f.body))
				case "activity_create":
					_, err = activities.CreateForUser(f.vault, f.user, f.activityRequest(f.body))
				}
				written <- err
			}()
			waited := waitForContentMergeLock(f.svc.db)
			close(resume)
			mergeErr, writeErr := awaitMergeBoundaryError(t, merged), awaitMergeBoundaryError(t, written)
			if mergeErr != nil || !errors.Is(writeErr, ErrContactNotFound) {
				t.Fatalf("merge=%v writer=%v", mergeErr, writeErr)
			}
			if !waited {
				t.Fatal("writer did not wait for merging contact")
			}
			var notes []models.Note
			var activities []models.Activity
			if err := f.svc.db.Where("vault_id = ?", f.vault).Find(&notes).Error; err != nil {
				t.Fatal(err)
			}
			if err := f.svc.db.Where("vault_id = ?", f.vault).Find(&activities).Error; err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(f.body, f.source, f.target)
			if len(notes) != 1 || notes[0].Body != want || len(activities) != 1 || ptrToStr(activities[0].Description) != want {
				t.Fatal("stale writer restored source or created a record")
			}
		})
	}
}

func TestActivityOmittedFormatCannotRestoreForeignParticipants(t *testing.T) {
	f := setupContactMergeContent(t)
	if f.svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL concurrent format update")
	}
	foreign, err := NewVaultService(f.svc.db).CreateVault(f.account, f.user, dto.CreateVaultRequest{Name: "Other space"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.db.Model(&models.Contact{}).Where("id = ?", f.source).Update("vault_id", foreign.ID).Error; err != nil {
		t.Fatal(err)
	}
	locked, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	hook := "activity_format:pause_initial_read"
	if err := f.svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Context.Value(contentMutationContextKey{}) != true {
			return
		}
		if _, ok := tx.Statement.Dest.(*models.Activity); !ok {
			return
		}
		if _, ok := tx.Statement.Clauses["FOR"]; ok {
			return
		}
		if paused.CompareAndSwap(false, true) {
			close(locked)
			select {
			case <-resume:
			case <-time.After(15 * time.Second):
				tx.AddError(errors.New("format barrier timed out"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer f.svc.db.Callback().Query().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	completed := make(chan error, 1)
	go func() {
		req := f.activityRequest("`" + f.body + "`")
		req.DescriptionFormat = ""
		req.ParticipantIDs = nil
		_, err := NewActivityService(f.svc.db.WithContext(context.WithValue(ctx, contentMutationContextKey{}, true))).UpdateForUser(f.vault, f.user, f.activity.ID, req)
		completed <- err
	}()
	awaitMergeBoundarySignal(t, locked)
	// Preserve a historical foreign link, while explicitly retaining no participants.
	changed := f.activityRequest(f.body)
	changed.DescriptionFormat = "plain"
	_, changeErr := NewActivityService(f.svc.db).UpdateForUser(f.vault, f.user, f.activity.ID, changed)
	close(resume)
	writeErr := awaitMergeBoundaryError(t, completed)
	if changeErr != nil {
		t.Fatal(changeErr)
	}
	if !errors.Is(writeErr, ErrContactNotFound) {
		t.Errorf("stale format writer accepted foreign participant: %v", writeErr)
	}
	var count int64
	if err := f.svc.db.Model(&models.ActivityParticipant{}).Where("activity_id = ?", f.activity.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cross-vault participant persisted: %d", count)
	}
	stored, err := NewActivityService(f.svc.db).Get(f.vault, f.user, f.activity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Description != f.body || stored.DescriptionFormat != "plain" {
		t.Fatal("rejected write changed stored activity")
	}
}
