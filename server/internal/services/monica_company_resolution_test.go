package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type monicaCompanyResolutionContext struct{}

// Exercises the real Monica and vCard import services; the query barrier only
// controls scheduling after Monica has observed a missing company.
func TestMonicaConcurrentVCardCompanyResolution(t *testing.T) {
	svc, vault, user, _ := setupMonicaImportTest(t)
	if svc.DB.Dialector.Name() != "postgres" {
		t.Skip("requires PostgreSQL transactions and advisory lock observation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lookedUp, resume := make(chan struct{}), make(chan struct{})
	var paused atomic.Bool
	const hook = "employment:monica_company_candidate_barrier"
	err := svc.DB.Callback().Query().After("gorm:after_query").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Context.Value(monicaCompanyResolutionContext{}) != true || tx.Statement.Table != "companies" {
			return
		}
		if _, ok := tx.Statement.Dest.(*models.Company); !ok {
			return
		}
		if paused.CompareAndSwap(false, true) {
			close(lookedUp)
			select {
			case <-resume:
			case <-ctx.Done():
				tx.AddError(ctx.Err())
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Callback().Query().Remove(hook)
	type monicaResult struct {
		response *dto.MonicaImportResponse
		err      error
	}
	monicaData := []byte(`{"version":"1.0-preview.1","account":{"instance":{},"data":[{"type":"contacts","values":[{"uuid":"synthetic-monica-employee","properties":{"first_name":"Monica employee","company":"Shared employer","job":"Analyst"},"data":[]}]}]}}`)
	imported := make(chan monicaResult, 1)
	go func() {
		s := NewMonicaImportService(svc.DB.WithContext(context.WithValue(ctx, monicaCompanyResolutionContext{}, true)), "")
		response, err := s.Import(vault, user, monicaData)
		imported <- monicaResult{response, err}
	}()
	select {
	case <-lookedUp:
	case <-ctx.Done():
		close(resume)
		<-imported
		t.Fatal("Monica candidate query not reached")
	}
	wire := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:VCard employee\r\nORG:Shared employer\r\nTITLE:Engineer\r\nEND:VCARD\r\n"
	done := make(chan error, 1)
	pid := make(chan int, 1)
	go func() {
		done <- svc.DB.WithContext(ctx).Connection(func(conn *gorm.DB) error {
			conn = conn.Session(&gorm.Session{NewDB: true})
			var n int
			if err := conn.Raw("SELECT pg_backend_pid()").Scan(&n).Error; err != nil {
				return err
			}
			pid <- n
			r, err := NewVCardService(conn).ImportVCard(vault, user, strings.NewReader(wire))
			if err == nil && (r.ImportedCount != 1 || len(r.Contacts) != 1) {
				return fmt.Errorf("unexpected vCard import response: %+v", r)
			}
			return err
		})
	}()
	var workerPID int
	select {
	case workerPID = <-pid:
	case err := <-done:
		close(resume)
		<-imported
		t.Fatal(err)
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	finished, waiting := false, false
	var vcardErr, observationErr error
observe:
	for {
		select {
		case vcardErr = <-done:
			finished = true
			break observe
		case <-ctx.Done():
			observationErr = ctx.Err()
			break observe
		case <-ticker.C:
			var n int64
			if err := svc.DB.WithContext(ctx).Raw("SELECT count(*) FROM pg_locks WHERE pid = ? AND locktype='advisory' AND NOT granted", workerPID).Scan(&n).Error; err != nil {
				observationErr = err
				break observe
			}
			if n > 0 {
				waiting = true
				break observe
			}
		}
	}
	ticker.Stop()
	close(resume)
	mr := <-imported
	if !finished {
		vcardErr = <-done
	}
	if observationErr != nil || vcardErr != nil || mr.err != nil {
		t.Fatalf("operations failed: observe=%v vcard=%v monica=%v", observationErr, vcardErr, mr.err)
	}
	if mr.response.ImportedContacts != 1 || len(mr.response.Errors) != 0 {
		t.Fatalf("Monica import incomplete: %+v", mr.response)
	}
	var companies []models.Company
	if err := svc.DB.Where("vault_id = ? AND name = ?", vault, "Shared employer").Order("id").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	var contacts []models.Contact
	if err := svc.DB.Where("vault_id = ?", vault).Find(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	var jobs []models.ContactCompany
	if err := svc.DB.Where("contact_id IN (?)", svc.DB.Model(&models.Contact{}).Select("id").Where("vault_id = ?", vault)).Order("id").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("actual result: waiting=%t companies=%d contacts=%d jobs=%d", waiting, len(companies), len(contacts), len(jobs))
	if !waiting {
		t.Error("vCard resolved and committed while Monica still held a stale candidate snapshot")
	}
	if len(contacts) != 2 {
		t.Errorf("expected two imported contacts, got %d", len(contacts))
	}
	if len(companies) != 1 {
		t.Errorf("automatic imports must share one employer identity; got %d", len(companies))
	}
	if len(jobs) != 2 {
		t.Errorf("expected both imported employments, got %d", len(jobs))
	} else if jobs[0].CompanyID != jobs[1].CompanyID {
		t.Errorf("actual employments split between company IDs %d and %d", jobs[0].CompanyID, jobs[1].CompanyID)
	}
	next, err := NewVCardService(svc.DB).ImportVCard(vault, user, strings.NewReader(wire))
	if err != nil {
		t.Fatalf("subsequent unambiguous input rejected after automatic identity race: %v", err)
	}
	if next.ImportedCount != 1 || len(next.Contacts) != 1 {
		t.Fatalf("unexpected subsequent import: %+v", next)
	}
	var finalCompanies []models.Company
	if err := svc.DB.Where("vault_id = ?", vault).Find(&finalCompanies).Error; err != nil {
		t.Fatal(err)
	}
	if len(finalCompanies) != 1 {
		t.Fatalf("later import manufactured companies: %d", len(finalCompanies))
	}
	var finalJobs []models.ContactCompany
	if err := svc.DB.Where("contact_id IN (?)", svc.DB.Model(&models.Contact{}).Select("id").Where("vault_id = ?", vault)).Find(&finalJobs).Error; err != nil {
		t.Fatal(err)
	}
	if len(finalJobs) != 3 {
		t.Fatalf("expected three persisted jobs, got %d", len(finalJobs))
	}
	titles := map[string]int{}
	for _, job := range finalJobs {
		if job.CompanyID != finalCompanies[0].ID {
			t.Error("job did not reuse the resolved company identity")
		}
		titles[ptrToStr(job.JobPosition)]++
	}
	if titles["Analyst"] != 1 || titles["Engineer"] != 2 {
		t.Errorf("lost or overwritten job titles: %+v", titles)
	}

	repeated, err := svc.Import(vault, user, monicaData)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ImportedContacts != 0 || repeated.SkippedCount != 1 || len(repeated.Errors) != 0 {
		t.Fatalf("reimport did not preserve source identity: %+v", repeated)
	}
	var companyCount, contactCount, jobCount int64
	if err := svc.DB.Model(&models.Company{}).Where("vault_id = ?", vault).Count(&companyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.DB.Model(&models.Contact{}).Where("vault_id = ?", vault).Count(&contactCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.DB.Model(&models.ContactCompany{}).Where("contact_id IN (?)", svc.DB.Model(&models.Contact{}).Select("id").Where("vault_id = ?", vault)).Count(&jobCount).Error; err != nil {
		t.Fatal(err)
	}
	if companyCount != 1 || contactCount != 3 || jobCount != 3 {
		t.Errorf("reimport changed persisted counts: company=%d contact=%d job=%d", companyCount, contactCount, jobCount)
	}
}

func TestMonicaCompanyResolutionRollbackKeepsContact(t *testing.T) {
	svc, vault, user, _ := setupMonicaImportTest(t)
	const hook = "employment:reject_monica_company_write"
	inserted := false
	if err := svc.DB.Callback().Create().After("gorm:create").Before("gorm:commit_or_rollback_transaction").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "companies" && tx.Error == nil {
			inserted = tx.RowsAffected == 1
			tx.AddError(errors.New("synthetic company write rejected"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.DB.Callback().Create().Remove(hook)
	response, err := svc.Import(vault, user, []byte(`{"version":"1.0-preview.1","account":{"instance":{},"data":[{"type":"contacts","values":[{"uuid":"synthetic-company-rollback","properties":{"first_name":"Retained employee","company":"Rejected employer","job":"Analyst"},"data":[]}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("failure was not injected after the actual company write")
	}
	if response.ImportedContacts != 1 || len(response.Errors) != 1 || response.Errors[0] != `company "Rejected employer": could not import company` {
		t.Fatalf("partial import or sanitized error changed: %+v", response)
	}
	var companyCount int64
	if err := svc.DB.Model(&models.Company{}).Where("vault_id = ?", vault).Count(&companyCount).Error; err != nil {
		t.Fatal(err)
	}
	if companyCount != 0 {
		t.Error("failed resolution left a company in the database")
	}
	var contacts []models.Contact
	if err := svc.DB.Where("vault_id = ?", vault).Find(&contacts).Error; err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected retained contact, got %d", len(contacts))
	}
	contact := contacts[0]
	if contact.CompanyID != nil || ptrToStr(contact.FirstName) != "Retained employee" || ptrToStr(contact.JobPosition) != "Analyst" {
		t.Errorf("retained contact lost unrelated data: company=%v title=%q", contact.CompanyID, ptrToStr(contact.JobPosition))
	}
	var jobCount int64
	if err := svc.DB.Model(&models.ContactCompany{}).Where("contact_id = ?", contact.ID).Count(&jobCount).Error; err != nil {
		t.Fatal(err)
	}
	if jobCount != 0 {
		t.Error("failed company write left an employment association")
	}
}
