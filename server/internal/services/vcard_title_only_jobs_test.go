package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardTitleOnlyJobsSubscriptionAndImport(t *testing.T) {
	for _, entrypoint := range []string{"subscription", "incremental import"} {
		t.Run(entrypoint, func(t *testing.T) {
			syncer, clients, cards, vault, user, account := setupDavSyncTest(t)
			contact, err := NewContactService(cards.db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic", LastName: "Employee"})
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Named employer", "   "} {
				company, err := NewCompanyService(cards.db).Create(vault, dto.CreateCompanyRequest{Name: name})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := NewContactJobService(cards.db).Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Role for " + name}); err != nil {
					t.Fatal(err)
				}
			}
			var before models.Contact
			var beforeJobs []models.ContactCompany
			if err := cards.db.First(&before, "id = ?", contact.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := cards.db.Where("contact_id = ?", contact.ID).Order("id").Find(&beforeJobs).Error; err != nil {
				t.Fatal(err)
			}
			sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
			path := "/addressbooks/user/default/title-only.vcf"
			if err := upsertContactSubscriptionState(cards.db, contact.ID, sub.ID, path, "before"); err != nil {
				t.Fatal(err)
			}
			for round := range 3 {
				wire, err := cards.ExportContact(contact.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				card, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
				if err != nil {
					t.Fatal(err)
				}
				etag := fmt.Sprintf("title-only-%d", round)
				if entrypoint == "subscription" {
					syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
						return []carddav.AddressObject{{Path: path, ETag: etag, Card: card}}, nil
					}}})
					result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
					if err != nil || result.Errors != 0 || result.Updated != 1 {
						t.Fatalf("pull round %d: %+v %v", round, result, err)
					}
					var state models.ContactSubscriptionState
					if err := cards.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
						t.Fatal(err)
					}
					if state.DistantEtag != etag || state.DistantURI != path || state.RejectedEtag != "" {
						t.Fatalf("accepted pull mapping not committed: %+v", state)
					}
				} else {
					if err := cards.db.Transaction(func(tx *gorm.DB) error { return ImportContactVCardFields(tx, card, contact.ID, vault, account) }); err != nil {
						t.Fatal(err)
					}
				}
				var after models.Contact
				var afterJobs []models.ContactCompany
				if err := cards.db.First(&after, "id = ?", contact.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := cards.db.Where("contact_id = ?", contact.ID).Order("id").Find(&afterJobs).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(beforeJobs, afterJobs) || !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) {
					t.Fatal("round trip changed job content or selection")
				}
				if hasStandaloneEmployment(&after) {
					t.Fatal("round trip copied a Job into an independent title")
				}
				if !reflect.DeepEqual(before.FirstName, after.FirstName) || !reflect.DeepEqual(before.LastName, after.LastName) {
					t.Fatal("round trip changed unrelated name")
				}
				got, err := cards.ExportContactToVCard(contact.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(exportedJobPairs(card), exportedJobPairs(got)) {
					t.Fatal("round trip changed employment projection")
				}
			}
		})
	}
}
