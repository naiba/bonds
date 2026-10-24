package services

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Historical LegacyUpdate accepted a company from another vault. Real company
// FKs allow these references, so upgrades must coordinate their cleanup too.
func createHistoricalCompanyReference(t *testing.T, db *gorm.DB, contactID, vaultID, userID string, companyID uint) uint {
	t.Helper()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		_, err := NewContactJobService(tx).legacyUpdate(contactID, vaultID, userID, dto.UpdateJobInfoRequest{CompanyID: &companyID, JobPosition: "Historical role"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var job models.ContactCompany
	if err := db.Where("contact_id = ? AND company_id = ?", contactID, companyID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	return job.ID
}

func setupHistoricalCompanyReference(t *testing.T) (*gorm.DB, string, string, string, string, uint, uint) {
	t.Helper()
	svc, vault, user, account := setupContactTest(t)
	installCompanyForeignKeys(t, svc.db)
	other, err := NewVaultService(svc.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Historical employee vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	company, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Historical employer"})
	if err != nil {
		t.Fatal(err)
	}
	contact, err := svc.CreateContact(other.ID, user, dto.CreateContactRequest{FirstName: "Preserved person", LastName: "Unrelated surname"})
	if err != nil {
		t.Fatal(err)
	}
	jobID := createHistoricalCompanyReference(t, svc.db, contact.ID, other.ID, user, company.ID)
	return svc.db, vault, other.ID, user, contact.ID, company.ID, jobID
}

func waitForHistoricalContactLock(t *testing.T, db *gorm.DB, done <-chan error) (bool, error) {
	t.Helper()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			return true, err
		case <-timer.C:
			t.Fatal("company deletion neither completed nor waited on contact")
			return false, nil
		case <-tick.C:
			var count int64
			if err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count > 0 {
				return false, nil
			}
		}
	}
}

func TestCompanyDeleteSerializesHistoricalJobRemoval(t *testing.T) {
	for _, entry := range []string{"legacy_delete", "job_delete"} {
		t.Run(entry, func(t *testing.T) {
			db, vault, other, user, id, companyID, jobID := setupHistoricalCompanyReference(t)
			if db.Dialector.Name() != "postgres" {
				t.Skip("PostgreSQL deterministic row-lock cycle; serial cleanup runs on both engines")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			type operationKey struct{}
			companyDB := db.WithContext(ctx)
			jobDB := db.WithContext(context.WithValue(ctx, operationKey{}, true))
			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			jobDone, companyDone := make(chan error, 1), make(chan error, 1)
			jobFinished, companyFinished, companyStarted := false, false, false
			const callback = "hold_historical_job_removal_after_contact_lock"
			if err := db.Callback().Delete().Before("gorm:delete").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == "contact_companies" && tx.Statement.Context.Value(operationKey{}) == true {
					close(entered)
					select {
					case <-resume:
					case <-ctx.Done():
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer func() {
				release()
				if !jobFinished {
					<-jobDone
				}
				if companyStarted && !companyFinished {
					<-companyDone
				}
				db.Callback().Delete().Remove(callback)
			}()
			go func() {
				jobs := NewContactJobService(jobDB)
				if entry == "legacy_delete" {
					_, err := jobs.LegacyDelete(id, other, user)
					jobDone <- err
				} else {
					jobDone <- jobs.Delete(id, other, jobID)
				}
			}()
			select {
			case <-entered:
			case err := <-jobDone:
				jobFinished = true
				t.Fatalf("job deletion missed boundary: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			companyStarted = true
			go func() { companyDone <- NewCompanyService(companyDB).Delete(companyID, vault) }()
			early, companyErr := waitForHistoricalContactLock(t, db, companyDone)
			companyFinished = early
			release()
			jobErr := <-jobDone
			jobFinished = true
			if !companyFinished {
				companyErr = <-companyDone
				companyFinished = true
			}
			if early || jobErr != nil || companyErr != nil {
				t.Fatalf("deletions must serialize: early=%t job=%v company=%v", early, jobErr, companyErr)
			}
			var stored models.Contact
			if err := db.First(&stored, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			if stored.CompanyID != nil || ptrToStr(stored.FirstName) != "Preserved person" || ptrToStr(stored.LastName) != "Unrelated surname" {
				t.Fatal("cleanup retained company or changed unrelated fields")
			}
			if entry == "legacy_delete" && stored.JobPosition != nil {
				t.Fatal("legacy deletion did not clear role")
			}
			if entry == "job_delete" && ptrToStr(stored.JobPosition) != "Historical role" {
				t.Fatal("job deletion changed unrelated legacy role")
			}
			var jobs, companies int64
			if err := db.Model(&models.ContactCompany{}).Where("company_id = ?", companyID).Count(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.Company{}).Where("id = ?", companyID).Count(&companies).Error; err != nil {
				t.Fatal(err)
			}
			if jobs != 0 || companies != 0 {
				t.Fatal("company/job deletion incomplete")
			}
		})
	}
}

func TestCompanyDeleteLocksAllHistoricalReferencesBeforeJobs(t *testing.T) {
	for _, kind := range []string{"both", "legacy_only", "job_only"} {
		for _, tombstone := range []bool{false, true} {
			name := kind + "/live"
			if tombstone {
				name = kind + "/tombstone"
			}
			t.Run(name, func(t *testing.T) {
				db, vault, _, _, id, companyID, _ := setupHistoricalCompanyReference(t)
				if db.Dialector.Name() != "postgres" {
					t.Skip("PostgreSQL row locks; cross-vault/tombstone cleanup runs on both engines")
				}
				if kind == "legacy_only" {
					if err := db.Where("contact_id = ?", id).Delete(&models.ContactCompany{}).Error; err != nil {
						t.Fatal(err)
					}
				}
				if kind == "job_only" {
					if err := db.Model(&models.Contact{}).Where("id = ?", id).Update("company_id", nil).Error; err != nil {
						t.Fatal(err)
					}
				}
				if tombstone {
					if err := db.Where("id = ?", id).Delete(&models.Contact{}).Error; err != nil {
						t.Fatal(err)
					}
				}
				var before models.Contact
				if err := db.Unscoped().First(&before, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				entered, resume := make(chan struct{}), make(chan struct{})
				var once sync.Once
				release := func() { once.Do(func() { close(resume) }) }
				held, deleted := make(chan error, 1), make(chan error, 1)
				holderFinished, deleteFinished, deleteStarted := false, false, false
				var jobsTouched atomic.Bool
				const callback = "observe_historical_company_job_cleanup"
				if err := db.Callback().Delete().Before("gorm:delete").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Table == "contact_companies" {
						jobsTouched.Store(true)
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer func() {
					release()
					if !holderFinished {
						<-held
					}
					if deleteStarted && !deleteFinished {
						<-deleted
					}
					db.Callback().Delete().Remove(callback)
				}()
				go func() {
					held <- db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
						var c models.Contact
						if err := tx.Unscoped().Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).First(&c, "id = ?", id).Error; err != nil {
							return err
						}
						close(entered)
						select {
						case <-resume:
							return nil
						case <-ctx.Done():
							return ctx.Err()
						}
					})
				}()
				select {
				case <-entered:
				case err := <-held:
					holderFinished = true
					t.Fatalf("holder failed: %v", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				deleteStarted = true
				go func() { deleted <- NewCompanyService(db.WithContext(ctx)).Delete(companyID, vault) }()
				early, deleteErr := waitForHistoricalContactLock(t, db, deleted)
				deleteFinished = early
				touchedBeforeLock := jobsTouched.Load()
				release()
				holdErr := <-held
				holderFinished = true
				if !deleteFinished {
					deleteErr = <-deleted
					deleteFinished = true
				}
				if early || touchedBeforeLock || holdErr != nil || deleteErr != nil {
					t.Fatalf("cleanup bypassed owner lock: early=%t jobs_touched=%t holder=%v delete=%v", early, touchedBeforeLock, holdErr, deleteErr)
				}
				var after models.Contact
				if err := db.Unscoped().First(&after, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				expected := before
				expected.CompanyID = nil
				expected.UpdatedAt = after.UpdatedAt
				if !reflect.DeepEqual(expected, after) {
					t.Fatal("historical cleanup changed fields other than company reference/timestamp")
				}
				var count int64
				if err := db.Model(&models.ContactCompany{}).Where("company_id = ?", companyID).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("remaining jobs=%d err=%v", count, err)
				}
			})
		}
	}
}

func TestCompanyHistoricalCleanupRollbackAndPushScope(t *testing.T) {
	f := setupJobSubscription(t)
	installCompanyForeignKeys(t, f.push.db)
	local := f.contact(t, "Local employee")
	companyID := f.company(t, "Historical employer")
	if _, err := f.jobs.Create(local, f.vaultID, dto.CreateContactJobRequest{CompanyID: companyID, JobPosition: "Local role"}); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{local: {"Historical employer", "Local role"}})
	other, err := NewVaultService(f.push.db).CreateVault(f.accountID, f.userID, dto.CreateVaultRequest{Name: "Other subscriber vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	otherSub := createPushSubscription(t, f.push.clientService, other.ID, f.userID, SyncWayPush)
	foreign := []string{}
	for _, kind := range []string{"both", "legacy_only", "job_only"} {
		for _, tombstone := range []bool{false, true} {
			c, err := NewContactService(f.push.db).CreateContact(other.ID, f.userID, dto.CreateContactRequest{FirstName: kind, LastName: "Preserved"})
			if err != nil {
				t.Fatal(err)
			}
			createHistoricalCompanyReference(t, f.push.db, c.ID, other.ID, f.userID, companyID)
			if kind == "legacy_only" {
				if err := f.push.db.Where("contact_id = ?", c.ID).Delete(&models.ContactCompany{}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if kind == "job_only" {
				if err := f.push.db.Model(&models.Contact{}).Where("id = ?", c.ID).Update("company_id", nil).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := upsertContactSubscriptionState(f.push.db, c.ID, otherSub.ID, "/private/"+c.ID+".vcf", "preserved-version"); err != nil {
				t.Fatal(err)
			}
			if tombstone {
				if err := f.push.db.Where("id = ?", c.ID).Delete(&models.Contact{}).Error; err != nil {
					t.Fatal(err)
				}
			}
			foreign = append(foreign, c.ID)
		}
	}
	// A company/job unrelated to the deleted employer must survive byte-for-byte.
	unrelated, err := NewCompanyService(f.push.db).Create(other.ID, dto.CreateCompanyRequest{Name: "Unrelated employer"})
	if err != nil {
		t.Fatal(err)
	}
	preserved, err := NewContactService(f.push.db).CreateContact(other.ID, f.userID, dto.CreateContactRequest{FirstName: "Unrelated employee"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContactJobService(f.push.db).Create(preserved.ID, other.ID, dto.CreateContactJobRequest{CompanyID: unrelated.ID, JobPosition: "Unchanged role"}); err != nil {
		t.Fatal(err)
	}
	type snapshot struct {
		Contacts  []models.Contact
		Jobs      []models.ContactCompany
		Companies []models.Company
		States    []models.ContactSubscriptionState
	}
	load := func() snapshot {
		t.Helper()
		var s snapshot
		for _, q := range []struct {
			dest  any
			order string
		}{{&s.Contacts, "id"}, {&s.Jobs, "id"}, {&s.Companies, "id"}, {&s.States, "id"}} {
			if err := f.push.db.Unscoped().Order(q.order).Find(q.dest).Error; err != nil {
				t.Fatal(err)
			}
		}
		return s
	}
	before := load()
	if err := f.companies.Delete(companyID, other.ID); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("foreign-vault delete: %v", err)
	}
	if !reflect.DeepEqual(before, load()) {
		t.Fatal("rejected deletion changed data")
	}
	f.expectNoPush(t)
	rejected := errors.New("synthetic late company deletion failure")
	const callback = "reject_historical_company_delete"
	if err := f.push.db.Callback().Delete().After("gorm:delete").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "companies" {
			tx.AddError(rejected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	err = f.companies.Delete(companyID, f.vaultID)
	f.push.db.Callback().Delete().Remove(callback)
	if !errors.Is(err, rejected) || !reflect.DeepEqual(before, load()) {
		t.Fatalf("failed company deletion did not fully roll back: %v", err)
	}
	f.expectNoPush(t)
	if err := f.companies.Delete(companyID, f.vaultID); err != nil {
		t.Fatal(err)
	}
	f.expect(t, map[string][2]string{local: {"", ""}})
	f.expectNoPush(t)
	after := load()
	if len(after.Companies) != 1 || len(after.Jobs) != 1 || after.Companies[0].ID != unrelated.ID || after.Jobs[0].ContactID != preserved.ID {
		t.Fatal("company cleanup left references")
	}
	for _, c := range before.Companies {
		if c.ID == unrelated.ID && !reflect.DeepEqual(c, after.Companies[0]) {
			t.Fatal("unrelated company changed")
		}
	}
	for _, j := range before.Jobs {
		if j.ContactID == preserved.ID && !reflect.DeepEqual(j, after.Jobs[0]) {
			t.Fatal("unrelated job changed")
		}
	}
	var originalPreserved, currentPreserved models.Contact
	for _, c := range before.Contacts {
		if c.ID == preserved.ID {
			originalPreserved = c
		}
	}
	for _, c := range after.Contacts {
		if c.ID == preserved.ID {
			currentPreserved = c
		}
	}
	if !reflect.DeepEqual(originalPreserved, currentPreserved) {
		t.Fatal("unrelated contact changed")
	}
	for _, id := range foreign {
		var old, current models.Contact
		for _, c := range before.Contacts {
			if c.ID == id {
				old = c
			}
		}
		for _, c := range after.Contacts {
			if c.ID == id {
				current = c
			}
		}
		expected := old
		expected.CompanyID = nil
		expected.UpdatedAt = current.UpdatedAt
		if !reflect.DeepEqual(expected, current) {
			t.Fatal("foreign contact profile, edit marker or tombstone changed")
		}
		var oldState, currentState models.ContactSubscriptionState
		for _, s := range before.States {
			if s.ContactID == id {
				oldState = s
			}
		}
		for _, s := range after.States {
			if s.ContactID == id {
				currentState = s
			}
		}
		if !reflect.DeepEqual(oldState, currentState) {
			t.Fatal("foreign subscriber mapping changed")
		}
	}
}
