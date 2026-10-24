package services

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type companyRecipientDiscoveryContext struct{}

// Discovery completes first, then a real pull
// commits a new employment before the writer tries the resolution lock.
func TestCompanyMutationIncludesPullCommittedAfterRecipientDiscovery(t *testing.T) {
	for _, operation := range []string{"delete", "rename"} {
		t.Run(operation, func(t *testing.T) {
			f := setupJobSubscription(t)
			if f.push.db.Dialector.Name() != "postgres" {
				t.Skip("requires PostgreSQL row locks and real foreign keys")
			}
			installCompanyForeignKeys(t, f.push.db)
			id := f.contact(t, "Synthetic boundary employee")
			companyID := f.company(t, "Boundary employer")
			unrelatedID := f.contact(t, "Unrelated contact")
			var unrelatedBefore models.Contact
			if err := f.push.db.First(&unrelatedBefore, "id = ?", unrelatedID).Error; err != nil {
				t.Fatal(err)
			}
			pullSub := createPushSubscription(t, f.push.clientService, f.vaultID, f.userID, SyncWayPull)
			path := "/addressbooks/user/default/boundary-employee.vcf"
			if err := upsertContactSubscriptionState(f.push.db, id, pullSub.ID, path, "before-employment"); err != nil {
				t.Fatal(err)
			}
			card := makeVCard("Synthetic boundary employee", "", id)
			card[vcard.FieldOrganization] = []*vcard.Field{vcard.NewOrganization([]string{"Boundary employer"})}
			card.SetValue(vcard.FieldTitle, "Pulled role")
			syncer := NewDavSyncService(f.push.db, f.push.clientService, f.push.vcardService)
			syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
				queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
					return []carddav.AddressObject{{Path: path, ETag: "with-employment", Card: card}}, nil
				},
			}})
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			f.companies.db = f.companies.db.WithContext(context.WithValue(ctx, companyRecipientDiscoveryContext{}, true))
			discovered, resume, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			var first atomic.Bool
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(resume) }) }
			const hook = "test:pause_after_company_recipient_discovery"
			if err := f.push.db.Callback().Query().After("gorm:query").Register(hook, func(tx *gorm.DB) {
				if tx.Statement.Context.Value(companyRecipientDiscoveryContext{}) == true && tx.Statement.Table == "contacts" && first.CompareAndSwap(false, true) {
					t.Logf("completed recipient discovery: rows=%d sql=%s", tx.RowsAffected, tx.Statement.SQL.String())
					close(discovered)
					select {
					case <-resume:
					case <-ctx.Done():
						tx.AddError(ctx.Err())
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			finished := false
			defer func() {
				release()
				if !finished {
					<-done
				}
				f.push.db.Callback().Query().Remove(hook)
			}()
			go func() {
				if operation == "delete" {
					done <- f.companies.Delete(companyID, f.vaultID)
				} else {
					_, err := f.companies.Update(companyID, f.vaultID, dto.UpdateCompanyRequest{Name: "Renamed boundary employer"})
					done <- err
				}
			}()
			select {
			case <-discovered:
			case err := <-done:
				finished = true
				t.Fatalf("writer exited before discovery: %v", err)
			case <-ctx.Done():
				t.Fatal("writer did not reach discovery")
			}
			pulled, err := syncer.SyncSubscription(ctx, pullSub.ID, f.vaultID)
			if err != nil || pulled.Errors != 0 || pulled.Updated != 1 {
				t.Fatalf("real pull did not commit in discovery window: result=%+v error=%v", pulled, err)
			}
			var before models.Contact
			if err := f.push.db.First(&before, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			var originalJob models.ContactCompany
			if err := f.push.db.First(&originalJob, "contact_id = ? AND company_id = ?", id, companyID).Error; err != nil {
				t.Fatal(err)
			}
			if before.CompanyID == nil || *before.CompanyID != companyID || before.PreferredJobID == nil || *before.PreferredJobID != originalJob.ID || ptrToStr(before.JobPosition) != "Pulled role" {
				t.Fatalf("pull did not select persisted employment: %+v", before)
			}
			var beforeJobs int64
			if err := f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ? AND company_id = ?", id, companyID).Count(&beforeJobs).Error; err != nil || beforeJobs != 1 {
				t.Fatalf("pull did not persist real employment: count=%d error=%v", beforeJobs, err)
			}
			release()
			writerErr := <-done
			finished = true
			if writerErr != nil {
				t.Fatal(writerErr)
			}
			var after models.Contact
			if err := f.push.db.First(&after, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			exported, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
			if err != nil {
				t.Fatal(err)
			}
			var jobs int64
			if err := f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			t.Logf("after %s: jobs=%d company=%v preferred=%v legacy_title=%q exported_org=%q exported_title=%q local_before=%v local_after=%v", operation, jobs, after.CompanyID, after.PreferredJobID, ptrToStr(after.JobPosition), exported.Value(vcard.FieldOrganization), exported.Value(vcard.FieldTitle), before.LastUpdatedAt, after.LastUpdatedAt)
			if operation == "delete" && (jobs != 0 || after.CompanyID != nil || after.PreferredJobID != nil || after.JobPosition != nil || exported.Value(vcard.FieldTitle) != "") {
				t.Error("company deletion did not fully clear the employment committed after discovery")
			}
			if operation == "rename" && (jobs != 1 || exported.Value(vcard.FieldOrganization) != "Renamed boundary employer" || exported.Value(vcard.FieldTitle) != "Pulled role") {
				t.Error("company rename changed employment identity or projected values")
			}
			if after.LastUpdatedAt == nil || (before.LastUpdatedAt != nil && !after.LastUpdatedAt.After(*before.LastUpdatedAt)) {
				t.Error("company mutation omitted the newly associated contact's local-edit marker")
			}
			if ptrToStr(after.FirstName) != ptrToStr(before.FirstName) {
				t.Error("company mutation changed unrelated name")
			}
			if operation == "rename" {
				var preserved models.ContactCompany
				if err := f.push.db.First(&preserved, originalJob.ID).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(originalJob, preserved) || !reflect.DeepEqual(before.CompanyID, after.CompanyID) || !reflect.DeepEqual(before.PreferredJobID, after.PreferredJobID) || !reflect.DeepEqual(before.JobPosition, after.JobPosition) {
					t.Error("rename replaced employment identity or compatibility fields")
				}
				f.expect(t, map[string][2]string{id: {"Renamed boundary employer", "Pulled role"}})
			} else {
				if exported.Value(vcard.FieldOrganization) != "" {
					t.Error("deleted employer still exported")
				}
				f.expect(t, map[string][2]string{id: {"", ""}})
			}
			var unrelatedAfter models.Contact
			if err := f.push.db.First(&unrelatedAfter, "id = ?", unrelatedID).Error; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(unrelatedBefore, unrelatedAfter) {
				t.Error("company mutation changed unrelated contact")
			}
			f.expectNoPush(t)
		})
	}
}
