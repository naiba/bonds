package dav

import (
	"fmt"
	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
	"testing"
)

func TestCardDAVBirthdayDeletionPreservesOrdinaryDates(t *testing.T) {
	for _, entry := range []string{"local_carddav_put", "subscription_upsert"} {
		t.Run(entry, func(t *testing.T) {
			backend, db, ctx, vaultID, userID := setupCardDAVTest(t)
			target := createTestContact(t, db, vaultID, userID, "Alice", "Chen")
			source := createTestContact(t, db, vaultID, userID, "Alice", "Chen")
			var kind models.ContactImportantDateType
			if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&kind).Error; err != nil {
				t.Fatal(err)
			}
			year, alternateYear, month, day := 1990, 1991, 3, 2
			primary := models.ContactImportantDate{ContactID: target.ID, Label: "Birthdate", ContactImportantDateTypeID: &kind.ID, Year: &year, Month: &month, Day: &day, DatePrecision: "full"}
			alternate := models.ContactImportantDate{ContactID: source.ID, Label: "Birthdate", ContactImportantDateTypeID: &kind.ID, Year: &alternateYear, Month: &month, Day: &day, DatePrecision: "full"}
			for _, date := range []*models.ContactImportantDate{&primary, &alternate} {
				if err := db.Create(date).Error; err != nil {
					t.Fatal(err)
				}
			}
			reminder := models.ContactReminder{ContactID: source.ID, Label: "Alternate birthday", Type: "recurring_year", ImportantDateID: &alternate.ID, Month: &month, Day: &day}
			if err := db.Create(&reminder).Error; err != nil {
				t.Fatal(err)
			}
			svc := services.NewContactService(db)
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			preview, err := svc.PreviewContactMerge(vaultID, userID, req)
			if err != nil {
				t.Fatal(err)
			}
			if len(preview.Blockers) != 0 {
				t.Fatalf("unexpected blockers: %v", preview.Blockers)
			}
			req.ReviewToken = preview.ReviewToken
			if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
				t.Fatal(err)
			}
			var ordinary models.ContactImportantDate
			if err := db.First(&ordinary, alternate.ID).Error; err != nil {
				t.Fatal(err)
			}
			if ordinary.ContactID != target.ID || ordinary.ContactImportantDateTypeID != nil {
				t.Fatalf("alternate was not preserved as ordinary date: %+v", ordinary)
			}
			path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + target.ID + ".vcf"
			original, err := backend.GetAddressObject(ctx, path, &carddav.AddressDataRequest{AllProp: true})
			if err != nil {
				t.Fatal(err)
			}
			card := original.Card
			if card.Value(vcard.FieldBirthday) != "1990-03-02" {
				t.Fatalf("unexpected projection: %v", card)
			}
			writeCard := func(revision int) {
				t.Helper()
				if entry == "local_carddav_put" {
					_, err = backend.PutAddressObject(ctx, path, card, nil)
				} else {
					err = db.Transaction(func(tx *gorm.DB) error {
						_, _, err := services.NewVCardService(tx).UpsertContactFromVCard(tx, card, vaultID, userID, AccountIDFromContext(ctx), target.ID, "/remote/alice.vcf", fmt.Sprintf("revision-%d", revision), nil)
						return err
					})
				}
				if err != nil {
					t.Fatalf("entry=%s revision=%d: %v", entry, revision, err)
				}
			}
			card.SetValue(vcard.FieldBirthday, "1992-03-02")
			writeCard(0)
			var edited models.ContactImportantDate
			if err := db.First(&edited, primary.ID).Error; err != nil || edited.Year == nil || *edited.Year != 1992 {
				t.Fatalf("primary birthday edit lost identity: %+v %v", edited, err)
			}
			delete(card, vcard.FieldBirthday)
			for attempt := 1; attempt <= 3; attempt++ {
				if attempt == 3 {
					card.SetValue(vcard.FieldNickname, "Updated after birthday removal")
				}
				writeCard(attempt)
				var count int64
				if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", primary.ID).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatal("requested primary birthday deletion did not take effect")
				}
				if err := db.Model(&models.ContactImportantDate{}).Where("id = ?", alternate.ID).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Errorf("attempt=%d deleted ordinary alternate birthday id=%d without a projected BDAY", attempt, alternate.ID)
				}
				if err := db.Model(&models.ContactReminder{}).Where("id = ?", reminder.ID).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Errorf("attempt=%d deleted alternate reminder id=%d", attempt, reminder.ID)
				}
				exported, exportErr := services.NewVCardService(db).ExportContactToVCard(target.ID, vaultID)
				if exportErr != nil {
					t.Fatal(exportErr)
				}
				if birthday := exported.Value(vcard.FieldBirthday); birthday != "" {
					t.Errorf("attempt=%d promoted ordinary date into BDAY: %s", attempt, birthday)
				}
				var retained models.ContactImportantDate
				if err := db.First(&retained, alternate.ID).Error; err != nil {
					t.Error(err)
				} else if retained.ContactID != target.ID || retained.ContactImportantDateTypeID != nil || retained.Label != alternate.Label || retained.Year == nil || *retained.Year != alternateYear {
					t.Errorf("alternate date changed: %+v", retained)
				}

			}
		})
	}
}
