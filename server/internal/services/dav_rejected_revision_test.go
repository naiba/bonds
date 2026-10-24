package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// Regression: a rejected remote revision must not resurrect a deleted
// projected title when a server without sync tokens returns that same revision.
func TestRepeatedSubscriptionPullPreservesDeletedJob(t *testing.T) {
	for _, mode := range []string{"query", "full", "incremental"} {
		t.Run(mode, func(t *testing.T) { testRepeatedSubscriptionPullPreservesDeletedJob(t, mode) })
	}
}

func testRepeatedSubscriptionPullPreservesDeletedJob(t *testing.T, mode string) {
	syncer, clients, cards, vault, user, _ := setupDavSyncTest(t)
	sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
	contact, err := NewContactService(syncer.db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Local employee"})
	if err != nil {
		t.Fatal(err)
	}
	company, err := NewCompanyService(syncer.db).Create(vault, dto.CreateCompanyRequest{Name: "Employer"})
	if err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(syncer.db)
	job, err := jobs.Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Deleted position"})
	if err != nil {
		t.Fatal(err)
	}
	path := "/addressbooks/user/default/employee.vcf"
	if err := upsertContactSubscriptionState(syncer.db, contact.ID, sub.ID, path, "initial-remote"); err != nil {
		t.Fatal(err)
	}
	if err := clients.UpdateSyncStatus(sub.ID, nil); err != nil {
		t.Fatal(err)
	}
	stale, err := cards.ExportContactToVCard(contact.ID, vault)
	if err != nil {
		t.Fatal(err)
	}
	stale.SetName(&vcard.Name{GivenName: "Stale remote name"})
	stale.SetValue(vcard.FieldFormattedName, "Stale remote name")
	if err := jobs.Delete(contact.ID, vault, job.ID); err != nil {
		t.Fatal(err)
	}
	var before models.Contact
	if err := syncer.db.First(&before, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	remoteETag := "remote-change-before-local-deletion"
	token := ""
	if mode == "incremental" {
		token = "initial-token"
		if err := clients.UpdateSyncStatus(sub.ID, &token); err != nil {
			t.Fatal(err)
		}
		// Set the checkpoint before the deletion, as it would be in an existing subscription.
		if err := syncer.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("last_synchronized_at", before.LastUpdatedAt.Add(-time.Second)).Error; err != nil {
			t.Fatal(err)
		}
	}
	objects := func() []carddav.AddressObject {
		return []carddav.AddressObject{{Path: path, ETag: remoteETag, Card: stale}}
	}
	syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
		syncCollFn: func(_ context.Context, _ string, q *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			if mode == "query" {
				return nil, errors.New("synthetic server does not support sync-collection")
			}
			expected := token
			if mode == "full" {
				expected = ""
			}
			if q.SyncToken != expected {
				t.Fatalf("checkpoint token=%q want %q", q.SyncToken, expected)
			}
			return &carddav.SyncResponse{SyncToken: token + "-next", Updated: objects()}, nil
		},
		multiGetFn: func(_ context.Context, _ string, q *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			if !reflect.DeepEqual(q.Paths, []string{path}) {
				t.Fatalf("unexpected multiget: %v", q.Paths)
			}
			return objects(), nil
		},
		queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			return objects(), nil
		},
	}})
	var initialState models.ContactSubscriptionState
	var checkpoint models.AddressBookSubscription
	if err := syncer.db.Where("contact_id = ?", contact.ID).First(&initialState).Error; err != nil {
		t.Fatal(err)
	}
	if err := syncer.db.First(&checkpoint, "id = ?", sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	// A failed acknowledgement must not advance the global checkpoint or leave
	// half a mapping; retry must still see the local edit and reject the card.
	const callback = "reject_remote_conflict_acknowledgement"
	injected := errors.New("synthetic conflict acknowledgement failure")
	if err := syncer.db.Callback().Create().After("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_subscription_states" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syncer.db.Callback().Create().Remove(callback) })
	failed, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
	if err != nil || failed.Errors != 1 || failed.Updated != 0 || failed.Skipped != 0 {
		t.Fatalf("expected acknowledgement failure: %+v %v", failed, err)
	}
	syncer.db.Callback().Create().Remove(callback)
	var failedState models.ContactSubscriptionState
	var failedSub models.AddressBookSubscription
	if err := syncer.db.Where("contact_id = ?", contact.ID).First(&failedState).Error; err != nil {
		t.Fatal(err)
	}
	if err := syncer.db.First(&failedSub, "id = ?", sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(initialState, failedState) || !reflect.DeepEqual(checkpoint.LastSynchronizedAt, failedSub.LastSynchronizedAt) || !reflect.DeepEqual(checkpoint.DistantSyncToken, failedSub.DistantSyncToken) {
		t.Fatal("failed conflict acknowledgement changed mapping/checkpoint")
	}

	for round := 1; round <= 3; round++ {
		result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
		if err != nil || result == nil || result.Errors != 0 || result.Skipped != 1 || result.Updated != 0 || result.Created != 0 {
			t.Fatalf("round %d failed: %+v %v", round, result, err)
		}
		var status models.AddressBookSubscription
		if err := syncer.db.First(&status, "id = ?", sub.ID).Error; err != nil {
			t.Fatal(err)
		}
		if status.LastSynchronizedAt == nil || !status.LastSynchronizedAt.After(*before.LastUpdatedAt) {
			t.Fatal("successful conflict did not advance checkpoint")
		}
		if mode != "query" {
			token += "-next"
			if ptrToStr(status.DistantSyncToken) != token {
				t.Fatal("sync token not advanced")
			}
		}
		if mode == "full" {
			if err := syncer.db.Model(&status).Update("distant_sync_token", nil).Error; err != nil {
				t.Fatal(err)
			}
		}
		var state models.ContactSubscriptionState
		if err := syncer.db.Where("contact_id = ?", contact.ID).First(&state).Error; err != nil {
			t.Fatal(err)
		}
		expected := initialState
		expected.RejectedEtag = remoteETag
		if !reflect.DeepEqual(expected, state) {
			t.Fatalf("conflict changed accepted mapping: %+v", state)
		}
		var stored models.Contact
		if err := syncer.db.First(&stored, "id = ?", contact.ID).Error; err != nil {
			t.Fatal(err)
		}
		exported, err := cards.ExportContactToVCard(contact.ID, vault)
		if err != nil {
			t.Fatal(err)
		}
		var count int64
		if err := syncer.db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		t.Logf("round=%d updated=%d skipped=%d jobs=%d legacy_title=%q exported_title=%q", round, result.Updated, result.Skipped, count, ptrToStr(stored.JobPosition), exported.Value(vcard.FieldTitle))
		if count != 0 {
			t.Fatal("deleted job row recreated")
		}
		if !reflect.DeepEqual(before, stored) || exported.Value(vcard.FieldTitle) != "" {
			t.Fatalf("round %d accepted the previously rejected revision and restored deleted job title: legacy=%q exported=%q name=%q", round, ptrToStr(stored.JobPosition), exported.Value(vcard.FieldTitle), ptrToStr(stored.FirstName))
		}
	}

	// A distinct remote version can still update and clear legacy TITLE after the
	// conflict checkpoint. No local job row may be resurrected in either case.
	for i, title := range []string{"Legitimate remote title", ""} {
		remoteETag = fmt.Sprintf("legitimate-revision-%d", i)
		stale = makeVCard("Legitimate remote name", "", contact.ID)
		stale.SetValue(vcard.FieldTitle, title)
		result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
		if err != nil || result.Errors != 0 || result.Updated != 1 || result.Skipped != 0 {
			t.Fatalf("new version not accepted: %+v %v", result, err)
		}
		if mode != "query" {
			token += "-next"
		}
		if mode == "full" {
			if err := syncer.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("distant_sync_token", nil).Error; err != nil {
				t.Fatal(err)
			}
		}
		var stored models.Contact
		if err := syncer.db.First(&stored, "id = ?", contact.ID).Error; err != nil {
			t.Fatal(err)
		}
		if ptrToStr(stored.JobPosition) != title || ptrToStr(stored.FirstName) != "Legitimate remote name" || !stored.LastUpdatedAt.Equal(*before.LastUpdatedAt) {
			t.Fatal("new remote version or local marker incorrect")
		}
		exported, err := cards.ExportContactToVCard(contact.ID, vault)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJob(t, exported, "", title)
		var count int64
		if err := syncer.db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("local jobs changed: %d %v", count, err)
		}
		var state models.ContactSubscriptionState
		if err := syncer.db.Where("contact_id = ?", contact.ID).First(&state).Error; err != nil {
			t.Fatal(err)
		}
		if state.RejectedEtag != "" || state.DistantEtag != remoteETag || state.DistantURI != path {
			t.Fatal("accepted version did not finish conflict lifecycle")
		}
	}
}

func TestSubscriptionRejectedRevisionMigration(t *testing.T) {
	syncer, clients, _, vault, user, _ := setupDavSyncTest(t)
	sub := createPushSubscription(t, clients, vault, user, SyncWayPull)
	contact, err := NewContactService(syncer.db).CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Existing contact"})
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertContactSubscriptionState(syncer.db, contact.ID, sub.ID, "/contacts/existing.vcf", "existing-etag"); err != nil {
		t.Fatal(err)
	}
	// Explicit projection avoids reusing a PostgreSQL SELECT * plan across the
	// simulated column drop/re-add, which changes the physical column order.
	const mappingColumns = "id, contact_id, address_book_subscription_id, distant_uri, distant_etag, rejected_etag, created_at, updated_at"
	var before models.ContactSubscriptionState
	if err := syncer.db.Select(mappingColumns).Where("contact_id = ?", contact.ID).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if err := syncer.db.Migrator().DropColumn(&models.ContactSubscriptionState{}, "rejected_etag"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := database.AutoMigrate(syncer.db); err != nil {
			t.Fatal(err)
		}
	}
	var after models.ContactSubscriptionState
	if err := syncer.db.Select(mappingColumns).Where("contact_id = ?", contact.ID).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("migration altered existing mapping: before=%+v after=%+v", before, after)
	}
}

// A push can finish after pull has read its mapping but before the rejected
// version is recorded. The acknowledgement must update only its own column.
func TestRejectedSubscriptionRevisionPreservesConcurrentPush(t *testing.T) {
	f := setupJobSubscription(t)
	if f.push.db.Dialector.Name() != "postgres" {
		t.Skip("requires independent PostgreSQL connections; repeated-pull coverage runs on both databases")
	}
	id := f.contact(t, "Local employee")
	company := f.company(t, "Employer")
	job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Deleted position"})
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"Employer", "Deleted position"}})
	if err := f.push.db.Model(&models.AddressBookSubscription{}).Where("id = ?", f.subscriptionID).Update("sync_way", SyncWayBoth).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.push.clientService.UpdateSyncStatus(f.subscriptionID, nil); err != nil {
		t.Fatal(err)
	}
	stale, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.jobs.Delete(id, f.vaultID, job.ID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"", ""}})
	var original, newer models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ?", id).First(&original).Error; err != nil {
		t.Fatal(err)
	}
	pushed := false
	const callback = "push_before_rejected_revision_acknowledgement"
	if err := f.push.db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "contact_subscription_states" || pushed {
			return
		}
		pushed = true
		f.push.PushContactChange(id, f.vaultID)
		f.expect(t, map[string][2]string{id: {"", ""}})
		if err := f.push.db.Where("contact_id = ?", id).First(&newer).Error; err != nil {
			t.Fatal(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.push.db.Callback().Create().Remove(callback) })
	syncer := NewDavSyncService(f.push.db, f.push.clientService, f.push.vcardService)
	syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
		queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			return []carddav.AddressObject{{Path: original.DistantURI, ETag: "rejected-before-push", Card: stale}}, nil
		},
	}})
	for round := 0; round < 2; round++ {
		result, err := syncer.SyncSubscription(context.Background(), f.subscriptionID, f.vaultID)
		if err != nil || result.Errors != 0 || result.Skipped != 1 || result.Updated != 0 {
			t.Fatalf("conflict pull: %+v %v", result, err)
		}
		var actual models.ContactSubscriptionState
		if err := f.push.db.Where("contact_id = ?", id).First(&actual).Error; err != nil {
			t.Fatal(err)
		}
		if !pushed || newer.DistantEtag == original.DistantEtag {
			t.Fatal("concurrent push did not advance mapping")
		}
		expected := newer
		expected.RejectedEtag = "rejected-before-push"
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("rejected pull overwrote concurrent push: want=%+v got=%+v", expected, actual)
		}
		card, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
		if err != nil {
			t.Fatal(err)
		}
		assertVCardJob(t, card, "", "")
	}
	// A later successful push must also retain the rejected revision.
	f.push.PushContactChange(id, f.vaultID)
	f.expect(t, map[string][2]string{id: {"", ""}})
	result, err := syncer.SyncSubscription(context.Background(), f.subscriptionID, f.vaultID)
	if err != nil || result.Errors != 0 || result.Skipped != 1 || result.Updated != 0 {
		t.Fatalf("push erased rejected revision: %+v %v", result, err)
	}
}
