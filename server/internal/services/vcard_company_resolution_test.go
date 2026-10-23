package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestVCardCompanyCanonicalNameImport(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(map[bool]string{false: "reuse whitespace employer", true: "reject normalized ambiguity"}[ambiguous], func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			cs := NewCompanyService(svc.db)
			// Cross multiple pages before and between candidates: normalization
			// must examine the whole Vault, not only a SQL-limited prefix.
			var unrelated []models.Company
			for i := 0; i < 130; i++ {
				unrelated = append(unrelated, models.Company{VaultID: vault, Name: fmt.Sprintf("Unrelated employer %d", i)})
			}
			if err := svc.db.Create(&unrelated).Error; err != nil {
				t.Fatal(err)
			}
			original, err := cs.Create(vault, dto.CreateCompanyRequest{Name: "\u2003\t Acme \u00a0"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = NewContactJobService(svc.db).Create(id, vault, dto.CreateContactJobRequest{CompanyID: original.ID, JobPosition: "Engineer"}); err != nil {
				t.Fatal(err)
			}
			card, err := svc.ExportContactToVCard(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if card.Value(vcard.FieldOrganization) != "Acme" {
				t.Fatal("fixture did not use canonical exported name")
			}
			var wire bytes.Buffer
			if err = vcard.NewEncoder(&wire).Encode(card); err != nil {
				t.Fatal(err)
			}
			if ambiguous {
				for i := range unrelated {
					unrelated[i].ID = 0
					unrelated[i].Name = fmt.Sprintf("Another employer %d", i)
				}
				if err := svc.db.Create(&unrelated).Error; err != nil {
					t.Fatal(err)
				}
				if _, err = cs.Create(vault, dto.CreateCompanyRequest{Name: "Acme"}); err != nil {
					t.Fatal(err)
				}
			}
			count := func(m any) int64 {
				var n int64
				if err := svc.db.Model(m).Count(&n).Error; err != nil {
					t.Fatal(err)
				}
				return n
			}
			var before models.Contact
			if err := svc.db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			beforeContacts, beforeCompanies, beforeJobs := count(&models.Contact{}), count(&models.Company{}), count(&models.ContactCompany{})
			input := wire.String()
			if ambiguous {
				input = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Earlier valid contact\r\nORG:Earlier new employer\r\nEND:VCARD\r\n" + input
			}
			imported, err := svc.ImportVCard(vault, user, strings.NewReader(input))
			if ambiguous {
				if !errors.Is(err, ErrVCardUnsafeReplacement) {
					t.Errorf("normalized ambiguous company accepted: err=%v imported=%v", err, imported != nil)
				}
				if count(&models.Contact{}) != beforeContacts || count(&models.Company{}) != beforeCompanies || count(&models.ContactCompany{}) != beforeJobs {
					t.Error("ambiguous input committed domain rows")
				}
				var after models.Contact
				if err := svc.db.First(&after, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Error("ambiguous import modified original contact")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if imported.ImportedCount != 1 || len(imported.Contacts) != 1 {
				t.Fatal("unexpected import response")
			}
			jobs, err := NewContactJobService(svc.db).List(imported.Contacts[0].ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			var originalAfter models.Company
			if err := svc.db.First(&originalAfter, original.ID).Error; err != nil {
				t.Fatal(err)
			}
			if originalAfter.Name != original.Name {
				t.Error("resolution changed stored company spelling")
			}
			roundTrip, err := svc.ExportContactToVCard(imported.Contacts[0].ID, vault)
			if err != nil {
				t.Fatal(err)
			}
			if roundTrip.Value(vcard.FieldOrganization) != "Acme" || roundTrip.Value(vcard.FieldTitle) != "Engineer" {
				t.Error("imported employment did not round trip")
			}
			if count(&models.Company{}) != beforeCompanies || len(jobs) != 1 || jobs[0].CompanyID != original.ID {
				t.Error("export/import manufactured another company identity instead of resolving the canonical employer")
			}
		})
	}
}

// A real subscription pauses after reading the candidates. Company mutations
// must serialize without holding employee locks while waiting for resolution.
func TestVCardCompanyMutationDuringResolution(t *testing.T) {
	for _, mutation := range []string{"rename selected", "rename into ambiguity", "create duplicate"} {
		t.Run(mutation, func(t *testing.T) {
			svc, id, vault, user := setupVCardTest(t)
			if svc.db.Dialector.Name() != "postgres" {
				t.Skip("requires PostgreSQL transactions and observable advisory locks")
			}
			cs := NewCompanyService(svc.db)
			company, err := cs.Create(vault, dto.CreateCompanyRequest{Name: "Acme"})
			if err != nil {
				t.Fatal(err)
			}
			other, err := cs.Create(vault, dto.CreateCompanyRequest{Name: "Other"})
			if err != nil {
				t.Fatal(err)
			}
			var owner models.User
			if err := svc.db.First(&owner, "id = ?", user).Error; err != nil {
				t.Fatal(err)
			}
			var before models.Contact
			if err := svc.db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			lookedUp, resume := make(chan struct{}), make(chan struct{})
			var paused atomic.Bool
			const hook = "employment:pause_company_candidates"
			if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
				if tx.Statement.Context.Value(companyResolutionTestContact{}) != id || tx.Statement.Table != "companies" {
					return
				}
				if _, ok := tx.Statement.Dest.(*[]models.Company); !ok {
					return
				}
				if paused.CompareAndSwap(false, true) {
					close(lookedUp)
					select {
					case <-resume:
					case <-ctx.Done():
						tx.AddError(ctx.Err())
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer svc.db.Callback().Query().Remove(hook)
			card := vcard.Card{vcard.FieldVersion: []*vcard.Field{{Value: "3.0"}}, vcard.FieldFormattedName: []*vcard.Field{{Value: "Synthetic subscription"}}, vcard.FieldOrganization: []*vcard.Field{vcard.NewOrganization([]string{"Acme"})}, vcard.FieldTitle: []*vcard.Field{{Value: "Engineer"}}}
			syncStarted := time.Now()
			imported := make(chan error, 1)
			go func() {
				imported <- svc.db.WithContext(context.WithValue(ctx, companyResolutionTestContact{}, id)).Transaction(func(tx *gorm.DB) error {
					_, _, err := svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, id, "/company-resolution.vcf", "accepted-remote", nil)
					return err
				})
			}()
			select {
			case <-lookedUp:
			case <-ctx.Done():
				close(resume)
				<-imported
				t.Fatal("candidate lookup barrier missing")
			}
			changed := make(chan error, 1)
			pid := make(chan int, 1)
			go func() {
				changed <- svc.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
					var n int
					if err := conn.Raw("SELECT pg_backend_pid()").Scan(&n).Error; err != nil {
						return err
					}
					pid <- n
					service := NewCompanyService(conn)
					switch mutation {
					case "rename selected":
						_, err := service.Update(company.ID, vault, dto.UpdateCompanyRequest{Name: "Beta"})
						return err
					case "rename into ambiguity":
						_, err := service.Update(other.ID, vault, dto.UpdateCompanyRequest{Name: " Acme "})
						return err
					default:
						_, err := service.Create(vault, dto.CreateCompanyRequest{Name: "Acme"})
						return err
					}
				})
			}()
			var mutationPID int
			select {
			case mutationPID = <-pid:
			case err := <-changed:
				close(resume)
				<-imported
				t.Fatal(err)
			}
			ticker := time.NewTicker(10 * time.Millisecond)
			waiting, finished := false, false
			var changeErr, observeErr error
		observe:
			for {
				select {
				case changeErr = <-changed:
					finished = true
					break observe
				case <-ctx.Done():
					observeErr = ctx.Err()
					break observe
				case <-ticker.C:
					var n int64
					if err := svc.db.WithContext(ctx).Raw("SELECT count(*) FROM pg_locks WHERE pid = ? AND locktype = 'advisory' AND NOT granted", mutationPID).Scan(&n).Error; err != nil {
						observeErr = err
						break observe
					}
					if n > 0 {
						waiting = true
						break observe
					}
				}
			}
			ticker.Stop()
			close(resume)
			importErr := <-imported
			if !finished {
				changeErr = <-changed
			}
			if observeErr != nil {
				t.Fatal(observeErr)
			}
			if changeErr != nil {
				t.Fatal(changeErr)
			}
			if importErr != nil {
				t.Fatal(importErr)
			}
			if !waiting {
				t.Error("company mutation committed while subscription still used its candidate snapshot")
			}
			var stored models.Contact
			if err := svc.db.First(&stored, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			jobs, err := NewContactJobService(svc.db).List(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if len(jobs) != 1 || jobs[0].CompanyID != company.ID {
				t.Fatalf("wrong employment identity: %+v", jobs)
			}
			if ptrToStr(stored.DistantEtag) != "accepted-remote" {
				t.Error("subscription ETag not committed")
			}
			exported, err := svc.ExportContactToVCard(id, vault)
			if err != nil {
				t.Fatal(err)
			}
			if mutation == "rename selected" {
				if exported.Value(vcard.FieldOrganization) != "Beta" {
					t.Error("rename not reflected in export")
				}
				if stored.LastUpdatedAt == nil || before.LastUpdatedAt == nil || !stored.LastUpdatedAt.After(*before.LastUpdatedAt) {
					t.Error("rename after subscription did not mark employee locally modified")
				}
				if stored.LastUpdatedAt == nil || !stored.LastUpdatedAt.After(syncStarted) {
					t.Error("local rename does not outrank accepted subscription")
				}
				if err := svc.db.Transaction(func(tx *gorm.DB) error {
					_, action, err := svc.UpsertContactFromVCard(tx, card, vault, user, owner.AccountID, id, "/company-resolution.vcf", "later-remote", &syncStarted)
					if err == nil && action != "conflict_local_wins" {
						t.Errorf("next subscription overwrote local rename: %s", action)
					}
					return err
				}); err != nil {
					t.Fatal(err)
				}
				var afterConflict models.Contact
				if err := svc.db.First(&afterConflict, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stored, afterConflict) {
					t.Error("local-wins changed contact or advanced ETag")
				}
			} else {
				if exported.Value(vcard.FieldOrganization) != "Acme" {
					t.Error("unrelated mutation changed selected employer")
				}
				var wire bytes.Buffer
				if err := vcard.NewEncoder(&wire).Encode(card); err != nil {
					t.Fatal(err)
				}
				var countBefore int64
				if err := svc.db.Model(&models.Contact{}).Count(&countBefore).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := svc.ImportVCard(vault, user, strings.NewReader(wire.String())); !errors.Is(err, ErrVCardUnsafeReplacement) {
					t.Fatalf("subsequent ambiguous input accepted: %v", err)
				}
				var countAfter int64
				if err := svc.db.Model(&models.Contact{}).Count(&countAfter).Error; err != nil {
					t.Fatal(err)
				}
				if countAfter != countBefore {
					t.Error("ambiguous subsequent import left contact")
				}
			}
		})
	}
}
