package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type contactJobTestContext struct {
	svc       *ContactJobService
	db        *gorm.DB
	contactID string
	vaultID   string
	companyID uint
}

func setupContactJobTest(t *testing.T) *contactJobTestContext {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "job-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	contactSvc := NewContactService(db)
	contact, err := contactSvc.CreateContact(vault.ID, resp.User.ID, dto.CreateContactRequest{FirstName: "John"})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	// Create a company dynamically — do NOT reference seed CompanyID directly
	company := &models.Company{
		VaultID: vault.ID,
		Name:    "Test Corp",
	}
	if err := db.Create(company).Error; err != nil {
		t.Fatalf("Create company failed: %v", err)
	}

	return &contactJobTestContext{
		svc:       NewContactJobService(db),
		db:        db,
		contactID: contact.ID,
		vaultID:   vault.ID,
		companyID: company.ID,
	}
}

func TestContactJob_List_Empty(t *testing.T) {
	ctx := setupContactJobTest(t)

	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs, got %d", len(jobs))
	}
}

func TestContactJob_Create(t *testing.T) {
	ctx := setupContactJobTest(t)

	job, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if job.CompanyID != ctx.companyID {
		t.Errorf("Expected company ID %d, got %d", ctx.companyID, job.CompanyID)
	}
	if job.JobPosition != "Engineer" {
		t.Errorf("Expected job position 'Engineer', got '%s'", job.JobPosition)
	}
	if job.CompanyName != "Test Corp" {
		t.Errorf("Expected company name 'Test Corp', got '%s'", job.CompanyName)
	}
	if job.ContactID != ctx.contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", ctx.contactID, job.ContactID)
	}
	if job.ID == 0 {
		t.Error("Expected non-zero job ID")
	}
}

func TestContactJob_Create_MultipleJobs(t *testing.T) {
	ctx := setupContactJobTest(t)

	// Create second company
	company2 := &models.Company{VaultID: ctx.vaultID, Name: "Other Corp"}
	if err := ctx.db.Create(company2).Error; err != nil {
		t.Fatalf("Create company2 failed: %v", err)
	}

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Create job1 failed: %v", err)
	}

	_, err = ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   company2.ID,
		JobPosition: "Manager",
	})
	if err != nil {
		t.Fatalf("Create job2 failed: %v", err)
	}

	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("Expected 2 jobs, got %d", len(jobs))
	}
}

func TestContactJob_Create_ContactNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.Create("nonexistent-id", ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJob_Create_CompanyNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   99999,
		JobPosition: "Engineer",
	})
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestContactJob_Update(t *testing.T) {
	ctx := setupContactJobTest(t)

	job, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := ctx.svc.Update(ctx.contactID, ctx.vaultID, job.ID, dto.UpdateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Senior Engineer",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.JobPosition != "Senior Engineer" {
		t.Errorf("Expected job position 'Senior Engineer', got '%s'", updated.JobPosition)
	}
}

func TestContactJob_Update_NotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.Update(ctx.contactID, ctx.vaultID, 99999, dto.UpdateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != ErrContactJobNotFound {
		t.Errorf("Expected ErrContactJobNotFound, got %v", err)
	}
}

func TestContactJob_Update_ContactNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.Update("nonexistent-id", ctx.vaultID, 1, dto.UpdateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJob_Delete(t *testing.T) {
	ctx := setupContactJobTest(t)

	job, err := ctx.svc.Create(ctx.contactID, ctx.vaultID, dto.CreateContactJobRequest{
		CompanyID:   ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = ctx.svc.Delete(ctx.contactID, ctx.vaultID, job.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after delete, got %d", len(jobs))
	}
}

func TestContactJob_Delete_NotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	err := ctx.svc.Delete(ctx.contactID, ctx.vaultID, 99999)
	if err != ErrContactJobNotFound {
		t.Errorf("Expected ErrContactJobNotFound, got %v", err)
	}
}

func TestContactJob_Delete_ContactNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	err := ctx.svc.Delete("nonexistent-id", ctx.vaultID, 1)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJob_AddEmployee(t *testing.T) {
	ctx := setupContactJobTest(t)

	job, err := ctx.svc.AddEmployee(ctx.companyID, ctx.vaultID, dto.AddEmployeeRequest{
		ContactID:   ctx.contactID,
		JobPosition: "CTO",
	})
	if err != nil {
		t.Fatalf("AddEmployee failed: %v", err)
	}
	if job.CompanyID != ctx.companyID {
		t.Errorf("Expected company ID %d, got %d", ctx.companyID, job.CompanyID)
	}
	if job.ContactID != ctx.contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", ctx.contactID, job.ContactID)
	}
	if job.JobPosition != "CTO" {
		t.Errorf("Expected job position 'CTO', got '%s'", job.JobPosition)
	}
}

func TestContactJob_AddEmployee_CompanyNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.AddEmployee(99999, ctx.vaultID, dto.AddEmployeeRequest{
		ContactID: ctx.contactID,
	})
	if err != ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}
}

func TestContactJob_AddEmployee_ContactNotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.AddEmployee(ctx.companyID, ctx.vaultID, dto.AddEmployeeRequest{
		ContactID: "nonexistent-id",
	})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJob_RemoveEmployee(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.AddEmployee(ctx.companyID, ctx.vaultID, dto.AddEmployeeRequest{
		ContactID:   ctx.contactID,
		JobPosition: "CTO",
	})
	if err != nil {
		t.Fatalf("AddEmployee failed: %v", err)
	}

	err = ctx.svc.RemoveEmployee(ctx.companyID, ctx.vaultID, ctx.contactID)
	if err != nil {
		t.Fatalf("RemoveEmployee failed: %v", err)
	}

	// Verify no jobs remain
	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after remove, got %d", len(jobs))
	}
}

func TestContactJob_RemoveEmployee_NotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	err := ctx.svc.RemoveEmployee(ctx.companyID, ctx.vaultID, ctx.contactID)
	if err != ErrContactJobNotFound {
		t.Errorf("Expected ErrContactJobNotFound, got %v", err)
	}
}

func TestContactJob_LegacyUpdate(t *testing.T) {
	ctx := setupContactJobTest(t)

	resp, err := ctx.svc.LegacyUpdate(ctx.contactID, ctx.vaultID, dto.UpdateJobInfoRequest{
		CompanyID:   &ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("LegacyUpdate failed: %v", err)
	}
	if resp.ID != ctx.contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", ctx.contactID, resp.ID)
	}

	// Verify a ContactCompany row was created
	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job after legacy update, got %d", len(jobs))
	}
	if jobs[0].CompanyID != ctx.companyID {
		t.Errorf("Expected company ID %d, got %d", ctx.companyID, jobs[0].CompanyID)
	}
}

func TestContactJob_LegacyUpdate_NotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.LegacyUpdate("nonexistent-id", ctx.vaultID, dto.UpdateJobInfoRequest{
		JobPosition: "Engineer",
	})
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}

func TestContactJob_LegacyDelete(t *testing.T) {
	ctx := setupContactJobTest(t)

	// First create a job via legacy
	_, err := ctx.svc.LegacyUpdate(ctx.contactID, ctx.vaultID, dto.UpdateJobInfoRequest{
		CompanyID:   &ctx.companyID,
		JobPosition: "Engineer",
	})
	if err != nil {
		t.Fatalf("LegacyUpdate failed: %v", err)
	}

	resp, err := ctx.svc.LegacyDelete(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("LegacyDelete failed: %v", err)
	}
	if resp.ID != ctx.contactID {
		t.Errorf("Expected contact ID '%s', got '%s'", ctx.contactID, resp.ID)
	}

	// Verify all ContactCompany rows deleted
	jobs, err := ctx.svc.List(ctx.contactID, ctx.vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after legacy delete, got %d", len(jobs))
	}
}

func TestContactJob_LegacyDelete_NotFound(t *testing.T) {
	ctx := setupContactJobTest(t)

	_, err := ctx.svc.LegacyDelete("nonexistent-id", ctx.vaultID)
	if err != ErrContactNotFound {
		t.Errorf("Expected ErrContactNotFound, got %v", err)
	}
}
