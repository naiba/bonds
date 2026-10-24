package services

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// An unrelated ORG cannot identify which of two identical company-less roles
// was retained. Check the shared writer and subscription transaction (#297).
func TestVCardMixedOrganizationTitleIdentity(t *testing.T) {
	for _, scenario := range []struct{ entry, independent string }{
		{"subscription", "Artist"}, {"subscription", "Engineer"}, {"incremental", "Engineer"},
	} {
		t.Run(scenario.entry+"/"+scenario.independent, func(t *testing.T) {
			syncer, clients, cards, vault, user, account := setupDavSyncTest(t)
			db := cards.db
			contact, err := NewContactService(db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Original", LastName: "Person"})
			if err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(db)
			if _, err := jobs.LegacyUpdate(contact.ID, vault, user, dto.UpdateJobInfoRequest{JobPosition: scenario.independent}); err != nil {
				t.Fatal(err)
			}
			for _, entry := range [][2]string{{"   ", "Engineer"}, {"Retained employer", "Analyst"}} {
				company, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: entry[0]})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: entry[1]}); err != nil {
					t.Fatal(err)
				}
			}
			note := models.Note{ContactID: contact.ID, VaultID: vault, Body: "Unrelated synthetic context"}
			if err := db.Create(&note).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&note, note.ID).Error; err != nil {
				t.Fatal(err)
			}
			var before models.Contact
			var beforeJobs []models.ContactCompany
			if err := db.First(&before, "id = ?", contact.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("contact_id = ?", contact.ID).Order("id").Find(&beforeJobs).Error; err != nil {
				t.Fatal(err)
			}
			wire, err := cards.ExportContact(contact.ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(bytes.NewReader(wire)).Decode()
			if err != nil {
				t.Fatal(err)
			}
			org := vcard.NewOrganization([]string{"Retained employer"})
			org.Group = "namedEmployer"
			card[vcard.FieldOrganization] = []*vcard.Field{org}
			card[vcard.FieldTitle] = []*vcard.Field{{Group: "retainedRole", Value: "Engineer"}, {Group: "namedEmployer", Value: "Analyst"}}
			card.SetName(&vcard.Name{GivenName: "Changed", FamilyName: "Person"})
			card.SetValue(vcard.FieldFormattedName, "Changed Person")
			sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
			path := "/addressbooks/user/default/mixed-employment.vcf"
			if err := upsertContactSubscriptionState(db, contact.ID, sub.ID, path, "before"); err != nil {
				t.Fatal(err)
			}
			var checkpoint models.AddressBookSubscription
			var beforeMapping models.ContactSubscriptionState
			if err := db.First(&checkpoint, "id = ?", sub.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&beforeMapping).Error; err != nil {
				t.Fatal(err)
			}
			reject := scenario.independent == "Engineer"
			for round := range 2 {
				if scenario.entry == "subscription" {
					syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
						return []carddav.AddressObject{{Path: path, ETag: "mixed-revision", Card: card}}, nil
					}}})
					result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
					if err != nil || result.Created != 0 {
						t.Fatalf("sync=%+v %v", result, err)
					}
					if reject {
						if result.Errors != 1 || result.Updated != 0 || result.Skipped != 0 {
							t.Fatalf("expected explicit unsafe-replacement error: %+v", result)
						}
					} else if result.Errors != 0 || round == 0 && result.Updated != 1 || round > 0 && (result.Skipped != 1 || result.Updated != 0) {
						t.Fatalf("accepted revision result=%+v", result)
					}
				} else {
					err := db.Transaction(func(tx *gorm.DB) error { return ImportContactVCardFields(tx, card, contact.ID, vault, account) })
					if !errors.Is(err, ErrVCardUnsafeReplacement) {
						t.Fatalf("ambiguous incremental input=%v", err)
					}
				}
				var after models.Contact
				var afterJobs []models.ContactCompany
				var afterNote models.Note
				var mapping models.ContactSubscriptionState
				var afterCheckpoint models.AddressBookSubscription
				if err := db.First(&after, "id = ?", contact.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ?", contact.ID).Order("id").Find(&afterJobs).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&afterNote, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&mapping).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&afterCheckpoint, "id = ?", sub.ID).Error; err != nil {
					t.Fatal(err)
				}
				exported, err := cards.ExportContact(contact.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(beforeJobs, afterJobs) || !reflect.DeepEqual(note, afterNote) {
					t.Fatal("mixed input changed Job identities/content or unrelated note")
				}
				if reject {
					if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeMapping, mapping) || !bytes.Equal(wire, exported) {
						t.Fatal("rejected input committed profile, mapping or exported data")
					}
					if !reflect.DeepEqual(checkpoint.LastSynchronizedAt, afterCheckpoint.LastSynchronizedAt) || !reflect.DeepEqual(checkpoint.SyncToken, afterCheckpoint.SyncToken) {
						t.Fatal("rejected input advanced subscription checkpoint")
					}
				} else {
					if ptrToStr(after.FirstName) != "Changed" || strings.Contains(string(exported), "TITLE:Artist\r\n") || strings.Count(string(exported), "TITLE:Engineer\r\n") != 1 || strings.Count(string(exported), "TITLE:Analyst\r\n") != 1 {
						t.Fatal("unambiguous mixed subscription lost or duplicated a role")
					}
					if mapping.DistantURI != path || mapping.DistantEtag != "mixed-revision" || mapping.RejectedEtag != "" {
						t.Fatalf("accepted mapping=%+v", mapping)
					}
				}
			}
		})
	}
}
