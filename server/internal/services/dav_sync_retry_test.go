package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestDAVPartialSyncRetryDistinguishesRemoteAndLocalEdits(t *testing.T) {
	for _, mode := range []string{"incremental", "full", "query"} {
		for _, localEdit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/local_edit_%t", mode, localEdit), func(t *testing.T) {
				syncer, client, _, vaultID, userID, _ := setupDavSyncTest(t)
				db := syncer.db
				sub, err := client.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{URI: "https://dav.example.test/contacts/", Username: "synthetic", Password: "synthetic"})
				if err != nil {
					t.Fatal(err)
				}
				checkpoint := time.Now().Add(-time.Hour).Truncate(time.Second)
				lastLocalEdit := checkpoint.Add(-time.Hour)
				token := ""
				if mode == "incremental" {
					token = "accepted-token"
				}
				if err := db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Updates(map[string]any{"last_synchronized_at": checkpoint, "distant_sync_token": token, "address_book_path": "/contacts/"}).Error; err != nil {
					t.Fatal(err)
				}
				protected := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Protected"), LastUpdatedAt: &lastLocalEdit, DistantURI: strPtrOrNil("/contacts/protected.vcf"), DistantEtag: strPtrOrNil("accepted-protected")}
				existing := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Existing"), LastUpdatedAt: &lastLocalEdit, DistantURI: strPtrOrNil("/contacts/existing.vcf"), DistantEtag: strPtrOrNil("accepted-existing")}
				address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Garden road"), Line2: strPtrOrNil("Apartment A")}
				for _, row := range []any{&protected, &existing, &address} {
					if err := db.Create(row).Error; err != nil {
						t.Fatal(err)
					}
				}
				link := models.ContactAddress{ContactID: protected.ID, AddressID: address.ID}
				if err := db.Create(&link).Error; err != nil {
					t.Fatal(err)
				}
				for _, contact := range []models.Contact{protected, existing} {
					if err := upsertContactSubscriptionState(db, contact.ID, sub.ID, *contact.DistantURI, *contact.DistantEtag); err != nil {
						t.Fatal(err)
					}
				}
				objects := []carddav.AddressObject{
					{Path: *protected.DistantURI, ETag: "changed-protected", Card: makeVCard("Protected", "", "protected")},
					{Path: *existing.DistantURI, ETag: "first-existing", Card: makeVCard("First remote existing", "", "existing")},
					{Path: "/contacts/created.vcf", ETag: "first-created", Card: makeVCard("First remote created", "", "created")},
				}
				// Omitting the detailed address is a real rejected replacement, while
				// the other two objects commit in the same subscription run.
				syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
					syncCollFn: func(_ context.Context, _ string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
						if query.SyncToken != token {
							t.Errorf("retry checkpoint = %q, want %q", query.SyncToken, token)
						}
						if mode == "query" {
							return nil, fmt.Errorf("sync collection unsupported")
						}
						return &carddav.SyncResponse{SyncToken: "next-token", Updated: objects}, nil
					},
					multiGetFn: func(context.Context, string, *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
						return objects, nil
					},
					queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
						return objects, nil
					},
				}})
				result, err := syncer.SyncSubscription(context.Background(), sub.ID, vaultID)
				if err != nil || result.Errors != 1 || result.Updated != 1 || result.Created != 1 || result.Skipped != 0 {
					t.Fatalf("partial sync: %+v, %v", result, err)
				}
				var storedSub models.AddressBookSubscription
				if err := db.First(&storedSub, "id = ?", sub.ID).Error; err != nil {
					t.Fatal(err)
				}
				if ptrToStr(storedSub.DistantSyncToken) != token || storedSub.LastSynchronizedAt == nil || !storedSub.LastSynchronizedAt.Equal(checkpoint) {
					t.Fatal("failed batch advanced its checkpoint")
				}
				if localEdit {
					if _, err := NewContactService(db).UpdateContact(existing.ID, vaultID, userID, dto.UpdateContactRequest{FirstName: "Actual local edit"}); err != nil {
						t.Fatal(err)
					}
				}
				objects[1].Card = makeVCard("Second remote existing", "", "existing")
				objects[1].ETag = "second-existing"
				objects[2].Card = makeVCard("Second remote created", "", "created")
				objects[2].ETag = "second-created"
				// Resolve the rejected object's representation, then retry from the
				// held checkpoint with newer remote changes to both successful objects.
				objects[0].Card.AddAddress(&vcard.Address{StreetAddress: "Garden road"})
				result, err = syncer.SyncSubscription(context.Background(), sub.ID, vaultID)
				wantUpdated, wantSkipped := 3, 0
				if localEdit {
					wantUpdated, wantSkipped = 2, 1
				}
				if err != nil || result.Errors != 0 || result.Created != 0 || result.Updated != wantUpdated || result.Skipped != wantSkipped {
					t.Errorf("recovered sync: %+v, %v; want updated=%d skipped=%d", result, err, wantUpdated, wantSkipped)
				}
				for i, obj := range objects {
					var contact models.Contact
					if err := db.Where("vault_id = ? AND distant_uri = ?", vaultID, obj.Path).First(&contact).Error; err != nil {
						t.Fatal(err)
					}
					wantName, wantETag := obj.Card.Name().GivenName, obj.ETag
					if i == 1 && localEdit {
						wantName, wantETag = "Actual local edit", "first-existing"
					}
					var state models.ContactSubscriptionState
					if err := db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
						t.Fatal(err)
					}
					if ptrToStr(contact.FirstName) != wantName || ptrToStr(contact.DistantEtag) != wantETag || state.DistantEtag != wantETag {
						t.Errorf("%s: name=%q etag=%q mapping=%q; want %q/%q", obj.Path, ptrToStr(contact.FirstName), ptrToStr(contact.DistantEtag), state.DistantEtag, wantName, wantETag)
					}
				}
				if err := db.First(&storedSub, "id = ?", sub.ID).Error; err != nil {
					t.Fatal(err)
				}
				if storedSub.LastSynchronizedAt == nil || !storedSub.LastSynchronizedAt.After(checkpoint) || (mode != "query" && ptrToStr(storedSub.DistantSyncToken) != "next-token") {
					t.Fatal("recovered sync did not advance checkpoint")
				}
				if err := db.First(&address, address.ID).Error; err != nil || ptrToStr(address.Line2) != "Apartment A" {
					t.Fatalf("retry lost protected address details: %+v, %v", address, err)
				}
			})
		}
	}
}
