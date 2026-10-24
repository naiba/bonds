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

func TestVCardRetainedGroupedTitleAcrossImportAndSubscription(t *testing.T) {
	for _, entry := range []string{"subscription", "incremental", "file"} {
		for _, primary := range []string{"independent", "named", "ambiguous", "identical"} {
			t.Run(entry+"/"+primary, func(t *testing.T) {
				syncer, clients, cards, vault, user, account := setupDavSyncTest(t)
				contact, err := NewContactService(cards.db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Original", LastName: "Person"})
				if err != nil {
					t.Fatal(err)
				}
				db := cards.db
				jobs := NewContactJobService(db)
				if primary == "named" {
					company, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Named employer"})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Analyst"}); err != nil {
						t.Fatal(err)
					}
				} else {
					title := "Artist"
					if primary == "identical" {
						title = "Engineer"
					}
					if _, err := jobs.LegacyUpdate(contact.ID, vault, user, dto.UpdateJobInfoRequest{JobPosition: title}); err != nil {
						t.Fatal(err)
					}
				}
				blank, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "   "})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: blank.ID, JobPosition: "Engineer"}); err != nil {
					t.Fatal(err)
				}
				note := models.Note{ContactID: contact.ID, VaultID: vault, Body: "Unrelated private context"}
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
				delete(card, vcard.FieldOrganization)
				retained := &vcard.Field{Group: "retainedRole", Value: "Engineer"}
				card[vcard.FieldTitle] = []*vcard.Field{retained}
				reject := primary == "ambiguous" || primary == "identical"
				if primary == "ambiguous" {
					retained.Value = "Unknown role"
				}
				if reject {
					card.SetName(&vcard.Name{GivenName: "Changed", FamilyName: "Person"})
					card.SetValue(vcard.FieldFormattedName, "Changed Person")
				}
				sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
				path := "/addressbooks/user/default/retained-role.vcf"
				if err := upsertContactSubscriptionState(db, contact.ID, sub.ID, path, "before"); err != nil {
					t.Fatal(err)
				}
				var checkpoint models.AddressBookSubscription
				if err := db.First(&checkpoint, "id = ?", sub.ID).Error; err != nil {
					t.Fatal(err)
				}
				if entry == "file" {
					// File import creates a new contact; it must not retarget the existing UID.
					// Without ORG there is no employer identity to infer for that new contact.
					var body bytes.Buffer
					if err := vcard.NewEncoder(&body).Encode(card); err != nil {
						t.Fatal(err)
					}
					result, err := cards.ImportVCard(vault, user, &body)
					if err != nil || result.ImportedCount != 1 || len(result.Errors) != 0 || len(result.Contacts) != 1 || result.Contacts[0].ID == contact.ID {
						t.Fatalf("file import=%+v %v", result, err)
					}
					var imported models.Contact
					if err := db.Preload("ContactCompanies").First(&imported, "id = ?", result.Contacts[0].ID).Error; err != nil {
						t.Fatal(err)
					}
					if ptrToStr(imported.JobPosition) != retained.Value || imported.CompanyID != nil || len(imported.ContactCompanies) != 0 {
						t.Fatal("file import guessed an employer or lost standalone title")
					}
				} else {
					for round := range 2 {
						if entry == "subscription" {
							syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
								return []carddav.AddressObject{{Path: path, ETag: "retained-revision", Card: card}}, nil
							}}})
							result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
							if err != nil || result.Created != 0 {
								t.Fatalf("subscription=%+v %v", result, err)
							}
							if reject {
								if result.Errors != 1 || result.Updated != 0 || result.Skipped != 0 {
									t.Fatalf("expected explicit unsafe-replacement error: %+v", result)
								}
							} else if result.Errors != 0 {
								t.Fatalf("accepted subscription had errors: %+v", result)
							} else if round > 0 {
								if result.Skipped != 1 || result.Updated != 0 {
									t.Fatalf("expected skipped revision: %+v", result)
								}
							} else if result.Updated != 1 {
								t.Fatalf("expected accepted revision: %+v", result)
							}
							var state models.ContactSubscriptionState
							if err := db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
								t.Fatal(err)
							}
							if state.DistantURI != path {
								t.Fatal("subscription URI changed")
							}
							if reject {
								if state.RejectedEtag != "" || state.DistantEtag != "before" {
									t.Fatalf("rejection mapping=%+v", state)
								}
								var afterCheckpoint models.AddressBookSubscription
								if err := db.First(&afterCheckpoint, "id = ?", sub.ID).Error; err != nil {
									t.Fatal(err)
								}
								if !reflect.DeepEqual(checkpoint.LastSynchronizedAt, afterCheckpoint.LastSynchronizedAt) || !reflect.DeepEqual(checkpoint.SyncToken, afterCheckpoint.SyncToken) {
									t.Fatal("unsafe revision advanced checkpoint")
								}
							} else if state.DistantEtag != "retained-revision" || state.RejectedEtag != "" {
								t.Fatalf("accepted mapping=%+v", state)
							}
						} else {
							err := db.Transaction(func(tx *gorm.DB) error { return ImportContactVCardFields(tx, card, contact.ID, vault, account) })
							if reject {
								if !errors.Is(err, ErrVCardUnsafeReplacement) {
									t.Fatalf("ambiguous import=%v", err)
								}
							} else if err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				var after models.Contact
				var afterJobs []models.ContactCompany
				var afterNote models.Note
				if err := db.First(&after, "id = ?", contact.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Where("contact_id = ?", contact.ID).Order("id").Find(&afterJobs).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&afterNote, note.ID).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(beforeJobs, afterJobs) || !reflect.DeepEqual(note, afterNote) {
					t.Fatal("input changed existing Jobs or unrelated note")
				}
				if reject || entry == "file" {
					if !reflect.DeepEqual(before, after) {
						t.Fatal("rejected update/file import changed original profile")
					}
				} else {
					if !reflect.DeepEqual(before.FirstName, after.FirstName) || !reflect.DeepEqual(before.LastName, after.LastName) || !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) {
						t.Fatal("retained title changed unrelated profile/selection")
					}
					if primary == "named" && (after.CompanyID == nil || *after.CompanyID != beforeJobs[0].CompanyID) {
						t.Fatal("changed selected employer")
					}
					expected := "Artist"
					if primary == "named" {
						expected = "Analyst"
					}
					if entry == "subscription" && primary == "independent" {
						expected = ""
					}
					if ptrToStr(after.JobPosition) != expected {
						t.Fatalf("profile title=%q want %q", ptrToStr(after.JobPosition), expected)
					}
				}
				exported, err := cards.ExportContact(contact.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if (reject || entry == "file") && !bytes.Equal(exported, wire) {
					t.Fatal("rejection/file import changed original vCard")
				}
				if !reject && entry != "file" && strings.Count(string(exported), "TITLE:Engineer\r\n") != 1 {
					t.Fatal("retained Job was duplicated in export")
				}
			})
		}
	}
}
