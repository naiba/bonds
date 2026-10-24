package services

import (
	"context"
	"errors"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmploymentLegacyRelationshipSurvivesOtherEmploymentRemoval(t *testing.T) {
	for _, removal := range []string{"job", "employee", "company"} {
		t.Run(removal, func(t *testing.T) {
			svc, id, vault, _ := setupVCardTest(t)
			legacy := models.Company{VaultID: vault, Name: "Legacy Acme"}
			other := models.Company{VaultID: vault, Name: "Other Beta"}
			for _, c := range []*models.Company{&legacy, &other} {
				if err := svc.db.Create(c).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": legacy.ID, "job_position": "Legacy Engineer"}).Error; err != nil {
				t.Fatal(err)
			}
			jobs := NewContactJobService(svc.db)
			job, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: other.ID, JobPosition: "Other Consultant"})
			if err != nil {
				t.Fatal(err)
			}
			switch removal {
			case "job":
				err = jobs.Delete(id, vault, job.ID)
			case "employee":
				err = jobs.RemoveEmployee(other.ID, vault, id)
			case "company":
				err = NewCompanyService(svc.db).Delete(other.ID, vault)
			}
			if err != nil {
				t.Fatal(err)
			}
			var contact models.Contact
			if err := svc.db.First(&contact, "id = ?", id).Error; err != nil {
				t.Fatal(err)
			}
			var kept int64
			if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ? AND company_id = ? AND job_position = ?", id, legacy.ID, "Legacy Engineer").Count(&kept).Error; err != nil {
				t.Fatal(err)
			}
			t.Logf("after %s legacy company=%v title=%q retained jobs=%d", removal, contact.CompanyID, ptrToStr(contact.JobPosition), kept)
			if kept == 0 && (contact.CompanyID == nil || *contact.CompanyID != legacy.ID || ptrToStr(contact.JobPosition) != "Legacy Engineer") {
				t.Error("deleting unrelated employment permanently erased legacy relationship")
			}
		})
	}
}
func TestEmploymentCompanyDeletionConcurrentNewEmployee(t *testing.T) {
	svc, id, vault, user := setupVCardTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("requires PostgreSQL concurrent transactions")
	}
	company := models.Company{VaultID: vault, Name: "Concurrent Acme"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	installCompanyForeignKeys(t, svc.db)
	var before models.Contact
	if err := svc.db.First(&before, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entered, resume := make(chan struct{}), make(chan struct{})
	var released bool
	release := func() {
		if !released {
			close(resume)
			released = true
		}
	}
	done, writerDone := make(chan error, 1), make(chan error, 1)
	deleteFinished, writerFinished, writerStarted := false, false, false
	const hook = "employment:pause_company_cleanup_with_owner_set_locked"
	if err := svc.db.Callback().Delete().Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "contact_companies" {
			close(entered)
			select {
			case <-resume:
			case <-ctx.Done():
				tx.AddError(ctx.Err())
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		release()
		if !deleteFinished {
			<-done
		}
		if writerStarted && !writerFinished {
			<-writerDone
		}
		svc.db.Callback().Delete().Remove(hook)
	}()
	go func() { done <- NewCompanyService(svc.db.WithContext(ctx)).Delete(company.ID, vault) }()
	select {
	case <-entered:
	case err := <-done:
		deleteFinished = true
		t.Fatalf("delete missed boundary: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Company-first discovery excludes new associations until commit. Observe
	// the writer's real lock wait instead of depending on a []string SQL shape.
	writerStarted = true
	go func() {
		_, err := NewContactJobService(svc.db.WithContext(ctx)).LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"})
		writerDone <- err
	}()
	early, writeErr := waitForCompanyLockOrCompletion(t, svc.db, writerDone)
	writerFinished = early
	release()
	deleteErr := <-done
	deleteFinished = true
	if !writerFinished {
		writeErr = <-writerDone
		writerFinished = true
	}
	if early || deleteErr != nil || !errors.Is(writeErr, ErrCompanyNotFound) {
		t.Fatalf("company/association did not serialize: early=%t delete=%v write=%v", early, deleteErr, writeErr)
	}
	var after models.Contact
	if err := svc.db.First(&after, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected association changed profile or selection")
	}
	for _, model := range []any{&models.Company{}, &models.ContactCompany{}} {
		var count int64
		if err := svc.db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("company deletion left records")
		}
	}
}

func TestEmploymentSubscriptionPreservesSelectedCompanyTitle(t *testing.T) {
	svc, id, vault, user := setupVCardTest(t)
	var account models.User
	if err := svc.db.First(&account, "id = ?", user).Error; err != nil {
		t.Fatal(err)
	}
	jobs := NewContactJobService(svc.db)
	var primary uint
	for i, name := range []string{"Acme", "Beta"} {
		company := models.Company{VaultID: vault, Name: name}
		if err := svc.db.Create(&company).Error; err != nil {
			t.Fatal(err)
		}
		title := ""
		if i == 1 {
			title = "Consultant"
		} else {
			primary = company.ID
		}
		if _, err := jobs.Create(id, vault, dto.CreateContactJobRequest{CompanyID: company.ID, JobPosition: title}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.db.Model(&models.Contact{}).Where("id = ?", id).Updates(map[string]any{"company_id": primary, "job_position": nil}).Error; err != nil {
		t.Fatal(err)
	}
	card, err := svc.ExportContactToVCard(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	before, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Transaction(func(tx *gorm.DB) error {
		_, action, err := svc.UpsertContactFromVCard(tx, card, vault, user, account.AccountID, id, "/selected-employment.vcf", "changed", nil)
		if err == nil && action != "updated" {
			return errors.New("subscription not applied")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	after, err := jobs.List(id, vault)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unchanged subscription changed jobs")
	}
	response, err := NewContactService(svc.db).GetContact(id, user, vault)
	if err != nil {
		t.Fatal(err)
	}
	if response.CompanyID == nil || *response.CompanyID != primary || response.JobPosition != "" {
		t.Fatal("contact response combines different employments")
	}
}

func TestEmploymentAssociationWaitsForCompanyDeletion(t *testing.T) {
	svc, id, vault, user := setupVCardTest(t)
	if svc.db.Dialector.Name() != "postgres" {
		t.Skip("requires PostgreSQL concurrent transactions")
	}
	company := models.Company{VaultID: vault, Name: "Deleting Employer"}
	if err := svc.db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Exec("ALTER TABLE contacts ADD CONSTRAINT employment_company_constraint FOREIGN KEY (company_id) REFERENCES companies(id)").Error; err != nil {
		t.Fatal(err)
	}
	held, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "employment:pause_locked_company"
	if err := svc.db.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		c, ok := tx.Statement.Dest.(*models.Company)
		_, locking := tx.Statement.Clauses["FOR"]
		if ok && locking && c.ID == company.ID && paused.CompareAndSwap(false, true) {
			close(held)
			select {
			case <-resume:
			case <-time.After(10 * time.Second):
				tx.AddError(errors.New("test scheduling timeout"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Query().Remove(hook)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	deletion, association := make(chan error, 1), make(chan error, 1)
	go func() { deletion <- NewCompanyService(svc.db.WithContext(ctx)).Delete(company.ID, vault) }()
	select {
	case <-held:
	case <-ctx.Done():
		close(resume)
		<-deletion
		t.Fatal("company lock not reached")
	}
	go func() {
		_, err := NewContactJobService(svc.db.WithContext(ctx)).LegacyUpdate(id, vault, user, dto.UpdateJobInfoRequest{CompanyID: &company.ID, JobPosition: "Engineer"})
		association <- err
	}()
	waiting := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var n int64
		err := svc.db.Raw(`SELECT count(*) FROM pg_stat_activity a WHERE a.pid<>pg_backend_pid() AND a.wait_event_type='Lock' AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='companies'::regclass)`).Scan(&n).Error
		if err != nil {
			break
		}
		if n > 0 {
			waiting = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(resume)
	deleteErr, writeErr := <-deletion, <-association
	if !waiting || deleteErr != nil || !errors.Is(writeErr, ErrCompanyNotFound) {
		t.Fatalf("waiting=%v deletion=%v association=%v", waiting, deleteErr, writeErr)
	}
	var contact models.Contact
	if err := svc.db.First(&contact, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := svc.db.Model(&models.ContactCompany{}).Where("contact_id = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if contact.CompanyID != nil || contact.JobPosition != nil || count != 0 {
		t.Fatal("rejected association left employment state")
	}
}
