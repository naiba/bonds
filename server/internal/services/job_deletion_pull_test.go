package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestJobDeletionDuringSubscriptionPull(t *testing.T) {
	for _, entry := range []string{"job", "employee"} {
		t.Run(entry, func(t *testing.T) {
			syncSvc, _, vcardSvc, vault, user, account := setupDavSyncTest(t)
			db := syncSvc.db
			if db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL row-lock interleaving; serial job lifecycle runs on both databases")
			}
			dbctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			db = db.WithContext(dbctx)
			syncSvc.db = db
			vcardSvc.db = db
			contact, err := NewContactService(db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Before", LastName: "Name"})
			if err != nil {
				t.Fatal(err)
			}
			company, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Local company"})
			if err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(db)
			job, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Deleted position"})
			if err != nil {
				t.Fatal(err)
			}
			path := fmt.Sprintf("/dav/addressbooks/%s/%s/%s.vcf", user, vault, contact.ID)
			original, err := vcardSvc.ExportContactToVCard(contact.ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			original.SetName(&vcard.Name{GivenName: "Edited", FamilyName: "Name"})
			original.SetValue(vcard.FieldFormattedName, "Edited Name")
			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			putDone, deleteDone := make(chan error, 1), make(chan error, 1)
			putFinished, deleteStarted, deleteFinished := false, false, false
			const callback = "pause_subscription_pull_after_contact_save"
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
			subscription := models.AddressBookSubscription{VaultID: vault, CreatedByUserID: user, URI: "https://synthetic.invalid", Username: "synthetic", Password: "synthetic", Capabilities: "addressbook"}
			if err := db.Create(&subscription).Error; err != nil {
				t.Fatal(err)
			}
			if err := upsertContactSubscriptionState(db, contact.ID, subscription.ID, path, "before-pull"); err != nil {
				t.Fatal(err)
			}
			go func() {
				result := dto.TriggerSyncResponse{}
				syncSvc.upsertFromObject(carddav.AddressObject{Path: path, ETag: "accepted-pull", Card: original}, subscription.ID, vault, user, account, nil, &result)
				if result.Errors != 0 || result.Updated != 1 {
					putDone <- fmt.Errorf("pull result: %+v", result)
				} else {
					putDone <- nil
				}
			}()
			select {
			case <-entered:
			case err := <-putDone:
				putFinished = true
				t.Fatalf("PUT did not reach contact save: %v", err)
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
				t.Log("deletion committed while subscription pull holds contact lock")
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
			exported, err := vcardSvc.ExportContactToVCard(contact.ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			var state models.ContactSubscriptionState
			if err := db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, subscription.ID).First(&state).Error; err != nil {
				t.Fatal(err)
			}
			if state.DistantEtag != "accepted-pull" || state.DistantURI != path || stored.DistantEtag == nil || *stored.DistantEtag != "accepted-pull" {
				t.Fatal("pull state was not persisted")
			}
			legacy := ""
			if stored.JobPosition != nil {
				legacy = *stored.JobPosition
			}
			t.Logf("persisted jobs=%d legacy=%q exported TITLE=%q", count, legacy, exported.Value(vcard.FieldTitle))
			if legacy != "" || exported.Value(vcard.FieldTitle) != "" {
				t.Fatal("deleted job title was resurrected by unrelated pulled name edit")
			}
		})
	}
}

func TestJobDeletionRejectsForeignContact(t *testing.T) {
	svc, vault, user, account := setupContactTest(t)
	company, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Local company"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewVaultService(svc.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Other vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	contact, err := svc.CreateContact(other.ID, user, dto.CreateContactRequest{FirstName: "Protected employee"})
	if err != nil {
		t.Fatal(err)
	}
	// Historical invalid cross-vault references must not bypass ownership checks.
	job := models.ContactCompany{CompanyID: company.ID, ContactID: contact.ID, JobPosition: strPtrOrNil("Protected role")}
	if err := svc.db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	if err := jobs.Delete(contact.ID, vault, job.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("job delete: %v", err)
	}
	if err := jobs.RemoveEmployee(company.ID, vault, contact.ID); !errors.Is(err, ErrContactJobNotFound) {
		t.Fatalf("employee delete: %v", err)
	}
	var stored models.ContactCompany
	if err := svc.db.First(&stored, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ContactID != contact.ID || stored.CompanyID != company.ID || ptrToStr(stored.JobPosition) != "Protected role" {
		t.Fatal("rejected deletion modified foreign job")
	}
}
