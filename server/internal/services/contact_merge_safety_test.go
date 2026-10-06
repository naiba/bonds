package services

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestMergeContactsDoesNotCopyDatabaseRowsIntoNotes(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice"), DistantUUID: strPtrOrNil("import-identity"), Description: strPtrOrNil("Gardener")}
	for _, row := range []any{&target, &source} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	_, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID))
	if err != nil {
		t.Fatal(err)
	}
	var notes []models.Note
	if err := svc.db.Where("contact_id = ?", target.ID).Find(&notes).Error; err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("merge persisted unnecessary profile copies: %+v", notes)
	}
	if err := svc.db.First(&target, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(target.Description) != "Gardener" {
		t.Fatal("valuable nonconflicting field lost")
	}
}

func TestMergeContactsRejectsUnauthorizedIncomingReferences(t *testing.T) {
	for _, reference := range []string{"relationship", "first_met_through"} {
		t.Run(reference, func(t *testing.T) {
			svc, vaultID, userID, accountID := setupContactTest(t)
			external, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Restricted"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			incoming := models.Contact{VaultID: external.ID, FirstName: strPtrOrNil("Friend")}
			for _, row := range []any{&target, &source, &incoming} {
				if err := svc.db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			relation := models.Relationship{}
			if reference == "relationship" {
				var kind models.RelationshipType
				if err := svc.db.First(&kind).Error; err != nil {
					t.Fatal(err)
				}
				relation = models.Relationship{ContactID: incoming.ID, RelatedContactID: source.ID, RelationshipTypeID: kind.ID}
				if err := svc.db.Create(&relation).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				if err := svc.db.Model(&incoming).Update("first_met_through_contact_id", source.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
			if err := svc.db.Model(&models.UserVault{}).Where("user_id = ? AND vault_id = ?", userID, external.ID).Update("permission", models.PermissionViewer).Error; err != nil {
				t.Fatal(err)
			}
			_, err = svc.MergeContacts(vaultID, userID, req)
			if !errors.Is(err, ErrContactMergeBlocked) {
				t.Fatal("merge changed a reference owned by a read-only vault")
			}
			if err := svc.db.First(&source, "id = ?", source.ID).Error; err != nil {
				t.Fatal("rejected merge removed source", err)
			}
			if reference == "relationship" {
				if err := svc.db.First(&relation, relation.ID).Error; err != nil {
					t.Fatal(err)
				}
				if relation.RelatedContactID != source.ID {
					t.Fatal("incoming relationship changed")
				}
			} else {
				if err := svc.db.First(&incoming, "id = ?", incoming.ID).Error; err != nil {
					t.Fatal(err)
				}
				if incoming.FirstMetThroughContactID == nil || *incoming.FirstMetThroughContactID != source.ID {
					t.Fatal("incoming introducer changed")
				}
			}
			if err := svc.db.Where("user_id = ? AND vault_id = ?", userID, external.ID).Delete(&models.UserVault{}).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeBlocked) {
				t.Fatalf("non-member reference update: %v", err)
			}
			if err := svc.db.Create(&models.UserVault{UserID: userID, VaultID: external.ID, Permission: models.PermissionEditor}).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
				t.Fatal(err)
			}
			if reference == "relationship" {
				if err := svc.db.First(&relation, relation.ID).Error; err != nil {
					t.Fatal(err)
				}
				if relation.RelatedContactID != target.ID {
					t.Fatal("authorized relationship not migrated")
				}
			} else {
				if err := svc.db.First(&incoming, "id = ?", incoming.ID).Error; err != nil {
					t.Fatal(err)
				}
				if incoming.FirstMetThroughContactID == nil || *incoming.FirstMetThroughContactID != target.ID {
					t.Fatal("authorized introducer not migrated")
				}
			}

		})
	}
}

func TestMergeContactsRejectsLinkedDAVContacts(t *testing.T) {
	for _, way := range []uint8{SyncWayPull, SyncWayPush, SyncWayPull | SyncWayPush} {
		for _, active := range []bool{true, false} {
			push, client, _, svc, vaultID, userID, _ := setupDavPushTest(t)
			target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			for _, row := range []any{&target, &source} {
				if err := svc.db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			sub := createPushSubscription(t, client, vaultID, userID, way)
			if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("active", active).Error; err != nil {
				t.Fatal(err)
			}
			state := models.ContactSubscriptionState{ContactID: source.ID, AddressBookSubscriptionID: sub.ID, DistantURI: "/contacts/source.vcf", DistantEtag: "original"}
			if err := svc.db.Create(&state).Error; err != nil {
				t.Fatal(err)
			}
			svc.SetDavPushService(push)
			_, err := svc.MergeContacts(vaultID, userID, dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}})
			if !errors.Is(err, ErrContactMergeBlocked) {
				t.Fatalf("unsafe linked merge allowed: way=%d active=%v", way, active)
			}
			if err := svc.db.First(&source, "id = ?", source.ID).Error; err != nil {
				t.Fatal(err)
			}
			var retained models.ContactSubscriptionState
			if err := svc.db.First(&retained, state.ID).Error; err != nil {
				t.Fatal(err)
			}
			if retained.ContactID != source.ID || retained.DistantEtag != "original" {
				t.Fatal("DAV identity modified")
			}
		}
	}
}

func TestMergeContactsRequiresCurrentReviewAndExplicitChoices(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice"), Description: strPtrOrNil("Work profile")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alicia"), Description: strPtrOrNil("Personal profile"), FoodPreferences: strPtrOrNil("Vegetarian")}
	for _, row := range []any{&target, &source} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	req := reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	delete(req.FieldChoices, "description")
	if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeReviewChanged) {
		t.Fatalf("missing conflict selection: %v", err)
	}
	req.FieldChoices["description"] = source.ID
	if err := svc.db.Model(&source).Update("description", "Updated personal profile").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeReviewChanged) {
		t.Fatalf("stale review accepted: %v", err)
	}
	req = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	req.FieldChoices["description"] = source.ID
	result, err := svc.MergeContacts(vaultID, userID, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.FirstName != "Alice" {
		t.Fatal("field selection changed target name")
	}
	if err := svc.db.First(&target, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(target.Description) != "Updated personal profile" || ptrToStr(target.FoodPreferences) != "Vegetarian" {
		t.Fatalf("field choices lost: %+v", target)
	}
	var notes int64
	if err := svc.db.Model(&models.Note{}).Count(&notes).Error; err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Fatal("unselected data copied to notes")
	}
}

func TestMergeContactsMonicaReimportDoesNotRecreateDeletedSource(t *testing.T) {
	svc, vaultID, userID, accountID := setupContactTest(t)
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice"), DistantUUID: strPtrOrNil("monica-source")}
	for _, row := range []any{&target, &source} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	importer := &MonicaImportService{DB: svc.db}
	report := &dto.MonicaImportResponse{}
	id, created, err := importer.importContact(svc.db, &MonicaContact{UUID: "monica-source"}, vaultID, accountID, userID, nil, report)
	if err != nil {
		t.Fatal(err)
	}
	if created || id != "" || report.SkippedCount != 1 {
		t.Fatalf("merged source reimported: id=%q created=%v report=%+v", id, created, report)
	}
	var active int64
	if err := svc.db.Model(&models.Contact{}).Where("vault_id = ?", vaultID).Count(&active).Error; err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("reimport recreated duplicates: %d", active)
	}
}

func TestMergeContactsVCardReimportIsAppendOnly(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	card := "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:synthetic-source\r\nFN:Alicia Chen\r\nN:Chen;Alicia;;;\r\nTEL:+15550101010\r\nEND:VCARD\r\n"
	importer := NewVCardService(svc.db)
	first, err := importer.ImportVCard(vaultID, userID, strings.NewReader(card))
	if err != nil {
		t.Fatal(err)
	}
	if first.ImportedCount != 1 || len(first.Contacts) != 1 {
		t.Fatalf("initial import: %+v", first)
	}
	target, err := svc.CreateContact(vaultID, userID, dto.CreateContactRequest{FirstName: "Alice", LastName: "Chen"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, first.Contacts[0].ID)); err != nil {
		t.Fatal(err)
	}
	repeated, err := importer.ImportVCard(vaultID, userID, strings.NewReader(card))
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ImportedCount != 1 || len(repeated.Contacts) != 1 || repeated.Contacts[0].ID == target.ID || repeated.Contacts[0].ID == first.Contacts[0].ID {
		t.Fatalf("unexpected reimport identity: %+v", repeated)
	}
	var retained models.Contact
	if err := svc.db.First(&retained, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(retained.FirstName) != "Alice" {
		t.Fatal("reimport overwrote chosen profile")
	}
	var phones []models.ContactInformation
	if err := svc.db.Where("contact_id = ?", target.ID).Find(&phones).Error; err != nil {
		t.Fatal(err)
	}
	if len(phones) != 1 || phones[0].Data != "+15550101010" {
		t.Fatalf("merged contact information was lost: %+v", phones)
	}
}

func TestMergeContactsDAVTargetAndUnrelatedSubscription(t *testing.T) {
	_, client, _, svc, vaultID, userID, _ := setupDavPushTest(t)
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	unrelated := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Remote friend")}
	for _, row := range []any{&target, &source, &unrelated} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	sub := createPushSubscription(t, client, vaultID, userID, SyncWayPull)
	state := models.ContactSubscriptionState{ContactID: target.ID, AddressBookSubscriptionID: sub.ID, DistantURI: "/remote/alice.vcf", DistantEtag: "original"}
	if err := svc.db.Create(&state).Error; err != nil {
		t.Fatal(err)
	}
	req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
	if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeBlocked) {
		t.Fatalf("linked target accepted: %v", err)
	}
	if err := svc.db.Model(&state).Update("contact_id", unrelated.ID).Error; err != nil {
		t.Fatal(err)
	}
	// An unrelated pull subscription must not prevent a local-only merge.
	req = reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)
	if err := svc.db.Model(&target).Update("distant_uri", "/legacy/alice.vcf").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeBlocked) {
		t.Fatalf("legacy remote target accepted: %v", err)
	}
	if err := svc.db.Model(&target).Update("distant_uri", nil).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("sync_way", SyncWayPush).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeBlocked) {
		t.Fatalf("active outgoing synchronization accepted: %v", err)
	}
	if err := svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&state, state.ID).Error; err != nil {
		t.Fatal(err)
	}
	if state.ContactID != unrelated.ID || state.DistantEtag != "original" {
		t.Fatal("unrelated synchronization mapping changed")
	}
}

func TestMergeContactsDAVExportsRetainedPrimaryBirthday(t *testing.T) {
	svc, vaultID, userID, _ := setupContactTest(t)
	source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
	for _, row := range []any{&source, &target} {
		if err := svc.db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	var kind models.ContactImportantDateType
	if err := svc.db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&kind).Error; err != nil {
		t.Fatal(err)
	}
	month, day, sourceYear, targetYear := 3, 2, 1990, 1991
	for _, date := range []models.ContactImportantDate{{ContactID: source.ID, Label: "Birthday", ContactImportantDateTypeID: &kind.ID, Year: &sourceYear, Month: &month, Day: &day}, {ContactID: target.ID, Label: "Birthday", ContactImportantDateTypeID: &kind.ID, Year: &targetYear, Month: &month, Day: &day}} {
		if err := svc.db.Create(&date).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.MergeContacts(vaultID, userID, reviewedContactMerge(t, svc, vaultID, userID, target.ID, source.ID)); err != nil {
		t.Fatal(err)
	}
	card, err := NewVCardService(svc.db).ExportContactToVCard(target.ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if card.Value("BDAY") != "1991-03-02" {
		t.Fatalf("DAV exported discarded primary birthday: %q", card.Value("BDAY"))
	}
}

func TestMergeContactsPreservesIntroducerOrBlocksUnavailableReference(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(fmt.Sprint(available), func(t *testing.T) {
			svc, vaultID, userID, accountID := setupContactTest(t)
			introducerVault := vaultID
			if !available {
				vault, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "Other vault"}, "en")
				if err != nil {
					t.Fatal(err)
				}
				introducerVault = vault.ID
			}
			introducer := models.Contact{VaultID: introducerVault, Nickname: strPtrOrNil("Gardening friend")}
			if err := svc.db.Create(&introducer).Error; err != nil {
				t.Fatal(err)
			}
			target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice"), FirstMetThroughContactID: &introducer.ID}
			for _, row := range []any{&target, &source} {
				if err := svc.db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			review, err := svc.PreviewContactMerge(vaultID, userID, req)
			if err != nil {
				t.Fatal(err)
			}
			req.ReviewToken = review.ReviewToken
			if !available {
				if !slices.Contains(review.Blockers, "introducer_unavailable") {
					t.Fatalf("unavailable introducer must block, got %+v", review)
				}
				if _, err := svc.MergeContacts(vaultID, userID, req); !errors.Is(err, ErrContactMergeBlocked) {
					t.Fatalf("merge should preserve invalid reference for resolution: %v", err)
				}
				if err := svc.db.First(&source, "id = ?", source.ID).Error; err != nil {
					t.Fatal(err)
				}
				if source.FirstMetThroughContactID == nil || *source.FirstMetThroughContactID != introducer.ID {
					t.Fatal("blocked merge changed introducer")
				}
				return
			}
			found := false
			for _, field := range review.Fields {
				if field.Key == "introducer" && len(field.Options) == 1 && field.Options[0].Value == "Gardening friend" {
					found = true
				}
			}
			if !found {
				t.Fatal("nickname-only introducer missing from review")
			}
			if _, err := svc.MergeContacts(vaultID, userID, req); err != nil {
				t.Fatal(err)
			}
			if err := svc.db.First(&target, "id = ?", target.ID).Error; err != nil {
				t.Fatal(err)
			}
			if target.FirstMetThroughContactID == nil || *target.FirstMetThroughContactID != introducer.ID {
				t.Fatal("introducer silently lost")
			}
		})
	}
}
