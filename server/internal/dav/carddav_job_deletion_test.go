package dav

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestJobDeletionDuringConditionalCardDAVPut(t *testing.T) {
	for _, entry := range []string{"job", "employee"} {
		t.Run(entry, func(t *testing.T) {
			backend, db, ctx, vault, user := setupCardDAVTest(t)
			if db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL row-lock interleaving; serial job lifecycle runs on both databases")
			}
			dbctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			db = db.WithContext(dbctx)
			backend.db = db
			contact := createTestContact(t, db, vault, user, "Before", "Name")
			company, err := services.NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Local company"})
			if err != nil {
				t.Fatal(err)
			}
			jobs := services.NewContactJobService(db)
			job, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Deleted position"})
			if err != nil {
				t.Fatal(err)
			}
			path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, contact.ID)
			original, err := backend.GetAddressObject(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			original.Card.SetName(&vcard.Name{GivenName: "Edited", FamilyName: "Name"})
			original.Card.SetValue(vcard.FieldFormattedName, "Edited Name")
			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			putDone, deleteDone := make(chan error, 1), make(chan error, 1)
			putFinished, deleteStarted, deleteFinished := false, false, false
			const callback = "pause_conditional_put_after_contact_save"
			if err := db.Callback().Update().After("gorm:update").Register(callback, func(tx *gorm.DB) {
				if c, ok := tx.Statement.Dest.(*models.Contact); ok && c.ID == contact.ID {
					close(entered)
					<-resume
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer func() {
				release()
				if !putFinished {
					<-putDone
				}
				if deleteStarted && !deleteFinished {
					<-deleteDone
				}
				db.Callback().Update().Remove(callback)
			}()
			go func() {
				_, err := backend.PutAddressObject(ctx, path, original.Card, &carddav.PutAddressObjectOptions{IfMatch: webdav.ConditionalMatch(`"` + original.ETag + `"`)})
				putDone <- err
			}()
			select {
			case <-entered:
			case err := <-putDone:
				putFinished = true
				t.Fatalf("PUT did not reach post-IfMatch save: %v", err)
			case <-dbctx.Done():
				t.Fatal(dbctx.Err())
			}
			deleteStarted = true
			go func() {
				if entry == "job" {
					deleteDone <- jobs.Delete(contact.ID, vault, job.ID)
				} else {
					deleteDone <- jobs.RemoveEmployee(company.ID, vault, contact.ID)
				}
			}()
			select {
			case err := <-deleteDone:
				deleteFinished = true
				if err != nil {
					t.Fatal(err)
				}
				t.Log("deletion committed while conditional PUT holds contact lock")
			case <-time.After(time.Second):
				t.Log("delete waiting for PUT")
			}
			release()
			err = <-putDone
			putFinished = true
			if err != nil {
				t.Fatal(err)
			}
			if !deleteFinished {
				err = <-deleteDone
				deleteFinished = true
				if err != nil {
					t.Fatal(err)
				}
			}
			var stored models.Contact
			if err := db.First(&stored, "id = ?", contact.ID).Error; err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("job deletion did not persist: %d", count)
			}
			if stored.FirstName == nil || *stored.FirstName != "Edited" {
				t.Fatal("unrelated name edit did not persist")
			}
			exported, err := backend.GetAddressObject(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			legacy := ""
			if stored.JobPosition != nil {
				legacy = *stored.JobPosition
			}
			t.Logf("persisted jobs=%d legacy=%q exported TITLE=%q", count, legacy, exported.Card.Value(vcard.FieldTitle))
			if legacy != "" || exported.Card.Value(vcard.FieldTitle) != "" {
				t.Fatal("deleted job title was resurrected by unrelated conditional name edit")
			}
		})
	}
}
