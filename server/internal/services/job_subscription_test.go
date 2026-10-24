package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type jobSubscriptionPush struct {
	contactID, organization, title, path, etag string
	employments                                [][2]string
}
type jobSubscriptionFixture struct {
	push                                       *DavPushService
	jobs                                       *ContactJobService
	companies                                  *CompanyService
	contacts                                   *ContactService
	vaultID, userID, accountID, subscriptionID string
	received                                   chan jobSubscriptionPush
	remoteFailure                              atomic.Bool
}

func setupJobSubscription(t *testing.T) *jobSubscriptionFixture {
	t.Helper()
	push, clients, _, contacts, vaultID, userID, accountID := setupDavPushTest(t)
	sub := createPushSubscription(t, clients, vaultID, userID, SyncWayPush)
	f := &jobSubscriptionFixture{push: push, jobs: NewContactJobService(push.db), companies: NewCompanyService(push.db), contacts: contacts, vaultID: vaultID, userID: userID, accountID: accountID, subscriptionID: sub.ID, received: make(chan jobSubscriptionPush, 32)}
	var sequence atomic.Int64
	push.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
		etag := fmt.Sprintf("job-revision-%d", sequence.Add(1))
		f.received <- jobSubscriptionPush{card.Value(vcard.FieldUID), card.Value(vcard.FieldOrganization), card.Value(vcard.FieldTitle), path, etag, exportedJobPairs(card)}
		if f.remoteFailure.Load() {
			return nil, errors.New("synthetic DAV unavailable")
		}
		return &carddav.AddressObject{Path: path, ETag: etag}, nil
	}}})
	contacts.SetDavPushService(push)
	f.jobs.SetDavPushService(push)
	f.companies.SetDavPushService(push)
	return f
}

func (f *jobSubscriptionFixture) contact(t *testing.T, name string) string {
	t.Helper()
	c, err := f.contacts.CreateContact(f.vaultID, f.userID, dto.CreateContactRequest{FirstName: name})
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{c.ID: {"", ""}})
	return c.ID
}

func (f *jobSubscriptionFixture) company(t *testing.T, name string) uint {
	t.Helper()
	c, err := f.companies.Create(f.vaultID, dto.CreateCompanyRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return c.ID
}

func (f *jobSubscriptionFixture) expect(t *testing.T, want map[string][2]string) {
	t.Helper()
	for range len(want) {
		select {
		case got := <-f.received:
			projection, ok := want[got.contactID]
			matched := len(got.employments) == 0 && projection == [2]string{"", ""}
			for _, pair := range got.employments {
				if pair == projection {
					matched = true
				}
			}
			if !ok || !matched {
				t.Fatalf("unexpected push %+v, expected %v", got, want)
			}
			// Wait for the real asynchronous push to persist its result, not just enter the mock transport.
			release := f.push.operationLocks.lock(got.contactID)
			release()
			var state models.ContactSubscriptionState
			if err := f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", got.contactID, f.subscriptionID).First(&state).Error; err != nil {
				t.Fatal(err)
			}
			if state.DistantEtag != got.etag || state.DistantURI != got.path {
				t.Fatalf("push state not committed: %+v, transport %+v", state, got)
			}
			var count int64
			if err := f.push.db.Model(&models.DavSyncLog{}).Where("address_book_subscription_id = ? AND contact_id = ? AND action = ? AND distant_etag = ?", f.subscriptionID, got.contactID, "pushed", got.etag).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("successful push log count=%d", count)
			}
			delete(want, got.contactID)
		case <-time.After(5 * time.Second):
			t.Fatalf("persisted mutation did not trigger subscription push: %v", want)
		}
	}
}

func (f *jobSubscriptionFixture) expectNoPush(t *testing.T) {
	t.Helper()
	select {
	case got := <-f.received:
		t.Fatalf("unexpected push after rejected/duplicate/unrelated change: %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestJobSubscriptionLifecycle(t *testing.T) {
	f := setupJobSubscription(t)
	id := f.contact(t, "Employee")
	company := f.company(t, "First company")
	second := f.company(t, "Second company")
	job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"First company", "Engineer"}})
	if _, err = f.jobs.Update(id, f.vaultID, job.ID, dto.UpdateContactJobRequest{CompanyID: second, JobPosition: ""}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"Second company", ""}})
	if err = f.jobs.Delete(id, f.vaultID, job.ID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"", ""}})
	if _, err = f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Manager"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"First company", "Manager"}})
	if err = f.jobs.RemoveEmployee(company, f.vaultID, id); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"", ""}})
	if _, err = f.jobs.LegacyUpdate(id, f.vaultID, f.userID, dto.UpdateJobInfoRequest{CompanyID: &company, JobPosition: "Director"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"First company", "Director"}})
	if _, err = f.jobs.LegacyDelete(id, f.vaultID, f.userID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"", ""}})
	var count int64
	if err = f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("jobs not removed: count=%d err=%v", count, err)
	}
	var contact models.Contact
	if err = f.push.db.First(&contact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if contact.CompanyID != nil || contact.JobPosition != nil || ptrToStr(contact.FirstName) != "Employee" {
		t.Fatalf("legacy clear/unrelated name mismatch: %+v", contact)
	}
	f.expectNoPush(t)
}

func TestCompanySubscriptionLifecycle(t *testing.T) {
	f := setupJobSubscription(t)
	first := f.contact(t, "First employee")
	second := f.contact(t, "Second employee")
	legacy := f.contact(t, "Legacy employee")
	unrelated := f.contact(t, "Unrelated")
	company := f.company(t, "Shared company")
	previous := f.company(t, "Previous company")
	if _, err := f.jobs.Create(first, f.vaultID, dto.CreateContactJobRequest{CompanyID: previous, JobPosition: "Earlier role"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{first: {"Previous company", "Earlier role"}})
	for _, id := range []string{first, second, second} {
		if _, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Current role"}); err != nil {
			t.Fatal(err)
		}
		f.expect(t, map[string][2]string{id: {"Shared company", "Current role"}})
	}
	// Historical legacy-only data must receive company changes too.
	if err := f.push.db.Model(&models.Contact{}).Where("id = ?", legacy).Updates(map[string]any{"company_id": company, "job_position": "Legacy role"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "Renamed company"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{first: {"Renamed company", "Current role"}, second: {"Renamed company", "Current role"}, legacy: {"Renamed company", "Legacy role"}})
	f.expectNoPush(t) // duplicate jobs must not fan out duplicate pushes; unrelated contact is excluded.
	if _, err := f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "Renamed company", Type: "New type"}); err != nil {
		t.Fatal(err)
	}
	f.expectNoPush(t)
	if err := f.companies.Delete(company, f.vaultID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{first: {"Previous company", "Earlier role"}, second: {"", ""}, legacy: {"", ""}})
	var count int64
	if err := f.push.db.Model(&models.ContactCompany{}).Where("company_id = ?", company).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("company jobs remain: %d %v", count, err)
	}
	var contact models.Contact
	if err := f.push.db.First(&contact, "id = ?", legacy).Error; err != nil || contact.CompanyID != nil || contact.JobPosition != nil {
		t.Fatalf("legacy cleanup failed: %+v %v", contact, err)
	}
	var state models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", unrelated, f.subscriptionID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.DistantEtag != "job-revision-4" {
		t.Fatalf("unrelated state changed: %+v", state)
	}
	f.expectNoPush(t)
}

func TestJobSubscriptionRejectedAndRolledBackChanges(t *testing.T) {
	f := setupJobSubscription(t)
	id := f.contact(t, "Preserved")
	company := f.company(t, "Preserved company")
	job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Preserved role"})
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"Preserved company", "Preserved role"}})
	// Preserve real legacy fields too, so failed cleanup cannot silently erase them.
	if err = f.push.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": company, "job_position": "Preserved legacy role"}).Error; err != nil {
		t.Fatal(err)
	}
	other, err := NewVaultService(f.push.db).CreateVault(f.accountID, f.userID, dto.CreateVaultRequest{Name: "Other vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	var contactBeforeRejection models.Contact
	if err := f.push.db.First(&contactBeforeRejection, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = f.jobs.Create(id, other.ID, dto.CreateContactJobRequest{CompanyID: company}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("cross-vault create: %v", err)
	}
	if _, err = f.jobs.Update(id, f.vaultID, job.ID, dto.UpdateContactJobRequest{CompanyID: company + 10000}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("missing company update: %v", err)
	}
	if err = f.jobs.Delete(id, other.ID, job.ID); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("cross-vault delete: %v", err)
	}
	if _, err = f.jobs.AddEmployee(company, other.ID, dto.AddEmployeeRequest{ContactID: id}); !errors.Is(err, ErrContactNotFound) {
		t.Fatalf("cross-vault add: %v", err)
	}
	if err = f.jobs.RemoveEmployee(company, other.ID, id); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("cross-vault remove: %v", err)
	}
	if _, err = f.companies.Update(company, other.ID, dto.UpdateCompanyRequest{Name: "Forbidden"}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("cross-vault rename: %v", err)
	}
	if err = f.companies.Delete(company, other.ID); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("cross-vault company delete: %v", err)
	}
	injected := errors.New("synthetic company deletion failure")
	if err = f.push.db.Callback().Delete().Before("gorm:delete").Register("reject_company_deletion", func(tx *gorm.DB) {
		if tx.Statement.Table == "companies" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	err = f.companies.Delete(company, f.vaultID)
	if removeErr := f.push.db.Callback().Delete().Remove("reject_company_deletion"); removeErr != nil {
		t.Fatal(removeErr)
	}
	if !errors.Is(err, injected) {
		t.Fatalf("expected deletion failure: %v", err)
	}
	var stored models.ContactCompany
	if err = f.push.db.First(&stored, job.ID).Error; err != nil || stored.CompanyID != company || ptrToStr(stored.JobPosition) != "Preserved role" {
		t.Fatalf("failed delete changed job: %+v %v", stored, err)
	}
	var storedCompany models.Company
	if err = f.push.db.First(&storedCompany, company).Error; err != nil || storedCompany.Name != "Preserved company" {
		t.Fatalf("company changed: %+v %v", storedCompany, err)
	}
	// Fail after the legacy clear has removed join rows: both halves must roll back.
	if err = f.push.db.Callback().Update().Before("gorm:update").Register("reject_legacy_clear", func(tx *gorm.DB) {
		if tx.Statement.Table == "contacts" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err = f.jobs.LegacyDelete(id, f.vaultID, f.userID)
	if removeErr := f.push.db.Callback().Update().Remove("reject_legacy_clear"); removeErr != nil {
		t.Fatal(removeErr)
	}
	if !errors.Is(err, injected) {
		t.Fatalf("expected legacy clear failure: %v", err)
	}
	if err = f.push.db.First(&stored, job.ID).Error; err != nil {
		t.Fatalf("legacy clear lost job: %v", err)
	}
	var state models.ContactSubscriptionState
	if err = f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", id, f.subscriptionID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	var persistedContact models.Contact
	if err = f.push.db.First(&persistedContact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if contactBeforeRejection.LastUpdatedAt == nil || persistedContact.LastUpdatedAt == nil || !persistedContact.LastUpdatedAt.Equal(*contactBeforeRejection.LastUpdatedAt) {
		t.Fatal("rejected or rolled back job operation advanced local conflict marker")
	}
	if persistedContact.CompanyID == nil || *persistedContact.CompanyID != company || ptrToStr(persistedContact.JobPosition) != "Preserved legacy role" || ptrToStr(persistedContact.FirstName) != "Preserved" {
		t.Fatalf("rollback changed contact fields: %+v", persistedContact)
	}
	var jobCount int64
	if err = f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&jobCount).Error; err != nil || jobCount != 1 {
		t.Fatalf("rejected operation changed job count: %d %v", jobCount, err)
	}
	var pushLogs int64
	if err = f.push.db.Model(&models.DavSyncLog{}).Where("address_book_subscription_id = ?", f.subscriptionID).Count(&pushLogs).Error; err != nil || pushLogs != 2 {
		t.Fatalf("rejected operation wrote sync logs: %d %v", pushLogs, err)
	}
	if state.DistantEtag != "job-revision-2" {
		t.Fatalf("rejected mutations changed remote state: %+v", state)
	}
	f.expectNoPush(t)
}

func TestJobSubscriptionRemoteFailurePreservesLocalCommit(t *testing.T) {
	f := setupJobSubscription(t)
	id := f.contact(t, "Local commit")
	company := f.company(t, "Offline company")
	f.remoteFailure.Store(true)
	job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-f.received:
		if got.organization != "Offline company" || got.title != "Engineer" {
			t.Fatalf("wrong failed payload: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no push attempt")
	}
	release := f.push.operationLocks.lock(id)
	release()
	var stored models.ContactCompany
	if err = f.push.db.First(&stored, job.ID).Error; err != nil || stored.CompanyID != company {
		t.Fatalf("local job lost: %+v %v", stored, err)
	}
	var state models.ContactSubscriptionState
	if err = f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", id, f.subscriptionID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.DistantEtag != "job-revision-1" {
		t.Fatalf("failed push replaced state: %+v", state)
	}
	var logs []models.DavSyncLog
	if err = f.push.db.Where("address_book_subscription_id = ? AND contact_id = ? AND action = ?", f.subscriptionID, id, "error").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ErrorMessage == nil || *logs[0].ErrorMessage == "" {
		t.Fatalf("missing remote failure log: %+v", logs)
	}
}

func TestJobSubscriptionWithoutActivePushSubscriber(t *testing.T) {
	for _, mode := range []string{"absent", "pull only", "inactive"} {
		t.Run(mode, func(t *testing.T) {
			f := setupJobSubscription(t)
			var err error
			switch mode {
			case "absent":
				err = f.push.db.Delete(&models.AddressBookSubscription{}, "id = ?", f.subscriptionID).Error
			case "pull only":
				err = f.push.db.Model(&models.AddressBookSubscription{}).Where("id = ?", f.subscriptionID).Update("sync_way", SyncWayPull).Error
			case "inactive":
				err = f.push.db.Model(&models.AddressBookSubscription{}).Where("id = ?", f.subscriptionID).Update("active", false).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			contact, err := f.contacts.CreateContact(f.vaultID, f.userID, dto.CreateContactRequest{FirstName: "Local employee"})
			if err != nil {
				t.Fatal(err)
			}
			company := f.company(t, "Local company")
			job, err := f.jobs.Create(contact.ID, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Local role"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "Local rename"}); err != nil {
				t.Fatal(err)
			}
			f.expectNoPush(t)
			var stored models.ContactCompany
			if err = f.push.db.First(&stored, job.ID).Error; err != nil || ptrToStr(stored.JobPosition) != "Local role" {
				t.Fatalf("local write lost: %+v %v", stored, err)
			}
			card, err := NewVCardService(f.push.db).ExportContactToVCard(contact.ID, f.vaultID)
			if err != nil {
				t.Fatal(err)
			}
			assertVCardJob(t, card, "Local rename", "Local role")
			for _, model := range []any{&models.ContactSubscriptionState{}, &models.DavSyncLog{}} {
				var count int64
				if err = f.push.db.Model(model).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("non-push subscriber acquired sync state/log: %T %d %v", model, count, err)
				}
			}
		})
	}
}

func TestJobSubscriptionExportsAllEmploymentsWhenCardDAVFiltered(t *testing.T) {
	f := setupJobSubscription(t)
	id := f.contact(t, "Filtered employee")
	first := f.company(t, "Selected company")
	second := f.company(t, "Secondary; company")
	expectAll := func(want ...[2]string) {
		t.Helper()
		select {
		case got := <-f.received:
			if got.contactID != id || !reflect.DeepEqual(got.employments, want) {
				t.Fatalf("push pairs=%v want=%v", got.employments, want)
			}
			// Retain the established URI/ETag and successful-log assertions too.
			f.received <- got
			pair := [2]string{}
			if len(want) > 0 {
				pair = want[0]
			}
			f.expect(t, map[string][2]string{id: pair})
		case <-time.After(5 * time.Second):
			t.Fatal("committed employment did not push")
		}
	}
	selected, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: first, JobPosition: "Selected role"})
	if err != nil {
		t.Fatal(err)
	}
	expectAll([2]string{"Selected company", "Selected role"})
	if _, err := f.jobs.AddEmployee(second, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Secondary role"}); err != nil {
		t.Fatal(err)
	}
	expectAll([2]string{"Selected company", "Selected role"}, [2]string{"Secondary; company", "Secondary role"})
	filter := models.CardDAVFilter{Match: "all", Fields: []string{"phone"}}
	if err := f.push.db.Model(&models.Vault{ID: f.vaultID}).Select("CardDAVFilter").Updates(models.Vault{CardDAVFilter: &filter}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.companies.Update(second, f.vaultID, dto.UpdateCompanyRequest{Name: "Renamed secondary"}); err != nil {
		t.Fatal(err)
	}
	expectAll([2]string{"Selected company", "Selected role"}, [2]string{"Renamed secondary", "Secondary role"})
	if err := f.jobs.Delete(id, f.vaultID, selected.ID); err != nil {
		t.Fatal(err)
	}
	expectAll([2]string{"Renamed secondary", "Secondary role"})
	var stored models.Contact
	if err := f.push.db.First(&stored, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PreferredJobID != nil && *stored.PreferredJobID == selected.ID {
		t.Fatal("removed job remains selected")
	}
	if err := f.companies.Delete(second, f.vaultID); err != nil {
		t.Fatal(err)
	}
	expectAll()
	f.expectNoPush(t)
}
