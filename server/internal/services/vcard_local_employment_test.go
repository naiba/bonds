package services

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestSubscriptionLocalEmploymentWins(t *testing.T) {
	for _, mutation := range []string{"create", "update", "delete", "add employee", "remove employee", "legacy update", "legacy delete"} {
		t.Run(mutation, func(t *testing.T) {
			svc, _, vaultID, userID := setupVCardTest(t)
			var user models.User
			if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
				t.Fatal(err)
			}
			card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nORG:Acme\r\nTITLE:Engineer\r\nEND:VCARD\r\n")).Decode()
			if err != nil {
				t.Fatal(err)
			}
			var id string
			err = svc.db.Transaction(func(tx *gorm.DB) error {
				var err error
				id, _, err = svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, "", "/local-employment.vcf", "remote-first", nil)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			lastSync := time.Now().Add(-time.Second)
			jobs := NewContactJobService(svc.db)
			before, err := jobs.List(id, vaultID)
			if err != nil || len(before) != 1 {
				t.Fatalf("initial jobs: %v %v", before, err)
			}
			switch mutation {
			case "create":
				_, err = jobs.Create(id, vaultID, dto.CreateContactJobRequest{CompanyID: before[0].CompanyID, JobPosition: "Architect"})
			case "update":
				_, err = jobs.Update(id, vaultID, before[0].ID, dto.UpdateContactJobRequest{CompanyID: before[0].CompanyID, JobPosition: "Architect"})
			case "delete":
				err = jobs.Delete(id, vaultID, before[0].ID)
			case "add employee":
				_, err = jobs.AddEmployee(before[0].CompanyID, vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Architect"})
			case "remove employee":
				err = jobs.RemoveEmployee(before[0].CompanyID, vaultID, id)
			case "legacy update":
				_, err = jobs.LegacyUpdate(id, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: &before[0].CompanyID, JobPosition: "Architect"})
			case "legacy delete":
				_, err = jobs.LegacyDelete(id, vaultID, userID)
			}
			if err != nil {
				t.Fatal(err)
			}
			local, err := jobs.List(id, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			var contact models.Contact
			if err := svc.db.First(&contact, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if contact.LastUpdatedAt == nil || !contact.LastUpdatedAt.After(lastSync) {
				t.Error("local employment mutation did not set conflict marker")
			}
			card.SetValue(vcard.FieldTelephone, "+12025550123")
			var action string
			err = svc.db.Transaction(func(tx *gorm.DB) error {
				var err error
				_, action, err = svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, id, "/local-employment.vcf", "remote-phone-edited", &lastSync)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			after, err := jobs.List(id, vaultID)
			if err != nil {
				t.Fatal(err)
			}
			if action != "conflict_local_wins" || !reflect.DeepEqual(local, after) {
				t.Fatalf("local jobs overwritten: action=%s local=%+v after=%+v", action, local, after)
			}
		})
	}
}

func TestVCardEmploymentGroupsIgnoreCase(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)
	wire := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Group\r\nitem1.ORG:Acme\r\nITEM1.TITLE:Engineer\r\nEND:VCARD\r\n"
	imported, err := svc.ImportVCard(vaultID, userID, strings.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Contacts) != 1 {
		t.Fatal("missing imported contact")
	}
	id := imported.Contacts[0].ID
	jobs := NewContactJobService(svc.db)
	before, err := jobs.List(id, vaultID)
	if err != nil || len(before) != 1 || before[0].JobPosition != "Engineer" {
		t.Fatalf("file import lost grouped title: %+v %v", before, err)
	}
	var user models.User
	if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	var markerBefore models.Contact
	if err := svc.db.First(&markerBefore, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	lastSync := time.Now()
	for _, title := range []string{"Architect", "Director"} {
		card, err := vcard.NewDecoder(strings.NewReader(strings.ReplaceAll(wire, "Engineer", title))).Decode()
		if err != nil {
			t.Fatal(err)
		}
		err = svc.db.Transaction(func(tx *gorm.DB) error {
			_, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, id, "/group.vcf", title, &lastSync)
			if err == nil && action != "updated" {
				t.Fatalf("remote edit falsely marked local: %s", action)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		after, err := jobs.List(id, vaultID)
		if err != nil || len(after) != 1 || after[0].ID != before[0].ID || after[0].JobPosition != title {
			t.Fatalf("subscription lost grouped title or identity: %+v %v", after, err)
		}
		var stored models.Contact
		if err := svc.db.First(&stored, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored.LastUpdatedAt, markerBefore.LastUpdatedAt) {
			t.Fatal("remote update changed local conflict marker")
		}
	}
}
