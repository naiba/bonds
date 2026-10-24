package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// Per-vault company ordering is not global transaction ordering: the second
// vault's merge may already own a company while waiting for the first vault.
func TestAccountCascadeSerializesCrossVaultMerge(t *testing.T) {
	for _, entry := range []string{"cancel", "admin"} {
		for _, boundary := range []string{"relationship", "historical_employer", "contact_only"} {
			for _, rollback := range []bool{false, true} {
				outcome := "commit"
				if rollback {
					outcome = "rollback"
				}
				t.Run(entry+"/"+boundary+"/"+outcome, func(t *testing.T) { testAccountCascadeMerge(t, entry, boundary, rollback) })
			}
		}
	}
}

func testAccountCascadeMerge(t *testing.T, entry, boundary string, rollback bool) {
	svc, _, user, account := setupContactTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL deterministic row-lock boundary; account lifecycle and rollback also run on SQLite")
	}
	db := svc.db
	installCompanyForeignKeys(t, db)
	if _, err := NewVaultService(db).CreateVault(account, user, dto.CreateVaultRequest{Name: "Second deletion scope"}, "en"); err != nil {
		t.Fatal(err)
	}
	var originalVaults []models.Vault
	if err := db.Where("account_id = ?", account).Order("id ASC").Find(&originalVaults).Error; err != nil {
		t.Fatal(err)
	}
	if len(originalVaults) != 2 {
		t.Fatalf("vaults=%d", len(originalVaults))
	}
	first, last := originalVaults[0].ID, originalVaults[1].ID
	low, err := NewCompanyService(db).Create(last, dto.CreateCompanyRequest{Name: "Lower employer identity"})
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateContact(last, user, dto.CreateContactRequest{FirstName: "Merge survivor"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := svc.CreateContact(last, user, dto.CreateContactRequest{FirstName: "Merge source"})
	if err != nil {
		t.Fatal(err)
	}
	if boundary != "contact_only" {
		if _, err := NewContactJobService(db).Create(target.ID, last, dto.CreateContactJobRequest{CompanyID: low.ID, JobPosition: "Local role"}); err != nil {
			t.Fatal(err)
		}
	}
	lockTable := "contacts"
	neighborID := ""
	if boundary == "historical_employer" {
		high, err := NewCompanyService(db).Create(first, dto.CreateCompanyRequest{Name: "Higher employer identity"})
		if err != nil {
			t.Fatal(err)
		}
		if high.ID <= low.ID {
			t.Fatal("fixture must reverse vault and company ordering")
		}
		createHistoricalCompanyReference(t, db, source.ID, last, user, high.ID)
		lockTable = "companies"
	} else {
		neighborVault := first
		if boundary == "contact_only" {
			neighborVault = last
		}
		neighbor, err := svc.CreateContact(neighborVault, user, dto.CreateContactRequest{FirstName: "Cross-vault relationship owner"})
		if err != nil {
			t.Fatal(err)
		}
		if boundary == "contact_only" {
			// No company gate: force merge to own a later-vault contact before
			// it waits on the first vault's relationship owner. Build the
			// cross-vault fixture via the real move service before concurrency.
			contacts := []*dto.ContactResponse{target, source, neighbor}
			slices.SortFunc(contacts, func(a, b *dto.ContactResponse) int { return strings.Compare(a.ID, b.ID) })
			target, source, neighbor = contacts[0], contacts[1], contacts[2]
			if _, err := NewContactMoveService(db).Move(neighbor.ID, last, first, user); err != nil {
				t.Fatal(err)
			}
		}
		neighborID = neighbor.ID
		var kind models.RelationshipType
		if err := db.First(&kind).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := NewRelationshipService(db).Create(neighbor.ID, first, user, dto.CreateRelationshipRequest{RelatedContactID: source.ID, RelationshipTypeID: kind.ID}); err != nil {
			t.Fatal(err)
		}
	}
	var originalCompanies []models.Company
	if err := db.Where("vault_id IN ?", []string{first, last}).Order("id ASC").Find(&originalCompanies).Error; err != nil {
		t.Fatal(err)
	}
	var originalContacts []models.Contact
	if err := db.Unscoped().Where("vault_id IN ?", []string{first, last}).Order("id ASC").Find(&originalContacts).Error; err != nil {
		t.Fatal(err)
	}
	var originalJobs []models.ContactCompany
	if err := db.Where("contact_id IN ?", []string{target.ID, source.ID}).Order("id ASC").Find(&originalJobs).Error; err != nil {
		t.Fatal(err)
	}
	other, err := NewAuthService(db, testutil.TestJWTConfig()).Register(dto.RegisterRequest{FirstName: "Unrelated", LastName: "Administrator", Email: "account-cascade-observer@example.com", Password: "password123"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	unrelated := models.Vault{AccountID: other.User.AccountID, Name: "Unrelated account vault"}
	if err := db.Create(&unrelated).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&unrelated, "id = ?", unrelated.ID).Error; err != nil {
		t.Fatal(err)
	}
	var unrelatedUser models.User
	if err := db.First(&unrelatedUser, "id = ?", other.User.ID).Error; err != nil {
		t.Fatal(err)
	}
	request := reviewedContactMerge(t, svc, last, user, target.ID, source.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	type accountCascadeContext struct{}
	entered, resume := make(chan struct{}), make(chan struct{})
	var reached, releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(resume) }) }
	const hook = "test:account_cascade_first_vault_lock"
	if err := db.Callback().Query().After("gorm:query").Register(hook, func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Table != lockTable || tx.Statement.Context.Value(accountCascadeContext{}) != true {
			return
		}
		if _, ok := tx.Statement.Clauses["FOR"]; !ok {
			return
		}
		reached.Do(func() {
			close(entered)
			select {
			case <-resume:
			case <-ctx.Done():
				tx.AddError(ctx.Err())
			}
		})
	}); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("synthetic account cascade rollback")
	const rejectHook = "test:reject_account_cascade_last_vault"
	if rollback {
		if err := db.Callback().Delete().Before("gorm:delete").Register(rejectHook, func(tx *gorm.DB) {
			if tx.Statement.Context.Value(accountCascadeContext{}) != true || tx.Statement.Table != "vaults" {
				return
			}
			if where, ok := tx.Statement.Clauses["WHERE"]; ok && strings.Contains(fmt.Sprint(where.Expression), last) {
				tx.AddError(injected)
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.Callback().Delete().Remove(rejectHook)
	}
	cascadeDone, mergeDone := make(chan error, 1), make(chan error, 1)
	cascadeFinished, mergeFinished, mergeStarted := false, false, false
	defer func() {
		release()
		if !cascadeFinished {
			<-cascadeDone
		}
		if mergeStarted && !mergeFinished {
			<-mergeDone
		}
		db.Callback().Query().Remove(hook)
	}()
	uploadDir := t.TempDir()
	go func() {
		scoped := db.WithContext(context.WithValue(ctx, accountCascadeContext{}, true))
		if entry == "admin" {
			cascadeDone <- NewAdminService(scoped, uploadDir).DeleteUser(other.User.ID, user)
		} else {
			cascadeDone <- NewAccountCancelService(scoped).Cancel(user, account, "password123")
		}
	}()
	select {
	case <-entered:
	case err := <-cascadeDone:
		cascadeFinished = true
		t.Fatalf("cascade missed first-vault lock: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	mergeStarted = true
	go func() {
		_, err := NewContactService(db.WithContext(ctx)).MergeContacts(last, user, request)
		mergeDone <- err
	}()
	var early bool
	if boundary == "historical_employer" {
		early, err = waitForCompanyLockOrCompletion(t, db.WithContext(ctx), mergeDone)
	} else {
		early, err = waitForHistoricalContactLock(t, db.WithContext(ctx), mergeDone)
	}
	mergeFinished = early
	if early {
		t.Fatalf("merge missed competing lock: %v", err)
	}
	release()
	cascadeErr := <-cascadeDone
	cascadeFinished = true
	mergeErr := <-mergeDone
	mergeFinished = true
	t.Logf("cascade=%v merge=%v", cascadeErr, mergeErr)
	if rollback {
		if !errors.Is(cascadeErr, injected) {
			t.Errorf("expected full rollback: %v", cascadeErr)
		}
	} else if cascadeErr != nil {
		t.Errorf("account cascade failed: %v", cascadeErr)
	}
	if mergeErr != nil && !errors.Is(mergeErr, ErrContactNotFound) && !errors.Is(mergeErr, ErrContactMergeReviewChanged) {
		t.Errorf("merge must complete or return a recoverable domain error: %v", mergeErr)
	}
	var accountCount, userCount int64
	if err := db.Model(&models.Account{}).Where("id = ?", account).Count(&accountCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", user).Count(&userCount).Error; err != nil {
		t.Fatal(err)
	}
	var vaults []models.Vault
	if err := db.Where("account_id = ?", account).Order("id ASC").Find(&vaults).Error; err != nil {
		t.Fatal(err)
	}
	var companies []models.Company
	if err := db.Where("vault_id IN ?", []string{first, last}).Order("id ASC").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	var contacts []models.Contact
	if err := db.Unscoped().Where("vault_id IN ?", []string{first, last}).Order("id ASC").Find(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	var jobs []models.ContactCompany
	if err := db.Where("contact_id IN ?", []string{target.ID, source.ID}).Order("id ASC").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if rollback {
		if accountCount != 1 || userCount != 1 || !reflect.DeepEqual(vaults, originalVaults) || !reflect.DeepEqual(companies, originalCompanies) {
			t.Error("rollback lost account, user, earlier vault or company state")
		}
		if mergeErr != nil {
			if !reflect.DeepEqual(contacts, originalContacts) || !reflect.DeepEqual(jobs, originalJobs) {
				t.Error("rejected merge or cascade changed original contacts/jobs")
			}
		} else {
			if len(contacts) != len(originalContacts) || len(jobs) != len(originalJobs) {
				t.Fatal("rollback lost merged contact or employment rows")
			}
			for i, job := range jobs {
				before := originalJobs[i]
				if job.ID != before.ID || job.CompanyID != before.CompanyID || job.ContactID != target.ID || ptrToStr(job.JobPosition) != ptrToStr(before.JobPosition) {
					t.Error("rollback damaged merged employment identity/content")
				}
			}
			var survivor, tombstone models.Contact
			if err := db.First(&survivor, "id = ?", target.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Unscoped().First(&tombstone, "id = ?", source.ID).Error; err != nil {
				t.Fatal(err)
			}
			if ptrToStr(survivor.FirstName) != target.FirstName || !tombstone.DeletedAt.Valid {
				t.Error("rollback damaged legitimate merge")
			}
			if neighborID != "" {
				var count int64
				if err := db.Model(&models.Relationship{}).Where("contact_id = ? AND related_contact_id = ?", neighborID, target.ID).Count(&count).Error; err != nil || count != 1 {
					t.Fatalf("merged relationship lost: %d %v", count, err)
				}
			}
		}
	} else if accountCount != 0 || userCount != 0 || len(vaults) != 0 || len(companies) != 0 || len(contacts) != 0 || len(jobs) != 0 {
		t.Error("successful account cascade left owned data")
	}
	var preserved models.Vault
	if err := db.First(&preserved, "id = ?", unrelated.ID).Error; err != nil {
		t.Fatal(err)
	}
	var preservedUser models.User
	if err := db.First(&preservedUser, "id = ?", unrelatedUser.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preserved, unrelated) || !reflect.DeepEqual(preservedUser, unrelatedUser) {
		t.Error("cascade changed unrelated account state")
	}
}
