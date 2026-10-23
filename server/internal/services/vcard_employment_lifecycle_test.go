package services

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func TestVCardUngroupedTitleWithGroupedOrganization(t *testing.T) {
	svc, _, vaultID, userID := setupVCardTest(t)
	result, err := svc.ImportVCard(vaultID, userID, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nitem1.ORG:Acme\r\nTITLE:Engineer\r\nEND:VCARD\r\n"))
	if err != nil || len(result.Contacts) != 1 {
		t.Fatalf("import: %+v %v", result, err)
	}
	wire, err := svc.ExportContact(result.Contacts[0].ID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), "TITLE:Engineer") {
		t.Fatal("existing ungrouped TITLE was discarded by ORG reconciliation")
	}
	jobs := NewContactJobService(svc.db)
	before, err := jobs.List(result.Contacts[0].ID, vaultID)
	if err != nil || len(before) != 1 || before[0].JobPosition != "Engineer" {
		t.Fatalf("imported jobs: %+v %v", before, err)
	}
	var user models.User
	if err := svc.db.First(&user, "id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic\r\nitem1.ORG:Acme\r\nTITLE:Architect\r\nEND:VCARD\r\n")).Decode()
	if err != nil {
		t.Fatal(err)
	}
	err = svc.db.Transaction(func(tx *gorm.DB) error {
		_, action, err := svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, result.Contacts[0].ID, "/group-label.vcf", "remote-title-edit", nil)
		if err == nil && action != "updated" {
			t.Fatalf("subscription action: %s", action)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := jobs.List(result.Contacts[0].ID, vaultID)
	if err != nil || len(after) != 1 || after[0].ID != before[0].ID || after[0].JobPosition != "Architect" {
		t.Fatalf("subscription lost title or identity: %+v %v", after, err)
	}
}

func TestVCardLegacyTitleOnlyUpdate(t *testing.T) {
	svc, contactID, vaultID, userID := setupVCardTest(t)
	company := models.Company{VaultID: vaultID, Name: "Acme"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	if _, err := jobs.LegacyUpdate(contactID, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	before, err := jobs.List(contactID, vaultID)
	if err != nil || len(before) != 1 {
		t.Fatalf("initial jobs: %+v %v", before, err)
	}
	response, err := jobs.LegacyUpdate(contactID, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: nil, JobPosition: "Architect"})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := svc.ExportContact(contactID, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	after, err := jobs.List(contactID, vaultID)
	if err != nil || len(after) != 1 || after[0].ID != before[0].ID || after[0].CompanyID != company.ID || after[0].JobPosition != "Architect" {
		t.Fatalf("legacy edit did not update existing employment: %+v %v", after, err)
	}
	if response.JobPosition != "Architect" || response.CompanyID == nil || *response.CompanyID != company.ID {
		t.Fatal("response differs from canonical employment")
	}
	if !strings.Contains(string(wire), "TITLE:Architect") {
		t.Fatal("successful legacy update is ignored by shared projection")
	}
}
func TestVCardCompanyDeletionWinsAgainstSubscription(t *testing.T) {
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
		id, _, err = svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, "", "/deleted-company.vcf", "initial", nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := NewContactJobService(svc.db).List(id, vaultID)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("initial jobs: %+v %v", jobs, err)
	}
	lastSync := time.Now().Add(-time.Millisecond)
	if err := NewCompanyService(svc.db).Delete(jobs[0].CompanyID, vaultID); err != nil {
		t.Fatal(err)
	}
	remaining, err := NewContactJobService(svc.db).List(id, vaultID)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("delete did not remove job: %+v %v", remaining, err)
	}
	card.SetValue(vcard.FieldTelephone, "+12025550123")
	var action string
	err = svc.db.Transaction(func(tx *gorm.DB) error {
		var err error
		_, action, err = svc.UpsertContactFromVCard(tx, card, vaultID, userID, user.AccountID, id, "/deleted-company.vcf", "phone-edited", &lastSync)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := NewContactJobService(svc.db).List(id, vaultID)
	if err != nil {
		t.Fatal(err)
	}
	var companies int64
	if err := svc.db.Model(&models.Company{}).Where("vault_id = ?", vaultID).Count(&companies).Error; err != nil {
		t.Fatal(err)
	}
	if action != "conflict_local_wins" || len(after) != 0 || companies != 0 {
		t.Fatal("subscription recreated locally deleted company and employment")
	}
}
