package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Observe a real database wait, rather than assuming that a sleeping goroutine
// has reached the competing write. Scope the relation to this test's schema.
func waitForCompanyLockOrCompletion(t *testing.T, db *gorm.DB, done <-chan error) (bool, error) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			return true, err
		case <-deadline.C:
			t.Fatal("company operation neither completed nor waited for the company lock")
			return false, nil
		case <-tick.C:
			var waiting int64
			err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid <> pg_backend_pid() AND a.wait_event_type = 'Lock' AND a.query LIKE '%companies%FOR %UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid = a.pid AND l.relation = 'companies'::regclass)`).Scan(&waiting).Error
			if err != nil {
				t.Fatal(err)
			}
			if waiting > 0 {
				return false, nil
			}
		}
	}
}

func TestCompanyMutationCoordinatesConcurrentJobAssociation(t *testing.T) {
	for _, operation := range []string{"rename", "delete"} {
		for _, entry := range []string{"employee", "create", "retarget", "legacy_create", "legacy_retarget"} {
			t.Run(operation+"/"+entry, func(t *testing.T) {
				f := setupJobSubscription(t)
				if f.push.db.Dialector.Name() != "postgres" {
					t.Skip("row-lock interleaving requires PostgreSQL; serial lifecycle runs on both databases")
				}
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				f.jobs.db = f.jobs.db.WithContext(ctx)
				f.companies.db = f.companies.db.WithContext(ctx)
				id := f.contact(t, "Concurrent employee")
				company := f.company(t, "Before rename")
				var jobID uint
				expectedOrg, expectedTitle := "", ""
				if entry == "retarget" || entry == "legacy_retarget" {
					previous := f.company(t, "Preserved company")
					job, err := f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: previous, JobPosition: "Preserved role"})
					if err != nil {
						t.Fatal(err)
					}
					jobID = job.ID
					expectedOrg, expectedTitle = "Preserved company", "Preserved role"
					f.expect(t, map[string][2]string{id: {expectedOrg, expectedTitle}})
				}
				var before models.Contact
				if err := f.push.db.First(&before, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				var beforeState models.ContactSubscriptionState
				if err := f.push.db.Where("contact_id = ?", id).First(&beforeState).Error; err != nil {
					t.Fatal(err)
				}
				entered, resume := make(chan struct{}), make(chan struct{})
				var once sync.Once
				release := func() { once.Do(func() { close(resume) }) }
				companyDone, jobDone := make(chan error, 1), make(chan error, 1)
				var companyFinished, jobFinished bool
				jobStarted := false
				callback := func(tx *gorm.DB) {
					if (operation == "rename" && tx.Statement.Table == "companies") || (operation == "delete" && tx.Statement.Table == "contact_companies") {
						close(entered)
						<-resume
					}
				}
				const callbackName = "pause_company_mutation_for_association"
				if operation == "rename" {
					if err := f.push.db.Callback().Update().Before("gorm:update").Register(callbackName, callback); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := f.push.db.Callback().Delete().Before("gorm:delete").Register(callbackName, callback); err != nil {
						t.Fatal(err)
					}
				}
				defer func() {
					release()
					if !companyFinished {
						<-companyDone
					}
					if jobStarted && !jobFinished {
						<-jobDone
					}
					if operation == "rename" {
						f.push.db.Callback().Update().Remove(callbackName)
					} else {
						f.push.db.Callback().Delete().Remove(callbackName)
					}
				}()
				go func() {
					if operation == "rename" {
						_, err := f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "After rename"})
						companyDone <- err
					} else {
						companyDone <- f.companies.Delete(company, f.vaultID)
					}
				}()
				select {
				case <-entered:
				case err := <-companyDone:
					companyFinished = true
					t.Fatalf("company mutation failed before write boundary: %v", err)
				case <-time.After(10 * time.Second):
					t.Fatal("company mutation did not reach write boundary")
				}
				jobStarted = true
				go func() {
					var err error
					switch entry {
					case "employee":
						_, err = f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Engineer"})
					case "create":
						_, err = f.jobs.Create(id, f.vaultID, dto.CreateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
					case "retarget":
						_, err = f.jobs.Update(id, f.vaultID, jobID, dto.UpdateContactJobRequest{CompanyID: company, JobPosition: "Engineer"})
					default:
						_, err = f.jobs.LegacyUpdate(id, f.vaultID, f.userID, dto.UpdateJobInfoRequest{CompanyID: &company, JobPosition: "Engineer"})
					}
					jobDone <- err
				}()
				early, jobErr := waitForCompanyLockOrCompletion(t, f.push.db, jobDone)
				jobFinished = early
				if early {
					if jobErr != nil {
						t.Fatal(jobErr)
					}
					f.expect(t, map[string][2]string{id: {"Before rename", "Engineer"}})
				}
				release()
				companyErr := <-companyDone
				companyFinished = true
				if companyErr != nil {
					t.Fatal(companyErr)
				}
				if !early {
					jobErr = <-jobDone
					jobFinished = true
				}
				if operation == "rename" {
					if jobErr != nil {
						t.Fatal(jobErr)
					}
					expectedOrg, expectedTitle = "After rename", "Engineer"
					f.expect(t, map[string][2]string{id: {expectedOrg, expectedTitle}})
				} else if early {
					expectedOrg, expectedTitle = "", ""
					f.expect(t, map[string][2]string{id: {expectedOrg, expectedTitle}})
				} else {
					if !errors.Is(jobErr, ErrCompanyNotFound) {
						t.Fatalf("deleted target must be rejected, got %v", jobErr)
					}
					f.expectNoPush(t)
					var afterState models.ContactSubscriptionState
					if err := f.push.db.Where("contact_id = ?", id).First(&afterState).Error; err != nil {
						t.Fatal(err)
					}
					if afterState.DistantEtag != beforeState.DistantEtag || afterState.DistantURI != beforeState.DistantURI {
						t.Fatal("rejected association changed subscription state")
					}
				}
				card, err := NewVCardService(f.push.db).ExportContactToVCard(id, f.vaultID)
				if err != nil {
					t.Fatal(err)
				}
				assertVCardJob(t, card, expectedOrg, expectedTitle)
				var after models.Contact
				if err := f.push.db.First(&after, "id = ?", id).Error; err != nil {
					t.Fatal(err)
				}
				if ptrToStr(after.FirstName) != ptrToStr(before.FirstName) {
					t.Fatal("unrelated name changed")
				}
				if operation == "delete" && !early {
					if after.CompanyID != nil || ptrToStr(after.JobPosition) != ptrToStr(before.JobPosition) {
						t.Fatal("rejected association changed legacy fields")
					}
					var jobs []models.ContactCompany
					if err := f.push.db.Where("contact_id = ?", id).Find(&jobs).Error; err != nil {
						t.Fatal(err)
					}
					expectedCount := 0
					if jobID != 0 {
						expectedCount = 1
					}
					if len(jobs) != expectedCount {
						t.Fatalf("rejected association changed job count: %d", len(jobs))
					}
					if jobID != 0 && (jobs[0].ID != jobID || ptrToStr(jobs[0].JobPosition) != "Preserved role") {
						t.Fatal("existing job lost on rejection")
					}
				}
			})
		}
	}
}

// If the association owns the company lock first, the company mutation must
// include its committed contact in the notification set. Intermediate pushes
// may already contain the new state; only their final state is prescribed.
func TestCompanyMutationIncludesAssociationCommittedFirst(t *testing.T) {
	for _, operation := range []string{"rename", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := setupJobSubscription(t)
			if f.push.db.Dialector.Name() != "postgres" {
				t.Skip("row-lock interleaving requires PostgreSQL")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			f.jobs.db = f.jobs.db.WithContext(ctx)
			f.companies.db = f.companies.db.WithContext(ctx)
			id := f.contact(t, "First association")
			company := f.company(t, "Before rename")
			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			const callbackName = "pause_association_before_insert"
			if err := f.push.db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
				if tx.Statement.Table == "contact_companies" {
					close(entered)
					<-resume
				}
			}); err != nil {
				t.Fatal(err)
			}
			jobDone, companyDone := make(chan error, 1), make(chan error, 1)
			var jobFinished, companyFinished, companyStarted bool
			defer func() {
				release()
				if !jobFinished {
					<-jobDone
				}
				if companyStarted && !companyFinished {
					<-companyDone
				}
				f.push.db.Callback().Create().Remove(callbackName)
			}()
			go func() {
				_, err := f.jobs.AddEmployee(company, f.vaultID, dto.AddEmployeeRequest{ContactID: id, JobPosition: "Engineer"})
				jobDone <- err
			}()
			select {
			case <-entered:
			case err := <-jobDone:
				jobFinished = true
				t.Fatalf("association failed before insertion: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("association never reached insertion")
			}
			companyStarted = true
			go func() {
				if operation == "rename" {
					_, err := f.companies.Update(company, f.vaultID, dto.UpdateCompanyRequest{Name: "After rename"})
					companyDone <- err
				} else {
					companyDone <- f.companies.Delete(company, f.vaultID)
				}
			}()
			early, err := waitForCompanyLockOrCompletion(t, f.push.db, companyDone)
			companyFinished = early
			if early {
				t.Fatalf("company mutation completed before uncommitted association: %v", err)
			}
			release()
			err = <-jobDone
			jobFinished = true
			if err != nil {
				t.Fatal(err)
			}
			err = <-companyDone
			companyFinished = true
			if err != nil {
				t.Fatal(err)
			}
			wantOrg, wantTitle := "After rename", "Engineer"
			if operation == "delete" {
				wantOrg, wantTitle = "", ""
			}
			var last jobSubscriptionPush
			for range 2 {
				select {
				case last = <-f.received:
					if last.contactID != id {
						t.Fatalf("unrelated recipient: %+v", last)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("both committed mutations must notify the association")
				}
			}
			unlock := f.push.operationLocks.lock(id)
			unlock()
			if last.organization != wantOrg || last.title != wantTitle {
				t.Fatalf("final remote projection: %+v", last)
			}
			var state models.ContactSubscriptionState
			if err := f.push.db.Where("contact_id = ?", id).First(&state).Error; err != nil {
				t.Fatal(err)
			}
			if state.DistantEtag != last.etag || state.DistantURI != last.path {
				t.Fatalf("final remote state was not persisted: %+v", state)
			}
			var count int64
			if err := f.push.db.Model(&models.DavSyncLog{}).Where("contact_id = ? AND action = ?", id, "pushed").Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 3 {
				t.Fatalf("expected initial contact push and two successful mutation pushes, got %d", count)
			}
			card, err := NewVCardService(f.push.db).ExportContactToVCard(id, f.vaultID)
			if err != nil {
				t.Fatal(err)
			}
			assertVCardJob(t, card, wantOrg, wantTitle)
			if operation == "delete" {
				if err := f.push.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("deleted company retained jobs: %d %v", count, err)
				}
			}
			f.expectNoPush(t)
		})
	}
}

// Import resolves the company before writing the contact and later its job.
// Deleting it at either boundary must not leave dangling company references.
func TestMonicaImportCompanyDeletionAtAssociationBoundary(t *testing.T) {
	for _, boundary := range []string{"contact", "job"} {
		t.Run(boundary, func(t *testing.T) {
			svc, vaultID, userID := setupMonicaCompanyAssociationBoundary(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			svc.DB = svc.DB.WithContext(ctx)
			company, err := NewCompanyService(svc.DB).Create(vaultID, dto.CreateCompanyRequest{Name: "Imported company"})
			if err != nil {
				t.Fatal(err)
			}
			lockQueries := 0
			fired := false
			const callbackName = "delete_import_company_before_association_lock"
			if err := svc.DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
				if fired || tx.Statement.Table != "companies" {
					return
				}
				if _, locked := tx.Statement.Clauses["FOR"]; !locked {
					return
				}
				lockQueries++
				target := 1
				if boundary == "job" {
					target = 2
				}
				if lockQueries != target {
					return
				}
				fired = true
				if err := NewCompanyService(svc.DB).Delete(company.ID, vaultID); err != nil {
					tx.AddError(err)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer svc.DB.Callback().Query().Remove(callbackName)
			payload := []byte(`{"version":"1.0-preview.1","account":{"data":[{"type":"contacts","values":[{"uuid":"imported-employee","properties":{"first_name":"Preserved imported name","company":"Imported company","job":"Engineer"},"data":[]}]}]}}`)
			result, err := svc.Import(vaultID, userID, payload)
			if err != nil {
				t.Fatal(err)
			}
			if !fired {
				t.Fatal("deletion boundary was not exercised")
			}
			if len(result.Errors) == 0 {
				t.Fatal("deleted company association must be reported")
			}
			var jobs, companies int64
			if err := svc.DB.Model(&models.ContactCompany{}).Where("company_id = ?", company.ID).Count(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.DB.Model(&models.Company{}).Where("id = ?", company.ID).Count(&companies).Error; err != nil {
				t.Fatal(err)
			}
			if jobs != 0 || companies != 0 {
				t.Fatalf("deleted company retained: jobs=%d companies=%d", jobs, companies)
			}
			var contacts []models.Contact
			if err := svc.DB.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
				t.Fatal(err)
			}
			if boundary == "contact" {
				if result.ImportedContacts != 0 || len(contacts) != 0 {
					t.Fatal("rejected contact creation was partly committed")
				}
			} else {
				if result.ImportedContacts != 1 || len(contacts) != 1 {
					t.Fatal("later job rejection lost committed imported contact")
				}
				if contacts[0].CompanyID != nil || ptrToStr(contacts[0].FirstName) != "Preserved imported name" || ptrToStr(contacts[0].JobPosition) != "Engineer" {
					t.Fatal("company cleanup did not preserve unrelated imported fields")
				}
			}
		})
	}
}

// The boundary opens two independent transactions. SQLite's ordinary :memory:
// fixture has one connection (each additional connection has a different DB),
// so use a private file with the same migrations and real account/vault setup.
func setupMonicaCompanyAssociationBoundary(t *testing.T) (*MonicaImportService, string, string) {
	t.Helper()
	if os.Getenv(testutil.PostgresDSNEnv) != "" {
		svc, vault, user, _ := setupMonicaImportTest(t)
		return svc, vault, user
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "company-import.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	registered, err := NewAuthService(db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{FirstName: "Import", LastName: "Tester", Email: "company-import@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	vault, err := NewVaultService(db).CreateVault(registered.User.AccountID, registered.User.ID, dto.CreateVaultRequest{Name: "Company import"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	return NewMonicaImportService(db, ""), vault.ID, registered.User.ID
}
