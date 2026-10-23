package services

import (
	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"testing"
)

func TestMergeSelectedStandaloneEmploymentPreservesIdentity(t *testing.T) {
	for _, side := range []string{"target", "source"} {
		for _, action := range []string{"legacy edit", "vcard edit", "round trip", "remove job", "remove company", "clear title", "omit title"} {
			t.Run(side+"/"+action, func(t *testing.T) {
				svc, vault, user, account := setupContactTest(t)
				target, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic target"})
				if err != nil {
					t.Fatal(err)
				}
				source, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic source"})
				if err != nil {
					t.Fatal(err)
				}
				standaloneID, normalID := target.ID, source.ID
				if side == "source" {
					standaloneID, normalID = source.ID, target.ID
				}
				jobs := NewContactJobService(svc.db)
				if _, err = jobs.LegacyUpdate(standaloneID, vault, user, dto.UpdateJobInfoRequest{JobPosition: "Independent artist"}); err != nil {
					t.Fatal(err)
				}
				company := models.Company{VaultID: vault, Name: "Synthetic employer"}
				if err = svc.db.Create(&company).Error; err != nil {
					t.Fatal(err)
				}
				if _, err = jobs.LegacyUpdate(normalID, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Unrelated consultant"}); err != nil {
					t.Fatal(err)
				}
				req := reviewedContactMerge(t, svc, vault, user, target.ID, source.ID)
				req.FieldChoices["job"] = standaloneID
				response, err := svc.MergeContacts(vault, user, req)
				if err != nil {
					t.Fatal(err)
				}
				if response.JobPosition != "Independent artist" {
					t.Fatalf("reviewed response title lost: %+v", response)
				}
				card, err := NewVCardService(svc.db).ExportContactToVCard(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, title := range card[vcard.FieldTitle] {
					if title.Value == "Independent artist" {
						found = true
					}
				}
				if !found {
					t.Errorf("review selected standalone title but persisted export hides it: %v", card[vcard.FieldTitle])
				}
				before, err := jobs.List(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if len(before) != 1 {
					t.Fatalf("expected original normal job retained: %+v", before)
				}
				wantTitle := "Independent artist"
				vsvc := NewVCardService(svc.db)
				apply := func(card vcard.Card) {
					t.Helper()
					if err := svc.db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, target.ID, vault, account) }); err != nil {
						t.Fatal(err)
					}
				}
				switch action {
				case "legacy edit":
					wantTitle = "Edited independent artist"
					if _, err = jobs.LegacyUpdate(target.ID, vault, user, dto.UpdateJobInfoRequest{JobPosition: wantTitle}); err != nil {
						t.Fatal(err)
					}
				case "vcard edit":
					wantTitle = "Edited independent artist"
					apply(vcard.Card{vcard.FieldTitle: []*vcard.Field{{Value: wantTitle}}})
				case "round trip":
					wire, err := vsvc.ExportContact(target.ID, vault)
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := vcard.NewDecoder(strings.NewReader(string(wire))).Decode()
					if err != nil {
						t.Fatal(err)
					}
					if len(decoded[vcard.FieldTitle]) != 2 || decoded[vcard.FieldTitle][0].Group == decoded[vcard.FieldTitle][1].Group {
						t.Fatal("independent and employed titles need distinct wire groups")
					}
					apply(decoded)
					imported, err := vsvc.ImportVCard(vault, user, strings.NewReader(string(wire)))
					if err != nil {
						t.Fatal(err)
					}
					if len(imported.Contacts) != 1 {
						t.Fatalf("unexpected import: %+v", imported)
					}
					importedCard, err := vsvc.ExportContactToVCard(imported.Contacts[0].ID, vault)
					if err != nil {
						t.Fatal(err)
					}
					if importedCard.Value(vcard.FieldTitle) != wantTitle || len(importedCard[vcard.FieldTitle]) != 2 {
						t.Fatal("import lost standalone title or associated job")
					}
				case "remove job":
					if err = jobs.Delete(target.ID, vault, before[0].ID); err != nil {
						t.Fatal(err)
					}
				case "remove company":
					if err = NewCompanyService(svc.db).Delete(company.ID, vault); err != nil {
						t.Fatal(err)
					}
				case "clear title":
					wantTitle = ""
					apply(vcard.Card{vcard.FieldTitle: []*vcard.Field{{Value: "  "}}})
				case "omit title":
					wantTitle = ""
					apply(vcard.Card{})
				}
				after, err := jobs.List(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if action == "remove job" || action == "remove company" {
					if len(after) != 0 {
						t.Fatal("removed employment still exists")
					}
				} else if !reflect.DeepEqual(after, before) {
					t.Errorf("operation changed unrelated job: before=%+v after=%+v", before, after)
				}
				var stored models.Contact
				if err = svc.db.First(&stored, "id = ?", target.ID).Error; err != nil {
					t.Fatal(err)
				}
				if stored.CompanyID != nil || stored.PreferredJobID != nil || ptrToStr(stored.JobPosition) != wantTitle || (wantTitle == "" && stored.JobPosition != nil) {
					t.Errorf("standalone selection not persisted: company=%v preferred=%v title=%v", stored.CompanyID, stored.PreferredJobID, stored.JobPosition)
				}
				exported, err := vsvc.ExportContactToVCard(target.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				if wantTitle != "" && exported.Value(vcard.FieldTitle) != wantTitle {
					t.Errorf("export selected %q instead of %q", exported.Value(vcard.FieldTitle), wantTitle)
				}
				if wantTitle == "" {
					for _, title := range exported[vcard.FieldTitle] {
						if title.Value == "Independent artist" {
							t.Error("cleared title still exported")
						}
					}
				}
			})
		}
	}
}
