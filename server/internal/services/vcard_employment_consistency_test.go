package services

import (
	"context"
	"errors"
	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"testing"
	"time"
)

func employmentConsistencyCard(t *testing.T, fields string) vcard.Card {
	t.Helper()
	card, err := vcard.NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Synthetic Employment\r\n" + fields + "END:VCARD\r\n")).Decode()
	if err != nil {
		t.Fatal(err)
	}
	return card
}
func TestVCardRejectsSharedUngroupedTitle(t *testing.T) {
	for _, fields := range []string{
		"ORG:Acme\r\nORG:Beta\r\nTITLE:Engineer\r\n",
		"item1.ORG:Acme\r\nITEM1.ORG:Beta\r\nitem1.TITLE:Engineer\r\n",
		"ORG:Acme\r\nTITLE:Engineer\r\nTITLE:Consultant\r\n",
	} {
		t.Run(strings.ReplaceAll(fields, "\r\n", "/"), func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			jobs := NewContactJobService(svc.db)
			companies := []models.Company{{VaultID: vault, Name: "Acme"}, {VaultID: vault, Name: "Beta"}}
			for i := range companies {
				if err := svc.db.Create(&companies[i]).Error; err != nil {
					t.Fatal(err)
				}
				title := "Engineer"
				if i == 1 {
					title = "Consultant"
				}
				if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: companies[i].ID, JobPosition: title}); err != nil {
					t.Fatal(err)
				}
			}
			before, err := jobs.List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			var account models.User
			if err := svc.db.First(&account, "id = ?", user).Error; err != nil {
				t.Fatal(err)
			}
			card := employmentConsistencyCard(t, fields)
			err = svc.db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, id, vault, account.AccountID) })
			if !errors.Is(err, ErrVCardUnsafeReplacement) {
				t.Fatalf("ambiguous TITLE accepted: %v", err)
			}
			after, err := jobs.List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected write changed employment identity, title or timestamps")
			}
		})
	}
}
func TestVCardPreservesLegacyEmployment(t *testing.T) {
	for _, entry := range []string{"shared replacement", "subscription"} {
		t.Run(entry, func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			company := models.Company{VaultID: vault, Name: "Legacy Acme"}
			if err := svc.db.Create(&company).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": company.ID, "job_position": "Engineer"}).Error; err != nil {
				t.Fatal(err)
			}
			var account models.User
			if err := svc.db.First(&account, "id = ?", user).Error; err != nil {
				t.Fatal(err)
			}
			card := employmentConsistencyCard(t, "ORG:New Beta\r\nTITLE:Consultant\r\n")
			replace := func() error {
				return svc.db.Transaction(func(tx *gorm.DB) error {
					if entry == "subscription" {
						_, action, err := svc.UpsertContactFromVCard(tx, card, vault, user, account.AccountID, id, "/legacy.vcf", "new", nil)
						if err == nil && action != "updated" {
							return errors.New("subscription did not update")
						}
						return err
					}
					return ReplaceContactVCardFields(tx, card, id, vault, account.AccountID)
				})
			}
			if err := replace(); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadContactEmployment(svc.db, id, vault)
			if err != nil {
				t.Fatal(err)
			}
			projected := ContactVCardEmployments(loaded)
			if len(projected) != 2 || projected[0].Company != "New Beta" || projected[0].Title != "Consultant" || projected[1].CompanyID != company.ID || projected[1].Title != "Engineer" {
				t.Fatalf("replacement lost legacy relationship or new selection: %+v", projected)
			}
			if err := svc.db.Transaction(func(tx *gorm.DB) error { return ReplaceContactVCardFields(tx, card, id, vault, account.AccountID) }); err != nil {
				t.Fatal(err)
			}
			after, err := loadContactEmployment(svc.db, id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(projected, ContactVCardEmployments(after)) {
				t.Fatal("repeat write created or changed employment identities")
			}
		})
	}
}
func subscribedEmploymentFixture(t *testing.T) (*VCardService, string, string, string, string, vcard.Card) {
	t.Helper()
	svc, _, vault, user := setupVCardTest(t)
	var account models.User
	if err := svc.db.First(&account, "id = ?", user).Error; err != nil {
		t.Fatal(err)
	}
	card := employmentConsistencyCard(t, "ORG:Acme\r\nTITLE:Engineer\r\n")
	var id string
	err := svc.db.Transaction(func(tx *gorm.DB) error {
		var err error
		id, _, err = svc.UpsertContactFromVCard(tx, card, vault, user, account.AccountID, "", "/employment.vcf", "initial", nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, id, vault, user, account.AccountID, card
}
func TestVCardCompanyRenameWinsSubscription(t *testing.T) {
	svc, id, vault, user, account, card := subscribedEmploymentFixture(t)
	jobs := NewContactJobService(svc.db)
	before, err := jobs.List(id, vault)
	if err != nil || len(before) != 1 {
		t.Fatal(before, err)
	}
	lastSync := time.Now().Add(-time.Second)
	if _, err := NewCompanyService(svc.db).Update(before[0].CompanyID, vault, dto.UpdateCompanyRequest{Name: "Beta"}); err != nil {
		t.Fatal(err)
	}
	card.SetValue(vcard.FieldTelephone, "+12025550123")
	var action string
	err = svc.db.Transaction(func(tx *gorm.DB) error {
		var err error
		_, action, err = svc.UpsertContactFromVCard(tx, card, vault, user, account, id, "/employment.vcf", "phone-changed", &lastSync)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("action=%s jobs=%+v", action, after)
	if action != "conflict_local_wins" || len(after) != 1 || after[0].CompanyID != before[0].CompanyID || after[0].CompanyName != "Beta" {
		t.Fatal("subscription recreated old company after local rename")
	}
}

// Hold the local deletion uncommitted while the pull reaches its first conflicting
// database operation. Both the broken late Save and the correct early read block,
// so releasing the transaction does not depend on timing or an implementation hook.
func TestVCardConcurrentEmploymentDeletionWinsSubscription(t *testing.T) {
	for _, lookup := range []string{"subscription ID", "legacy URI"} {
		t.Run(lookup, func(t *testing.T) {
			svc, id, vault, user, account, card := subscribedEmploymentFixture(t)
			if svc.db.Dialector.Name() != "postgres" {
				t.Skip("requires concurrent PostgreSQL transactions")
			}
			before, err := NewContactJobService(svc.db).List(id, vault)
			if err != nil || len(before) != 1 {
				t.Fatal(before, err)
			}
			lastSync := time.Now().Add(-time.Second)
			local := svc.db.Begin()
			if local.Error != nil {
				t.Fatal(local.Error)
			}
			defer local.Rollback()
			if err := NewContactJobService(local).Delete(id, vault, before[0].ID); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := make(chan error, 1)
			var action string
			existingID := id
			if lookup == "legacy URI" {
				existingID = ""
			}
			go func() {
				done <- svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					var err error
					_, action, err = svc.UpsertContactFromVCard(tx, card, vault, user, account, existingID, "/employment.vcf", "remote-changed", &lastSync)
					return err
				})
			}()
			waiting := false
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				var count int64
				err = svc.db.Raw(`SELECT COUNT(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error
				if err != nil {
					break
				}
				if count > 0 {
					waiting = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			commitErr := local.Commit().Error
			subscriptionErr := <-done
			if err != nil || commitErr != nil || subscriptionErr != nil || !waiting {
				t.Fatalf("scheduling: query=%v commit=%v subscription=%v waiting=%v", err, commitErr, subscriptionErr, waiting)
			}
			after, err := NewContactJobService(svc.db).List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			var contact models.Contact
			if err := svc.db.First(&contact, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if action != "conflict_local_wins" || len(after) != 0 || contact.LastUpdatedAt == nil || !contact.LastUpdatedAt.After(lastSync) {
				t.Fatalf("stale pull restored deleted employment: action=%s jobs=%+v marker=%v", action, after, contact.LastUpdatedAt)
			}
		})
	}
}
