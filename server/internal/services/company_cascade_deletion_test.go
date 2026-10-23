package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func TestCompanyDeletionDuringVaultCascade(t *testing.T) {
	for _, entry := range []string{"vault", "account", "admin"} {
		for _, companyFirst := range []bool{false, true} {
			name := "cascade_first"
			if companyFirst {
				name = "company_first"
			}
			t.Run(entry+"/"+name, func(t *testing.T) { testCompanyDeletionDuringCascade(t, entry, companyFirst) })
		}
	}
}
func testCompanyDeletionDuringCascade(t *testing.T, entry string, companyFirst bool) {
	svc, vault, user, account := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL row-lock interleaving; cascade lifecycle runs on both databases")
	}
	installCompanyForeignKeys(t, svc.db)
	other, err := NewAuthService(svc.db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{FirstName: "Preserved", LastName: "Administrator", Email: "preserved@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	otherVault, err := NewVaultService(svc.db).CreateVault(other.User.AccountID, other.User.ID, dto.CreateVaultRequest{Name: "Preserved vault"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	adminService := NewAdminService(svc.db, t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := svc.db.WithContext(ctx)
	company, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Cascade company"})
	if err != nil {
		t.Fatal(err)
	}
	contact, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Cascade employee"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewContactJobService(db).Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	deleteCascade := func() error {
		switch entry {
		case "account":
			return NewAccountCancelService(db).Cancel(user, account, "password123")
		case "admin":
			adminService.db = db
			return adminService.DeleteUser(other.User.ID, user)
		default:
			return NewVaultService(db).DeleteVault(vault)
		}
	}
	deleteCompany := func() error { return NewCompanyService(db).Delete(company.ID, vault) }
	firstOperation, secondOperation := deleteCascade, deleteCompany
	if companyFirst {
		firstOperation, secondOperation = deleteCompany, deleteCascade
	}
	entered, resume := make(chan struct{}), make(chan struct{})
	var once, barrier sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	firstDone, secondDone := make(chan error, 1), make(chan error, 1)
	firstFinished, secondStarted, secondFinished := false, false, false
	const callback = "pause_cascade_company_job_cleanup"
	if err := db.Callback().Delete().After("gorm:delete").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_companies" {
			barrier.Do(func() { close(entered); <-resume })
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		release()
		if !firstFinished {
			<-firstDone
		}
		if secondStarted && !secondFinished {
			<-secondDone
		}
		db.Callback().Delete().Remove(callback)
	}()
	go func() { firstDone <- firstOperation() }()
	select {
	case <-entered:
	case err := <-firstDone:
		firstFinished = true
		t.Fatalf("first deletion missed boundary: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	secondStarted = true
	go func() { secondDone <- secondOperation() }()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	waiting := false
	for !waiting {
		select {
		case err := <-secondDone:
			secondFinished = true
			t.Fatalf("second deletion completed before boundary: %v", err)
		case <-deadline.C:
			t.Fatal("second deletion did not reach lock")
		case <-tick.C:
			var count int64
			if err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND (a.query LIKE 'DELETE FROM "contact_companies"%' OR a.query LIKE '%companies%FOR %UPDATE%') AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation IN ('contact_companies'::regclass,'companies'::regclass))`).Scan(&count).Error; err != nil {
				t.Fatal(err)
			}
			waiting = count > 0
		}
	}
	release()
	firstErr := <-firstDone
	firstFinished = true
	secondErr := <-secondDone
	secondFinished = true
	vaultErr, companyErr := firstErr, secondErr
	if companyFirst {
		vaultErr, companyErr = secondErr, firstErr
	}
	var vaultCount, jobCount int64
	if err := db.Model(&models.Vault{}).Where("id = ?", vault).Count(&vaultCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&jobCount).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("vault result=%v company result=%v persisted vaults=%d jobs=%d", vaultErr, companyErr, vaultCount, jobCount)
	if vaultErr != nil || (companyFirst && companyErr != nil) || (!companyFirst && !errors.Is(companyErr, ErrCompanyNotFound)) {
		t.Fatalf("company and vault deletion must serialize without deadlock: vault=%v company=%v", vaultErr, companyErr)
	}
	if vaultCount != 0 || jobCount != 0 {
		t.Fatal("successful cascade left persistent data")
	}
	var remaining int64
	for _, model := range []interface{}{&models.Company{}, &models.Contact{}} {
		if err := db.Unscoped().Model(model).Where("vault_id = ?", vault).Count(&remaining).Error; err != nil || remaining != 0 {
			t.Fatalf("cascade retained %T: %d %v", model, remaining, err)
		}
	}
	var preserved models.Vault
	if err := db.First(&preserved, "id = ?", otherVault.ID).Error; err != nil || preserved.Name != "Preserved vault" {
		t.Fatalf("unrelated vault changed: %v", err)
	}
	if entry != "vault" {
		if err := db.Model(&models.Account{}).Where("id = ?", account).Count(&remaining).Error; err != nil || remaining != 0 {
			t.Fatalf("account retained: %d %v", remaining, err)
		}
	}
}

// The same cascade must preserve all rows on failure on either supported DB,
// including legacy company references and jobs on soft-deleted contacts.
func TestCompanyCascadeLifecycleAndRollback(t *testing.T) {
	for _, entry := range []string{"vault", "account", "admin"} {
		for _, reject := range []bool{false, true} {
			outcome := "commit"
			if reject {
				outcome = "rollback"
			}
			t.Run(entry+"/"+outcome, func(t *testing.T) {
				svc, vault, user, account := setupContactTest(t)
				db := svc.db
				installCompanyForeignKeys(t, db)
				company, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Preserved employer"})
				if err != nil {
					t.Fatal(err)
				}
				contact, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Preserved employee"})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := NewContactJobService(db).LegacyUpdate(contact.ID, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Preserved role"}); err != nil {
					t.Fatal(err)
				}
				if err := db.Delete(&models.Contact{}, "id = ?", contact.ID).Error; err != nil {
					t.Fatal(err)
				}
				var jobsBefore int64
				if err := db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&jobsBefore).Error; err != nil || jobsBefore != 1 {
					t.Fatalf("fixture job missing: %d %v", jobsBefore, err)
				}
				other, err := NewAuthService(db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{FirstName: "Other", LastName: "Administrator", Email: "other-administrator@example.com", Password: "password123"}, "en")
				if err != nil {
					t.Fatal(err)
				}
				otherVault := models.Vault{AccountID: other.User.AccountID, Name: "Unrelated vault"}
				if err := db.Create(&otherVault).Error; err != nil {
					t.Fatal(err)
				}
				injected := errors.New("synthetic company cascade rejection")
				const callback = "reject_cascade_company_delete"
				if reject {
					if err := db.Callback().Delete().Before("gorm:delete").Register(callback, func(tx *gorm.DB) {
						if tx.Statement.Table == "companies" {
							tx.AddError(injected)
						}
					}); err != nil {
						t.Fatal(err)
					}
					defer db.Callback().Delete().Remove(callback)
				}
				switch entry {
				case "account":
					err = NewAccountCancelService(db).Cancel(user, account, "password123")
				case "admin":
					err = NewAdminService(db, t.TempDir()).DeleteUser(other.User.ID, user)
				default:
					err = NewVaultService(db).DeleteVault(vault)
				}
				if reject && !errors.Is(err, injected) {
					t.Fatalf("expected atomic rollback: %v", err)
				}
				if !reject && err != nil {
					t.Fatal(err)
				}
				expected := int64(0)
				if reject {
					expected = 1
				}
				for _, model := range []interface{}{&models.Vault{}, &models.Company{}, &models.Contact{}, &models.ContactCompany{}} {
					query := db.Unscoped().Model(model)
					switch model.(type) {
					case *models.Vault:
						query = query.Where("id = ?", vault)
					case *models.ContactCompany:
						query = query.Where("contact_id = ?", contact.ID)
					default:
						query = query.Where("vault_id = ?", vault)
					}
					var count int64
					if err := query.Count(&count).Error; err != nil || count != expected {
						t.Fatalf("%T count=%d want=%d err=%v", model, count, expected, err)
					}
				}
				var preserved models.Vault
				if err := db.First(&preserved, "id = ?", otherVault.ID).Error; err != nil || preserved.Name != "Unrelated vault" {
					t.Fatalf("unrelated vault changed: %v", err)
				}
				if reject {
					var historical models.Contact
					if err := db.Unscoped().First(&historical, "id = ?", contact.ID).Error; err != nil {
						t.Fatal(err)
					}
					if !historical.DeletedAt.Valid || historical.CompanyID == nil || *historical.CompanyID != company.ID || ptrToStr(historical.JobPosition) != "Preserved role" || ptrToStr(historical.FirstName) != "Preserved employee" {
						t.Fatal("rollback lost historical profile")
					}
					var job models.ContactCompany
					if err := db.First(&job, "contact_id = ?", contact.ID).Error; err != nil {
						t.Fatal(err)
					}
					if job.CompanyID != company.ID || ptrToStr(job.JobPosition) != "Preserved role" {
						t.Fatal("rollback changed job")
					}
				}
			})
		}
	}
}
