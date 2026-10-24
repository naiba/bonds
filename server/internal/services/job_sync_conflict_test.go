package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

// A remote response captured before a local job edit can arrive after its push
// has completed. Exercise the real importer, not just the field reconciler.
func TestJobChangesRejectDelayedSubscriptionPull(t *testing.T) {
	for _, entry := range []string{"delete_job", "remove_employee", "delete_company", "clear_legacy", "update_job", "rename_company", "add_job", "add_employee", "update_legacy"} {
		for _, firstSync := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/first_sync_%t", entry, firstSync), func(t *testing.T) {
				f := setupJobSubscription(t)
				id := f.contact(t, "Local employee")
				company := f.company(t, "Current company")
				job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Previous position"})
				if err != nil {
					t.Fatal(err)
				}
				f.expect(t, map[string][2]string{id: {"Current company", "Previous position"}})
				if err := f.push.db.Model(&models.AddressBookSubscription{}).Where("id = ?", f.subscriptionID).Update("sync_way", SyncWayBoth).Error; err != nil {
					t.Fatal(err)
				}
				if !firstSync {
					if err := f.push.clientService.UpdateSyncStatus(f.subscriptionID, nil); err != nil {
						t.Fatal(err)
					}
				}
				stale, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
				if err != nil {
					t.Fatal(err)
				}
				stale.SetName(&vcard.Name{GivenName: "Remote stale name"})
				stale.SetValue(vcard.FieldFormattedName, "Remote stale name")
				var stateBefore, stateAfter models.ContactSubscriptionState
				var contactBefore models.Contact
				var jobsBefore []models.ContactCompany
				wantOrg, wantTitle := "", ""
				syncer := NewDavSyncService(f.push.db, f.push.clientService, f.push.vcardService)
				syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
					syncCollFn: func(context.Context, string, *carddav.SyncQuery) (*carddav.SyncResponse, error) {
						return &carddav.SyncResponse{}, nil
					},
					queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
						switch entry {
						case "delete_job":
							err = f.jobs.Delete(id, f.vaultID, job.ID)
						case "remove_employee":
							err = f.jobs.RemoveEmployee(company, f.vaultID, id)
						case "delete_company":
							err = f.companies.Delete(company, f.vaultID)
						case "clear_legacy":
							_, err = f.jobs.LegacyDelete(id, f.vaultID, f.userID)
						case "update_job":
							wantOrg, wantTitle = "Current company", "New position"
							_, err = f.jobs.Update(id, f.vaultID, job.ID, dto.UpdateContactJobRequest{CompanyID: company, JobPosition: wantTitle})
						case "rename_company":
							wantOrg, wantTitle = "Renamed company", "Previous position"
							_, err = f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: wantOrg})
						case "add_job":
							wantOrg, wantTitle = "Current company", "New position"
							_, err = f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: wantTitle})
						case "add_employee":
							wantOrg, wantTitle = "Current company", "New position"
							_, err = f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: wantTitle})
						case "update_legacy":
							wantOrg, wantTitle = "Current company", "New position"
							_, err = f.jobs.LegacyUpdate(id, f.vaultID, f.userID, dto.UpdateJobInfoRequest{CompanyID: &company, JobPosition: wantTitle})
						}
						if err != nil {
							t.Fatal(err)
						}
						f.expect(t, map[string][2]string{id: {wantOrg, wantTitle}})
						if err := f.push.db.First(&contactBefore, "id = ?", id).Error; err != nil {
							t.Fatal(err)
						}
						if err := f.push.db.Where("contact_id = ?", id).Order("id").Find(&jobsBefore).Error; err != nil {
							t.Fatal(err)
						}
						if err := f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", id, f.subscriptionID).First(&stateBefore).Error; err != nil {
							t.Fatal(err)
						}
						return []carddav.AddressObject{{Path: stateBefore.DistantURI, ETag: "delayed-remote-revision", Card: stale}}, nil
					},
				}})
				result, err := syncer.SyncSubscription(context.Background(), f.subscriptionID, f.vaultID)
				if err != nil || result == nil || result.Errors != 0 || result.Created != 0 || result.Updated != 0 || result.Skipped != 1 {
					t.Fatalf("stale pull must keep local change: %+v %v", result, err)
				}
				var stored models.Contact
				var jobsAfter []models.ContactCompany
				if err := f.push.db.First(&stored, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := f.push.db.Where("contact_id = ?", id).Order("id").Find(&jobsAfter).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(contactBefore, stored) || !reflect.DeepEqual(jobsBefore, jobsAfter) {
					t.Fatal("conflicting pull modified the local contact or jobs")
				}
				if err := f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", id, f.subscriptionID).First(&stateAfter).Error; err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stateBefore, stateAfter) {
					t.Fatalf("conflicting pull replaced successful push state: before=%+v after=%+v", stateBefore, stateAfter)
				}
				card, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
				if err != nil {
					t.Fatal(err)
				}
				assertVCardJob(t, card, wantOrg, wantTitle)
				var count int64
				if err := f.push.db.Model(&models.DavSyncLog{}).Where("address_book_subscription_id = ? AND contact_id = ? AND action = ?", f.subscriptionID, id, "conflict_local_wins").Count(&count).Error; err != nil || count != 1 {
					t.Fatalf("conflict log count=%d err=%v", count, err)
				}
				f.expectNoPush(t)
			})
		}
	}
}

func TestSubscriptionPullPreservesLegacyTitleUpdates(t *testing.T) {
	for _, firstSync := range []bool{false, true} {
		t.Run(fmt.Sprintf("first_sync_%t", firstSync), func(t *testing.T) {
			syncer, clients, cards, vault, user, _ := setupDavSyncTest(t)
			sub := createPushSubscription(t, clients, vault, user, SyncWayBoth)
			oldEdit := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
			contact := models.Contact{VaultID: vault, FirstName: strPtrOrNil("Legacy"), JobPosition: strPtrOrNil("Old title"), LastUpdatedAt: &oldEdit}
			if err := syncer.db.Create(&contact).Error; err != nil {
				t.Fatal(err)
			}
			path := "/contacts/legacy.vcf"
			if err := upsertContactSubscriptionState(syncer.db, contact.ID, sub.ID, path, "initial"); err != nil {
				t.Fatal(err)
			}
			if !firstSync {
				if err := clients.UpdateSyncStatus(sub.ID, nil); err != nil {
					t.Fatal(err)
				}
			}
			for _, title := range []string{"Remote title", ""} {
				card := makeVCard("Remote name", "", contact.ID)
				if title != "" {
					card.SetValue(vcard.FieldTitle, title)
				}
				etag := "remote-" + title
				syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
					syncCollFn: func(context.Context, string, *carddav.SyncQuery) (*carddav.SyncResponse, error) {
						return &carddav.SyncResponse{}, nil
					},
					queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
						return []carddav.AddressObject{{Path: path, ETag: etag, Card: card}}, nil
					},
				}})
				result, err := syncer.SyncSubscription(context.Background(), sub.ID, vault)
				if err != nil || result.Errors != 0 || result.Updated != 1 || result.Skipped != 0 {
					t.Fatalf("legacy pull: %+v %v", result, err)
				}
				var stored models.Contact
				if err := syncer.db.First(&stored, "id = ?", contact.ID).Error; err != nil {
					t.Fatal(err)
				}
				if ptrToStr(stored.JobPosition) != title || ptrToStr(stored.FirstName) != "Remote name" || stored.LastUpdatedAt == nil || !stored.LastUpdatedAt.Equal(oldEdit) {
					t.Fatal("legacy title/name not accepted or pull marked as local change")
				}
				exported, err := cards.ExportContactToVCard(contact.ID, vault)
				if err != nil {
					t.Fatal(err)
				}
				assertVCardJob(t, exported, "", title)
				var state models.ContactSubscriptionState
				if err := syncer.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
					t.Fatal(err)
				}
				if state.DistantEtag != etag || state.DistantURI != path {
					t.Fatal("accepted remote revision not recorded")
				}
			}
		})
	}
}

func TestJobPullChecksConflictAfterWaitingForDeletion(t *testing.T) {
	f := setupJobSubscription(t)
	if f.push.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row-lock ordering; delayed-response coverage runs on both databases")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	f.jobs.db = f.jobs.db.WithContext(ctx)
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
	stale.SetName(&vcard.Name{GivenName: "Stale remote name"})
	stale.SetValue(vcard.FieldFormattedName, "Stale remote name")
	var state models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ?", id).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	entered, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	deleteDone := make(chan error, 1)
	pullDone := make(chan error, 1)
	deleteFinished, pullStarted, pullFinished := false, false, false
	const callback = "hold_job_deletion_before_commit"
	if err := f.push.db.Callback().Delete().Before("gorm:delete").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_companies" {
			close(entered)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		release()
		if !deleteFinished {
			<-deleteDone
		}
		if pullStarted && !pullFinished {
			<-pullDone
		}
		f.push.db.Callback().Delete().Remove(callback)
	}()
	go func() { deleteDone <- f.jobs.Delete(id, f.vaultID, job.ID) }()
	select {
	case <-entered:
	case err := <-deleteDone:
		deleteFinished = true
		t.Fatalf("delete missed boundary: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	syncer := NewDavSyncService(f.push.db.WithContext(ctx), f.push.clientService, f.push.vcardService)
	syncer.SetClientFactory(&mockCardDAVClientFactory{client: &mockCardDAVClient{
		syncCollFn: func(context.Context, string, *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return &carddav.SyncResponse{}, nil
		},
		queryFn: func(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			return []carddav.AddressObject{{Path: state.DistantURI, ETag: "stale-while-delete-locks", Card: stale}}, nil
		},
	}})
	pullStarted = true
	go func() {
		result, err := syncer.SyncSubscription(ctx, f.subscriptionID, f.vaultID)
		if err == nil && (result.Errors != 0 || result.Updated != 0 || result.Skipped != 1) {
			err = fmt.Errorf("expected local conflict after lock wait: %+v", result)
		}
		pullDone <- err
	}()
	// Observe a real wait on this schema's contact, not a scheduling sleep.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	waiting := false
	for !waiting {
		select {
		case err := <-pullDone:
			pullFinished = true
			t.Fatalf("pull did not wait for contact: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
			var count int64
			err := f.push.db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error
			if err != nil {
				t.Fatal(err)
			}
			waiting = count > 0
		}
	}
	release()
	err = <-deleteDone
	deleteFinished = true
	if err != nil {
		t.Fatal(err)
	}
	err = <-pullDone
	pullFinished = true
	if err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"", ""}})
	var stored models.Contact
	if err := f.push.db.First(&stored, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if ptrToStr(stored.FirstName) != "Local employee" || stored.JobPosition != nil {
		t.Fatal("stale pre-lock snapshot overwrote local contact")
	}
	var count int64
	if err := f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("deleted job reappeared: %d %v", count, err)
	}
	card, err := f.push.vcardService.ExportContactToVCard(id, f.vaultID)
	if err != nil {
		t.Fatal(err)
	}
	assertVCardJob(t, card, "", "")
}

func TestJobSyncMarkerFailureRollsBackMutation(t *testing.T) {
	for _, entry := range []string{"delete_job", "remove_employee", "delete_company", "clear_legacy", "rename_company"} {
		t.Run(entry, func(t *testing.T) {
			f := setupJobSubscription(t)
			id := f.contact(t, "Preserved employee")
			companyID := f.company(t, "Preserved company")
			job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: companyID, JobPosition: "Preserved position"})
			if err != nil {
				t.Fatal(err)
			}
			f.expect(t, map[string][2]string{id: {"Preserved company", "Preserved position"}})
			var before, after models.Contact
			var companyBefore, companyAfter models.Company
			var stateBefore, stateAfter models.ContactSubscriptionState
			var jobsBefore, jobsAfter []models.ContactCompany
			load := func(c *models.Contact, co *models.Company, st *models.ContactSubscriptionState, jobs *[]models.ContactCompany) {
				t.Helper()
				if err := f.push.db.First(c, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if err := f.push.db.First(co, companyID).Error; err != nil {
					t.Fatal(err)
				}
				if err := f.push.db.Where("contact_id = ?", id).First(st).Error; err != nil {
					t.Fatal(err)
				}
				if err := f.push.db.Where("contact_id = ?", id).Order("id").Find(jobs).Error; err != nil {
					t.Fatal(err)
				}
			}
			load(&before, &companyBefore, &stateBefore, &jobsBefore)
			rejected := errors.New("synthetic local job marker failure")
			const callback = "reject_local_job_marker"
			if err := f.push.db.Callback().Update().After("gorm:update").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == "contacts" && strings.Contains(tx.Statement.SQL.String(), "last_updated_at") {
					tx.AddError(rejected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.push.db.Callback().Update().Remove(callback)
			switch entry {
			case "delete_job":
				err = f.jobs.Delete(id, f.vaultID, job.ID)
			case "remove_employee":
				err = f.jobs.RemoveEmployee(companyID, f.vaultID, id)
			case "delete_company":
				err = f.companies.Delete(companyID, f.vaultID)
			case "clear_legacy":
				_, err = f.jobs.LegacyDelete(id, f.vaultID, f.userID)
			case "rename_company":
				_, err = f.companies.Update(companyID, f.vaultID, dto.UpdateCompanyRequest{Name: "Rejected name"})
			}
			if !errors.Is(err, rejected) {
				t.Fatalf("expected marker error: %v", err)
			}
			load(&after, &companyAfter, &stateAfter, &jobsAfter)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(companyBefore, companyAfter) || !reflect.DeepEqual(jobsBefore, jobsAfter) || !reflect.DeepEqual(stateBefore, stateAfter) {
				t.Fatal("marker failure left partial contact/job/company/subscriber changes")
			}
			f.expectNoPush(t)
		})
	}
}

func TestCompanyTypeDoesNotMarkContactAsLocallyEdited(t *testing.T) {
	f := setupJobSubscription(t)
	id := f.contact(t, "Employee")
	company := f.company(t, "Employer")
	if _, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Role"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{id: {"Employer", "Role"}})
	var before, after models.Contact
	if err := f.push.db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "Employer", Type: "New type"}); err != nil {
		t.Fatal(err)
	}
	if err := f.push.db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("company type-only edit changed contact or local conflict marker")
	}
	f.expectNoPush(t)
}
