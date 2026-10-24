package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// An incoming contact already owns the vault's FK KEY SHARE before resolving
// its company. Deletion must not hold resolution exclusion while waiting on it.
func TestVaultCascadeWithIncomingCompanyContact(t *testing.T) {
	for _, entry := range []string{"import", "subscription"} {
		t.Run(entry, func(t *testing.T) {
			syncer, clients, cards, vault, user, account := setupDavSyncTest(t)
			if cards.db.Dialector.Name() != "postgres" {
				t.Skip("requires real PostgreSQL foreign-key locks")
			}
			installCompanyForeignKeys(t, cards.db)
			for _, relation := range []string{"Contacts", "Companies"} {
				if err := cards.db.Migrator().CreateConstraint(&models.Vault{}, relation); err != nil {
					t.Fatal(err)
				}
				if !cards.db.Migrator().HasConstraint(&models.Vault{}, relation) {
					t.Fatal("missing vault FK", relation)
				}
			}
			other, err := NewVaultService(cards.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Unrelated incoming scope"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
			card := makeVCard("Incoming employee", "", "incoming-vault-boundary")
			card[vcard.FieldOrganization] = []*vcard.Field{vcard.NewOrganization([]string{"Incoming employer"})}
			card.SetValue(vcard.FieldTitle, "Incoming role")
			syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
				return []carddav.AddressObject{{Path: "/addressbooks/user/default/incoming.vcf", ETag: "incoming", Card: card}}, nil
			}}})
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			db := cards.db.WithContext(ctx)
			syncer.db = db
			inserted, resume := make(chan struct{}), make(chan struct{})
			var barrier, releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(resume) }) }
			const hook = "test:pause_incoming_contact_after_vault_fk"
			if err := db.Callback().Create().After("gorm:create").Register(hook, func(tx *gorm.DB) {
				if tx.Statement.Table == "contacts" {
					barrier.Do(func() {
						close(inserted)
						select {
						case <-resume:
						case <-ctx.Done():
							tx.AddError(ctx.Err())
						}
					})
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Create().Remove(hook)
			incomingDone, cascadeDone := make(chan error, 1), make(chan error, 1)
			incomingFinished, cascadeStarted, cascadeFinished := false, false, false
			defer func() {
				release()
				if !incomingFinished {
					<-incomingDone
				}
				if cascadeStarted && !cascadeFinished {
					<-cascadeDone
				}
			}()
			go func() {
				if entry == "subscription" {
					result, err := syncer.SyncSubscription(ctx, sub.ID, vault)
					if err == nil && (result.Errors != 0 || result.Created != 1) {
						err = fmt.Errorf("incoming pull failed: %+v", result)
					}
					incomingDone <- err
				} else {
					result, err := NewVCardService(db).ImportVCard(vault, user, strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Incoming employee\r\nORG:Incoming employer\r\nTITLE:Incoming role\r\nEND:VCARD\r\n"))
					if err == nil && (result.ImportedCount != 1 || len(result.Errors) != 0) {
						err = fmt.Errorf("incoming import failed: %+v", result)
					}
					incomingDone <- err
				}
			}()
			select {
			case <-inserted:
			case err := <-incomingDone:
				incomingFinished = true
				t.Fatalf("incoming missed boundary: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			cascadeStarted = true
			go func() { cascadeDone <- NewVaultService(db).DeleteVault(vault) }()
			deadline := time.NewTimer(5 * time.Second)
			defer deadline.Stop()
			tick := time.NewTicker(10 * time.Millisecond)
			defer tick.Stop()
			waiting := false
			for !waiting {
				select {
				case err := <-cascadeDone:
					cascadeFinished = true
					t.Fatalf("cascade did not wait for admitted contact: %v", err)
				case <-deadline.C:
					t.Fatal("vault FK wait not observed")
				case <-tick.C:
					var count int64
					if err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%vaults%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='vaults'::regclass)`).Scan(&count).Error; err != nil {
						t.Fatal(err)
					}
					waiting = count > 0
				}
			}
			release()
			incomingErr := <-incomingDone
			incomingFinished = true
			cascadeErr := <-cascadeDone
			cascadeFinished = true
			if incomingErr != nil || cascadeErr != nil {
				t.Fatalf("incoming employment/cascade must serialize: incoming=%v cascade=%v", incomingErr, cascadeErr)
			}
			for _, model := range []any{&models.Vault{}, &models.Contact{}, &models.Company{}} {
				q := db.Unscoped().Model(model)
				if _, ok := model.(*models.Vault); ok {
					q = q.Where("id = ?", vault)
				} else {
					q = q.Where("vault_id = ?", vault)
				}
				var count int64
				if err := q.Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("cascade left %T: count=%d err=%v", model, count, err)
				}
			}
			var jobs int64
			if err := db.Model(&models.ContactCompany{}).Count(&jobs).Error; err != nil || jobs != 0 {
				t.Fatalf("orphan jobs=%d %v", jobs, err)
			}
			var kept models.Vault
			if err := db.First(&kept, "id = ?", other.ID).Error; err != nil || kept.Name != "Unrelated incoming scope" {
				t.Fatalf("unrelated vault changed: %v", err)
			}
		})
	}
}
