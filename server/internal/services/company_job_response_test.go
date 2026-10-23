package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// Successful mutation responses describe the write that committed, even if a
// later company deletion has already removed that job by the time we return.
func TestJobResponseSerializesWithCompanyDeletion(t *testing.T) {
	for _, entry := range []string{"employee", "create", "update"} {
		t.Run(entry, func(t *testing.T) {
			f := setupJobSubscription(t)
			if f.push.db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL row-lock interleaving; response rollback runs on both databases")
			}
			installCompanyForeignKeys(t, f.push.db)
			id := f.contact(t, "Response employee")
			company := f.company(t, "Response company")
			var jobID uint
			if entry == "update" {
				job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Previous role"})
				if err != nil {
					t.Fatal(err)
				}
				jobID = job.ID
				f.expect(t, map[string][2]string{id: {"Response company", "Previous role"}})
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			f.jobs.db = f.jobs.db.WithContext(ctx)
			f.companies.db = f.companies.db.WithContext(ctx)
			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			type result struct {
				job *dto.ContactJobResponse
				err error
			}
			jobDone, deleteDone := make(chan result, 1), make(chan error, 1)
			var jobFinished, deleteFinished, deleteStarted bool
			const callback = "pause_job_response_before_read"
			if err := f.push.db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
				if row, ok := tx.Statement.Dest.(*models.ContactCompany); ok && row.ID != 0 && row.ContactID == id {
					close(entered)
					<-resume
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer func() {
				release()
				if !jobFinished {
					<-jobDone
				}
				if deleteStarted && !deleteFinished {
					<-deleteDone
				}
				f.push.db.Callback().Query().Remove(callback)
			}()
			go func() {
				var job *dto.ContactJobResponse
				var err error
				switch entry {
				case "employee":
					job, err = f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Engineer"})
				case "create":
					job, err = f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
				case "update":
					job, err = f.jobs.Update(id, f.vaultID, jobID, dto.UpdateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
				}
				jobDone <- result{job, err}
			}()
			select {
			case <-entered:
			case got := <-jobDone:
				jobFinished = true
				t.Fatalf("response boundary not reached: %v", got.err)
			case <-time.After(10 * time.Second):
				t.Fatal("response boundary not reached")
			}
			deleteStarted = true
			go func() { deleteDone <- f.companies.Delete(company, f.vaultID) }()
			early, deleteErr := waitForCompanyLockOrCompletion(t, f.push.db, deleteDone)
			deleteFinished = early
			release()
			got := <-jobDone
			jobFinished = true
			if !early {
				deleteErr = <-deleteDone
				deleteFinished = true
			}
			if deleteErr != nil {
				t.Fatal(deleteErr)
			}
			// Drain both committed operations before checking the response so a
			// regression reports its primary failure without racing DB teardown.
			assertFinalJobNotifications(t, f, id, 2, "", "")
			if got.err != nil {
				t.Fatalf("committed mutation must return its response: %v", got.err)
			}
			if got.job == nil || got.job.ID == 0 || got.job.ContactID != id || got.job.CompanyID != company || got.job.CompanyName != "Response company" || got.job.JobPosition != "Engineer" {
				t.Fatalf("incorrect committed response: %+v", got.job)
			}
			var count int64
			if err := f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("deleted company retained jobs: %d %v", count, err)
			}
			var contact models.Contact
			if err := f.push.db.First(&contact, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(contact.FirstName) != "Response employee" {
				t.Fatal("unrelated name changed")
			}
		})
	}
}

func assertFinalJobNotifications(t *testing.T, f *jobSubscriptionFixture, id string, count int, organization, title string) {
	t.Helper()
	var last jobSubscriptionPush
	var received []jobSubscriptionPush
	for range count {
		select {
		case last = <-f.received:
			if last.contactID != id {
				t.Fatalf("unexpected recipient: %+v", last)
			}
			received = append(received, last)
		case <-time.After(5 * time.Second):
			t.Fatal("committed job/company operation did not notify subscriber")
		}
	}
	unlock := f.push.operationLocks.lock(id)
	unlock()
	if last.organization != organization || last.title != title {
		t.Fatalf("final remote card incorrect: %+v", last)
	}
	var state models.ContactSubscriptionState
	if err := f.push.db.Where("contact_id = ? AND address_book_subscription_id = ?", id, f.subscriptionID).First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.DistantEtag != last.etag || state.DistantURI != last.path {
		t.Fatal("final notification state was not persisted")
	}
	for _, push := range received {
		var logs int64
		if err := f.push.db.Model(&models.DavSyncLog{}).Where("contact_id = ? AND address_book_subscription_id = ? AND action = ? AND distant_etag = ?", id, f.subscriptionID, "pushed", push.etag).Count(&logs).Error; err != nil || logs != 1 {
			t.Fatalf("push log missing: %d %v", logs, err)
		}
	}
	card, err := NewVCardService(f.push.db).ExportContactToVCard(id, f.vaultID)
	if err != nil {
		t.Fatal(err)
	}
	assertVCardJob(t, card, organization, title)
	f.expectNoPush(t)
}

func TestJobResponseReadFailureRollsBackMutation(t *testing.T) {
	for _, entry := range []string{"employee", "create", "update"} {
		t.Run(entry, func(t *testing.T) {
			f := setupJobSubscription(t)
			id := f.contact(t, "Preserved response name")
			company := f.company(t, "Preserved company")
			var jobID uint
			if entry == "update" {
				job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Preserved role"})
				if err != nil {
					t.Fatal(err)
				}
				jobID = job.ID
				f.expect(t, map[string][2]string{id: {"Preserved company", "Preserved role"}})
			}
			var before models.ContactSubscriptionState
			if err := f.push.db.Where("contact_id = ?", id).First(&before).Error; err != nil {
				t.Fatal(err)
			}
			injected := errors.New("synthetic job response read failure")
			const callback = "reject_job_response_read"
			if err := f.push.db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
				if row, ok := tx.Statement.Dest.(*models.ContactCompany); ok && row.ID != 0 && row.ContactID == id {
					tx.AddError(injected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer f.push.db.Callback().Query().Remove(callback)
			var err error
			switch entry {
			case "employee":
				_, err = f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Rejected role"})
			case "create":
				_, err = f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Rejected role"})
			case "update":
				_, err = f.jobs.Update(id, f.vaultID, jobID, dto.UpdateContactJobRequest{CompanyID: company, JobPosition: "Rejected role"})
			}
			if !errors.Is(err, injected) {
				t.Fatalf("expected response error, got %v", err)
			}
			var jobs []models.ContactCompany
			if err := f.push.db.Where("contact_id = ?", id).Find(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if jobID == 0 && len(jobs) != 0 {
				f.expect(t, map[string][2]string{id: {"Preserved company", "Rejected role"}})
				t.Fatal("failed response left a committed new job")
			}
			if jobID != 0 && (len(jobs) != 1 || jobs[0].ID != jobID || ptrToStr(jobs[0].JobPosition) != "Preserved role") {
				f.expect(t, map[string][2]string{id: {"Preserved company", "Rejected role"}})
				t.Fatal("failed response committed job edits")
			}
			f.expectNoPush(t)
			var after models.ContactSubscriptionState
			if err := f.push.db.Where("contact_id = ?", id).First(&after).Error; err != nil {
				t.Fatal(err)
			}
			if before.DistantEtag != after.DistantEtag || before.DistantURI != after.DistantURI {
				t.Fatal("rolled back response changed subscriber state")
			}
			var contact models.Contact
			if err := f.push.db.First(&contact, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(contact.FirstName) != "Preserved response name" || contact.CompanyID != nil || contact.JobPosition != nil {
				t.Fatal("rolled back response changed contact")
			}
		})
	}
}
