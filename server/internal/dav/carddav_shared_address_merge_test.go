package dav

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

func TestMergedSharedAddressDAVRoundTripsPreserveAssociations(t *testing.T) {
	for _, entry := range []string{"local_carddav_put", "subscription_upsert"} {
		t.Run(entry, func(t *testing.T) {
			backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
			target := createTestContact(t, db, vaultID, userID, "Alice", "Chen")
			source := createTestContact(t, db, vaultID, userID, "Alice", "Chen")
			neighbor := createTestContact(t, db, vaultID, userID, "Neighbor", "Chen")
			latitude, longitude := 12.5, 34.5
			address := models.Address{VaultID: vaultID, Line1: strPtrOrNil("Synthetic shared street"), Latitude: &latitude, Longitude: &longitude}
			if err := db.Create(&address).Error; err != nil {
				t.Fatal(err)
			}
			movedIn := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			movedOut := movedIn.AddDate(1, 0, 0)
			links := []models.ContactAddress{
				{ContactID: target.ID, AddressID: address.ID, DateFrom: &movedIn},
				{ContactID: source.ID, AddressID: address.ID, DateTo: &movedOut, IsPastAddress: true},
				{ContactID: neighbor.ID, AddressID: address.ID},
			}
			for i := range links {
				if err := db.Create(&links[i]).Error; err != nil {
					t.Fatal(err)
				}
			}
			svc := services.NewContactService(db)
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			preview, err := svc.PreviewContactMerge(vaultID, userID, req)
			if err != nil {
				t.Fatal(err)
			}
			req.ReviewToken = preview.ReviewToken
			if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
				t.Fatal(err)
			}
			path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + target.ID + ".vcf"
			for attempt := 0; attempt < 3; attempt++ {
				object, err := backend.GetAddressObject(ctx, path, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(object.Card.Addresses()) != 2 {
					t.Errorf("roundtrip %d: exported %d addresses, want 2 association projections", attempt, len(object.Card.Addresses()))
				}
				var encoded bytes.Buffer
				if err := vcard.NewEncoder(&encoded).Encode(object.Card); err != nil {
					t.Fatal(err)
				}
				card, err := vcard.NewDecoder(&encoded).Decode()
				if err != nil {
					t.Fatal(err)
				}
				if entry == "local_carddav_put" {
					_, err = backend.PutAddressObject(ctx, path, card, nil)
				} else {
					err = db.Transaction(func(tx *gorm.DB) error {
						_, _, err := services.NewVCardService(tx).UpsertContactFromVCard(tx, card, vaultID, userID, AccountIDFromContext(ctx), target.ID, "/remote/shared.vcf", fmt.Sprintf("shared-revision-%d", attempt), nil)
						return err
					})
				}
				if err != nil {
					t.Fatal(err)
				}
				var addresses, associations int64
				if err := db.Model(&models.Address{}).Where("vault_id = ?", vaultID).Count(&addresses).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Model(&models.ContactAddress{}).Where("contact_id = ?", target.ID).Count(&associations).Error; err != nil {
					t.Fatal(err)
				}
				if addresses != 1 || associations != 2 {
					t.Errorf("roundtrip %d created duplicates: addresses=%d associations=%d, want 1/2", attempt, addresses, associations)
				}
				for i, original := range links {
					var current models.ContactAddress
					if err := db.First(&current, original.ID).Error; err != nil {
						t.Fatal(err)
					}
					wantContact := target.ID
					if i == 2 {
						wantContact = neighbor.ID
					}
					if current.ContactID != wantContact || current.AddressID != address.ID || current.IsPastAddress != original.IsPastAddress || (original.DateFrom != nil && (current.DateFrom == nil || !current.DateFrom.Equal(*original.DateFrom))) || (original.DateTo != nil && (current.DateTo == nil || !current.DateTo.Equal(*original.DateTo))) {
						t.Errorf("association/history changed: %+v", current)
					}
				}
				if err := db.First(&address, address.ID).Error; err != nil || address.Latitude == nil || address.Longitude == nil || *address.Latitude != latitude || *address.Longitude != longitude {
					t.Fatalf("shared coordinates lost: %+v, %v", address, err)
				}
			}
		})
	}
}
