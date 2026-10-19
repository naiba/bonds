package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardEmptyOrganizationSubscriptionPersistsTitle(t *testing.T) {
	for _, title := range []string{"Architect", ""} {
		t.Run("title="+title, func(t *testing.T) {
			svc, _, vault, user := setupVCardTest(t)
			var owner models.User
			if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Artist\r\nTITLE:Artist\r\nEND:VCARD\r\n")).Decode()
			if err != nil {
				t.Fatal(err)
			}
			id := ""
			apply := func(etag string) string {
				t.Helper()
				action := ""
				err := svc.db.Transaction(func(tx *gorm.DB) error {
					var err error
					id, action, err = svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, id, "/synthetic-artist.vcf", etag, nil)
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				return action
			}
			if got := apply("first"); got != "created" {
				t.Fatal(got)
			}
			card.SetValue(vcard.FieldOrganization, "")
			card.SetValue(vcard.FieldTitle, title)
			if got := apply("revised"); got != "updated" {
				t.Fatal(got)
			}
			var c models.Contact
			if err := svc.db.First(&c, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(c.JobPosition) != title {
				t.Errorf("accepted subscription update persisted TITLE=%q, want=%q; ETag=%q", ptrToStr(c.JobPosition), title, ptrToStr(c.DistantEtag))
			}
			if ptrToStr(c.DistantEtag) != "revised" || (title == "" && c.JobPosition != nil) {
				t.Fatal("accepted update did not persist etag or SQL NULL")
			}
			before := c
			if got := apply("revised"); got != "skipped" {
				t.Fatal(got)
			}
			var after models.Contact
			if err := svc.db.First(&after, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("same etag retry changed persisted contact")
			}
			exported, err := svc.ExportContactToVCard(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if exported.Value(vcard.FieldTitle) != title {
				t.Fatal("export disagrees with accepted subscription")
			}
			var count int64
			if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("empty ORG created employment")
			}

		})
	}
}

func TestVCardEmptyOrganizationPreservesAssociatedJobs(t *testing.T) {
	for _, kind := range []string{"legacy company", "normal Jobs"} {
		t.Run(kind, func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			var owner models.User
			if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
				t.Fatal(err)
			}
			company := models.Company{VaultID: vault, Name: "Existing employer"}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			if kind == "normal Jobs" {
				if _, err := NewContactJobService(svc.db).LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"}); err != nil {
					t.Fatal(err)
				}
			} else if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": company.ID, "job_position": "Engineer"}).Error; err != nil {
				t.Fatal(err)
			}
			var before, after models.Contact
			var beforeJobs, afterJobs []models.ContactCompany
			read := func(c *models.Contact, j *[]models.ContactCompany) {
				t.Helper()
				if err := svc.db.First(c, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := svc.db.Where("contact_id = ?", id).Order("id").Find(j).Error; err != nil {
					t.Fatal(err)
				}
			}
			read(&before, &beforeJobs)
			card := vcard.Card{vcard.FieldOrganization: []*vcard.Field{vcard.NewOrganization([]string{"", "Department"})}, vcard.FieldTitle: []*vcard.Field{{Value: "Changed"}}}
			err := svc.db.Transaction(func(tx *gorm.DB) error {
				_, _, err := svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, id, "/existing-employment.vcf", "must-not-advance", nil)
				return err
			})
			if !errors.Is(err, ErrVCardUnsafeReplacement) {
				t.Fatalf("empty ORG should protect jobs: %v", err)
			}
			read(&after, &afterJobs)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeJobs, afterJobs) {
				t.Fatal("rejected subscription changed jobs, contact or etag")
			}
		})
	}
}

func TestVCardEmptyOrganizationIncrementalImportPreservesOmittedTitle(t *testing.T) {
	svc, id, vault, user := setupVCardTest(t)
	var owner models.User
	if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactJobService(svc.db).LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Independent title"}); err != nil {
		t.Fatal(err)
	}
	card := vcard.Card{vcard.FieldOrganization: []*vcard.Field{vcard.NewOrganization([]string{"", "Department"})}}
	if err := svc.db.Transaction(func(tx *gorm.DB) error { return ImportContactVCardFields(tx, card, id, vault, owner.AccountID) }); err != nil {
		t.Fatal(err)
	}
	var c models.Contact
	if err := svc.db.First(&c, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(c.JobPosition) != "Independent title" || c.CompanyID != nil {
		t.Fatal("incremental omission discarded standalone title")
	}
}
