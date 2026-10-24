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

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type vaultCompanyDiscoveryContext struct{}

func TestVaultCascadeConcurrentNewCompanyAssociation(t *testing.T) {
	for _, entry := range []string{"vault", "account", "admin"} {
		for _, writer := range []string{"create", "employee"} {
			for _, reject := range []bool{false, true} {
				outcome := "commit"
				if reject {
					outcome = "rollback"
				}
				t.Run(entry+"/"+writer+"/"+outcome, func(t *testing.T) { testVaultCascadeCompanyDiscovery(t, entry, writer, reject) })
			}
		}
	}
}
func testVaultCascadeCompanyDiscovery(t *testing.T, entry, writer string, reject bool) {
	svc, vault, user, account := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL lock interleaving; cascade lifecycle and rollback run on both databases")
	}
	installCompanyForeignKeys(t, svc.db)
	if err := svc.db.Migrator().CreateConstraint(&models.Vault{}, "Companies"); err != nil {
		t.Fatal(err)
	}
	if !svc.db.Migrator().HasConstraint(&models.Vault{}, "Companies") {
		t.Fatal("real company-vault FK missing")
	}
	// Account cascades visit an earlier vault before the discovery conflict;
	// retries and final failure must roll back that earlier deletion as well.
	_, err := NewVaultService(svc.db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Earlier cascade scope"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	var originalVaults []models.Vault
	if err := svc.db.Where("account_id = ?", account).Order("id ASC").Find(&originalVaults).Error; err != nil {
		t.Fatal(err)
	}
	vault = originalVaults[len(originalVaults)-1].ID
	other, err := NewAuthService(svc.db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{FirstName: "Unrelated", LastName: "Administrator", Email: "cascade-observer@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	unrelated := models.Vault{AccountID: other.User.AccountID, Name: "Unrelated vault"}
	if err := svc.db.Create(&unrelated).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.First(&unrelated, "id = ?", unrelated.ID).Error; err != nil {
		t.Fatal(err)
	}
	contact, err := svc.CreateContact(vault, user, dto.CreateContactRequest{FirstName: "Synthetic cascade owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Already discovered employer"}); err != nil {
		t.Fatal(err)
	}
	var originalOwner models.Contact
	if err := svc.db.First(&originalOwner, "id = ?", contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := svc.db.WithContext(ctx)
	reached, resume := make(chan struct{}), make(chan struct{})
	var barrier, releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(resume) }) }
	cascadeDone, jobDone := make(chan error, 1), make(chan error, 1)
	cascadeFinished, jobStarted, jobFinished := false, false, false
	const callback = "test:pause_vault_company_discovery"
	injected := errors.New("synthetic cascade rollback")
	const failureCallback = "test:reject_discovered_vault_delete"
	if reject {
		if err := db.Callback().Delete().Before("gorm:delete").Register(failureCallback, func(tx *gorm.DB) {
			if tx.Statement.Context.Value(vaultCompanyDiscoveryContext{}) == true && tx.Statement.Table == "vaults" {
				for _, value := range tx.Statement.Vars {
					if value == vault {
						tx.AddError(injected)
					}
				}
				// Before gorm:delete the SQL may not be built; the WHERE clause carries the ID.
				if where, ok := tx.Statement.Clauses["WHERE"]; ok && strings.Contains(fmt.Sprint(where.Expression), vault) {
					tx.AddError(injected)
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.Callback().Delete().Remove(failureCallback)
	}
	if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Context.Value(vaultCompanyDiscoveryContext{}) == true && tx.Statement.Table == "contacts" && strings.Contains(tx.Statement.SQL.String(), "FOR UPDATE") && strings.Contains(fmt.Sprint(tx.Statement.Vars), vault) {
			barrier.Do(func() {
				close(reached)
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
	defer func() {
		release()
		if !cascadeFinished {
			<-cascadeDone
		}
		if jobStarted && !jobFinished {
			<-jobDone
		}
		db.Callback().Query().Remove(callback)
	}()
	go func() {
		scoped := db.WithContext(context.WithValue(ctx, vaultCompanyDiscoveryContext{}, true))
		switch entry {
		case "account":
			cascadeDone <- NewAccountCancelService(scoped).Cancel(user, account, "password123")
		case "admin":
			cascadeDone <- NewAdminService(scoped, t.TempDir()).DeleteUser(other.User.ID, user)
		default:
			cascadeDone <- NewVaultService(scoped).DeleteVault(vault)
		}
	}()
	select {
	case <-reached:
	case err := <-cascadeDone:
		cascadeFinished = true
		t.Fatalf("cascade missed barrier: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	newer, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Employer created after discovery"})
	if err != nil {
		t.Fatal(err)
	}
	jobStarted = true
	go func() {
		var err error
		if writer == "employee" {
			_, err = NewContactJobService(db).AddEmployee(newer.ID, vault, dto.AddEmployeeRequest{ContactID: contact.ID, JobPosition: "Concurrent role"})
		} else {
			_, err = NewContactJobService(db).Create(contact.ID, vault, dto.CreateContactJobRequest{CompanyID: newer.ID, JobPosition: "Concurrent role"})
		}
		jobDone <- err
	}()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	waiting := false
	for !waiting {
		select {
		case err := <-jobDone:
			jobFinished = true
			t.Fatalf("job did not wait for cascade contact: %v", err)
		case <-deadline.C:
			t.Fatal("job contact-lock wait not observed")
		case <-tick.C:
			var count int64
			err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE '%contacts%FOR UPDATE%' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='contacts'::regclass)`).Scan(&count).Error
			if err != nil {
				t.Fatal(err)
			}
			waiting = count > 0
		}
	}
	t.Log("new company committed after cascade discovery; job holds its company lock and is waiting for the cascade-owned contact")
	release()
	cascadeErr := <-cascadeDone
	cascadeFinished = true
	jobErr := <-jobDone
	jobFinished = true
	var vaults, jobs int64
	if err := db.Model(&models.Vault{}).Where("id = ?", vault).Count(&vaults).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("cascade=%v job=%v persisted vaults=%d jobs=%d", cascadeErr, jobErr, vaults, jobs)
	if jobErr != nil {
		t.Errorf("already admitted job must commit without deadlock: %v", jobErr)
	}
	if reject {
		if !errors.Is(cascadeErr, injected) {
			t.Errorf("cascade must roll back injected failure: %v", cascadeErr)
		}
		var afterVaults []models.Vault
		if err := db.Where("account_id = ?", account).Order("id ASC").Find(&afterVaults).Error; err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(originalVaults, afterVaults) {
			t.Error("retry/rollback lost or changed an earlier vault")
		}
		var stored models.ContactCompany
		if err := db.First(&stored, "contact_id = ?", contact.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.CompanyID != newer.ID || ptrToStr(stored.JobPosition) != "Concurrent role" || jobs != 1 || vaults != 1 {
			t.Error("rollback lost committed employment")
		}
		var owner models.Contact
		if err := db.First(&owner, "id = ?", contact.ID).Error; err != nil {
			t.Fatal(err)
		}
		// The first local Job uses the existing ID fallback; creation need not
		// materialize PreferredJobID. Only its normal edit timestamps may change.
		originalOwner.UpdatedAt, originalOwner.LastUpdatedAt = owner.UpdatedAt, owner.LastUpdatedAt
		if !reflect.DeepEqual(originalOwner, owner) {
			t.Error("rollback damaged unrelated profile or selection")
		}
		loaded, err := loadContactEmployment(db, contact.ID, vault)
		if err != nil {
			t.Fatal(err)
		}
		projected := ContactVCardEmployments(loaded)
		if len(projected) != 1 || projected[0].JobID != stored.ID || projected[0].Title != "Concurrent role" {
			t.Error("rollback changed employment identity or export")
		}

	} else if cascadeErr != nil || vaults != 0 || jobs != 0 {
		t.Errorf("cascade must serialize and delete its rows: %v vaults=%d jobs=%d", cascadeErr, vaults, jobs)
	}
	var kept models.Vault
	if err := db.First(&kept, "id = ?", unrelated.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(kept, unrelated) {
		t.Error("cascade modified unrelated account vault")
	}
}

// Exercise both sides of the resolution boundary: a creator already holding
// exclusion forces rollback before waiting; a later creator sees a deleted owner.
func TestCompanyCreationAtVaultCascadeExclusion(t *testing.T) {
	for _, phase := range []string{"creator first", "cascade first"} {
		t.Run(phase, func(t *testing.T) {
			svc, vault, _, _ := setupContactTest(t)
			if svc.db.Dialector.Name() != "postgres" {
				t.Skip("requires PostgreSQL advisory lock interleaving")
			}
			if err := svc.db.Migrator().CreateConstraint(&models.Vault{}, "Companies"); err != nil {
				t.Fatal(err)
			}
			if !svc.db.Migrator().HasConstraint(&models.Vault{}, "Companies") {
				t.Fatal("company-vault FK missing")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			db := svc.db.WithContext(ctx)
			reached, resume := make(chan struct{}), make(chan struct{})
			creating, allowCreate := make(chan struct{}), make(chan struct{})
			var barrier, creatorBarrier, releaseOnce, createReleaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(resume) }) }
			releaseCreate := func() { createReleaseOnce.Do(func() { close(allowCreate) }) }
			pause := func(tx *gorm.DB) {
				barrier.Do(func() {
					close(reached)
					select {
					case <-resume:
					case <-ctx.Done():
						tx.AddError(ctx.Err())
					}
				})
			}
			const hook = "test:cascade_resolution_boundary"
			if err := db.Callback().Query().After("gorm:query").Register(hook, func(tx *gorm.DB) {
				if phase == "creator first" && tx.Statement.Context.Value(vaultCompanyDiscoveryContext{}) == true && tx.Statement.Table == "contacts" && strings.Contains(tx.Statement.SQL.String(), "FOR UPDATE") {
					pause(tx)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Query().Remove(hook)
			if err := db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
				if phase == "cascade first" && tx.Statement.Context.Value(vaultCompanyDiscoveryContext{}) == true && tx.Statement.Table == "mood_tracking_events" {
					pause(tx)
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Delete().Remove(hook)
			if err := db.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
				if phase == "creator first" && tx.Statement.Table == "companies" {
					creatorBarrier.Do(func() {
						close(creating)
						select {
						case <-allowCreate:
						case <-ctx.Done():
							tx.AddError(ctx.Err())
						}
					})
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Create().Remove(hook)
			cascadeDone, createDone := make(chan error, 1), make(chan error, 1)
			cascadeFinished, createStarted, createFinished := false, false, false
			defer func() {
				release()
				releaseCreate()
				if !cascadeFinished {
					<-cascadeDone
				}
				if createStarted && !createFinished {
					<-createDone
				}
			}()
			go func() {
				cascadeDone <- NewVaultService(db.WithContext(context.WithValue(ctx, vaultCompanyDiscoveryContext{}, true))).DeleteVault(vault)
			}()
			select {
			case <-reached:
			case err := <-cascadeDone:
				cascadeFinished = true
				t.Fatalf("missed cascade boundary: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			createStarted = true
			go func() {
				_, err := NewCompanyService(db).Create(vault, dto.CreateCompanyRequest{Name: "Boundary company"})
				createDone <- err
			}()
			if phase == "creator first" {
				select {
				case <-creating:
				case err := <-createDone:
					createFinished = true
					t.Fatalf("creator missed name lock: %v", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				release()
			}
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			tick := time.NewTicker(10 * time.Millisecond)
			defer tick.Stop()
			waiting := false
			for !waiting {
				select {
				case <-timer.C:
					t.Fatal("no resolution lock wait observed")
				case <-tick.C:
					var count int64
					if err := db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND a.query LIKE 'SELECT pg_advisory_xact_lock%'`).Scan(&count).Error; err != nil {
						t.Fatal(err)
					}
					waiting = count > 0
				}
			}
			release()
			releaseCreate()
			cascadeErr := <-cascadeDone
			cascadeFinished = true
			createErr := <-createDone
			createFinished = true
			if cascadeErr != nil {
				t.Fatal(cascadeErr)
			}
			if phase == "creator first" && createErr != nil || phase == "cascade first" && !errors.Is(createErr, ErrVaultNotFound) {
				t.Fatalf("wrong creator result: %v", createErr)
			}
			var count int64
			if err := db.Model(&models.Company{}).Where("vault_id = ?", vault).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("orphan company: %d %v", count, err)
			}
			if err := db.Model(&models.Vault{}).Where("id = ?", vault).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("vault survived: %d %v", count, err)
			}
		})
	}
}

func TestCompanyCreateRejectsDeletedVault(t *testing.T) {
	svc, vault, _, _ := setupContactTest(t)
	if err := NewVaultService(svc.db).DeleteVault(vault); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCompanyService(svc.db).Create(vault, dto.CreateCompanyRequest{Name: "Late company"}); !errors.Is(err, ErrVaultNotFound) {
		t.Fatalf("deleted vault create=%v", err)
	}
	var count int64
	if err := svc.db.Model(&models.Company{}).Where("vault_id = ?", vault).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("created orphan company: %d %v", count, err)
	}
}
