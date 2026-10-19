package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrContactJobNotFound = errors.New("contact job not found")

type ContactJobService struct {
	db *gorm.DB
}

func NewContactJobService(db *gorm.DB) *ContactJobService {
	return &ContactJobService{db: db}
}

// verifyContactInVault checks that a contact exists and belongs to the given vault.
func (s *ContactJobService) verifyContactInVault(contactID, vaultID string) error {
	var count int64
	if err := s.db.Model(&models.Contact{}).Where("id = ? AND vault_id = ?", contactID, vaultID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrContactNotFound
	}
	return nil
}

// verifyCompanyInVault checks that a company exists and belongs to the given vault.
func (s *ContactJobService) verifyCompanyInVault(companyID uint, vaultID string) error {
	return lockCompanyForEmployment(s.db, companyID, vaultID)
}

// List returns all jobs (ContactCompany rows) for a contact.
func (s *ContactJobService) List(contactID, vaultID string) ([]dto.ContactJobResponse, error) {
	if err := s.verifyContactInVault(contactID, vaultID); err != nil {
		return nil, err
	}

	var jobs []models.ContactCompany
	if err := s.db.Preload("Company").Where("contact_id = ?", contactID).Find(&jobs).Error; err != nil {
		return nil, err
	}

	result := make([]dto.ContactJobResponse, len(jobs))
	for i, j := range jobs {
		result[i] = toContactJobResponse(&j)
	}
	return result, nil
}

// Create adds a new job entry for a contact via the ContactCompany join table.
func (s *ContactJobService) create(contactID, vaultID string, req dto.CreateContactJobRequest) (*dto.ContactJobResponse, error) {
	if err := s.verifyContactInVault(contactID, vaultID); err != nil {
		return nil, err
	}
	if err := s.verifyCompanyInVault(req.CompanyID, vaultID); err != nil {
		return nil, err
	}

	job := models.ContactCompany{
		ContactID:   contactID,
		CompanyID:   req.CompanyID,
		JobPosition: strPtrOrNil(req.JobPosition),
	}
	if err := createContactRecord(s.db, &job, contactID, vaultID); err != nil {
		return nil, err
	}

	// Reload with Company preloaded for response
	if err := s.db.Preload("Company").First(&job, job.ID).Error; err != nil {
		return nil, err
	}

	resp := toContactJobResponse(&job)
	return &resp, nil
}

// Update modifies an existing job entry by its ID.
func (s *ContactJobService) Update(contactID, vaultID string, jobID uint, req dto.UpdateContactJobRequest) (*dto.ContactJobResponse, error) {
	var result *dto.ContactJobResponse
	err := s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		var err error
		result, err = NewContactJobService(tx).update(contactID, vaultID, jobID, req)
		return err
	})
	return result, err
}

func (s *ContactJobService) update(contactID, vaultID string, jobID uint, req dto.UpdateContactJobRequest) (*dto.ContactJobResponse, error) {
	if err := s.verifyContactInVault(contactID, vaultID); err != nil {
		return nil, err
	}

	var job models.ContactCompany
	if err := s.db.Where("id = ? AND contact_id = ?", jobID, contactID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactJobNotFound
		}
		return nil, err
	}

	if err := s.verifyCompanyInVault(req.CompanyID, vaultID); err != nil {
		return nil, err
	}

	selected, err := s.isSelectedJob(contactID, vaultID, jobID)
	if err != nil {
		return nil, err
	}
	oldCompanyID := job.CompanyID
	job.CompanyID = req.CompanyID
	job.JobPosition = strPtrOrNil(req.JobPosition)
	if err := updateContactRecord(s.db, &job, contactID, vaultID, ErrContactJobNotFound, "company_id", "job_position"); err != nil {
		return nil, err
	}

	// Several jobs can share a company. Only editing the selected job may
	// change its legacy selection; a secondary role must not replace it.
	if selected {
		if err := s.db.Model(&models.Contact{}).Where("id = ? AND vault_id = ? AND company_id = ?", contactID, vaultID, oldCompanyID).Updates(map[string]any{"company_id": job.CompanyID, "job_position": job.JobPosition}).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Preload("Company").First(&job, job.ID).Error; err != nil {
		return nil, err
	}

	resp := toContactJobResponse(&job)
	return &resp, nil
}

// Delete removes a specific job entry by its ID.
func (s *ContactJobService) Delete(contactID, vaultID string, jobID uint) error {
	return s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		var job models.ContactCompany
		if err := tx.Where("id = ? AND contact_id = ?", jobID, contactID).First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContactJobNotFound
			}
			return err
		}
		selected, err := NewContactJobService(tx).isSelectedJob(contactID, vaultID, jobID)
		if err != nil {
			return err
		}
		if err := tx.Delete(&job).Error; err != nil {
			return err
		}
		// Deleting another role at this company must preserve the selection too.
		if selected {
			return clearLegacyJobSelection(tx, contactID, vaultID, job.CompanyID)
		}
		return nil
	})
}

// AddEmployee creates a ContactCompany entry from the company side (VaultCompanies page).
func (s *ContactJobService) addEmployee(companyID uint, vaultID string, req dto.AddEmployeeRequest) (*dto.ContactJobResponse, error) {
	// Verify company belongs to vault
	if err := s.verifyCompanyInVault(companyID, vaultID); err != nil {
		return nil, err
	}
	// Verify contact belongs to same vault
	if err := s.verifyContactInVault(req.ContactID, vaultID); err != nil {
		return nil, err
	}

	job := models.ContactCompany{
		ContactID:   req.ContactID,
		CompanyID:   companyID,
		JobPosition: strPtrOrNil(req.JobPosition),
	}
	if err := createContactRecord(s.db, &job, req.ContactID, vaultID); err != nil {
		return nil, err
	}

	if err := s.db.Preload("Company").First(&job, job.ID).Error; err != nil {
		return nil, err
	}

	resp := toContactJobResponse(&job)
	return &resp, nil
}

// RemoveEmployee removes a ContactCompany entry by company and contact ID.
func (s *ContactJobService) RemoveEmployee(companyID uint, vaultID, contactID string) error {
	if err := s.verifyCompanyInVault(companyID, vaultID); err != nil {
		return err
	}
	return s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		result := tx.Where("company_id = ? AND contact_id = ?", companyID, contactID).Delete(&models.ContactCompany{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrContactJobNotFound
		}
		return clearLegacyJobSelection(tx, contactID, vaultID, companyID)
	})
}

// Normal jobs are authoritative. Clear their obsolete legacy selection in the
// same transaction so editing/deleting an imported job cannot resurrect it when
// the final job is removed and the vCard projection falls back to old fields.
func clearLegacyJobSelection(tx *gorm.DB, contactID, vaultID string, companyID uint) error {
	// Inspect the original selection before clearing deleted Job IDs: a standalone
	// title is independent of every company, including the last company removed.
	var contact models.Contact
	if err := tx.First(&contact, "id = ? AND vault_id = ?", contactID, vaultID).Error; err != nil {
		return err
	}
	if hasStandaloneEmployment(&contact) {
		return nil
	}
	// Clear deleted identities independently of legacy CompanyID (which can be
	// NULL). The subquery also prevents retaining another contact's job ID.
	remainingJobs := tx.Model(&models.ContactCompany{}).Select("id").Where("contact_id = ?", contactID)
	if err := tx.Model(&models.Contact{}).Where("id = ? AND vault_id = ? AND preferred_job_id NOT IN (?)", contactID, vaultID, remainingJobs).Update("preferred_job_id", nil).Error; err != nil {
		return err
	}
	var remaining int64
	if err := tx.Model(&models.ContactCompany{}).Where("contact_id = ?", contactID).Count(&remaining).Error; err != nil {
		return err
	}
	query := tx.Model(&models.Contact{}).Where("id = ? AND vault_id = ?", contactID, vaultID)
	// A title-only DAV edit may leave CompanyID nil. Removing the final normal
	// job must clear its fallback title regardless of that optional selection.
	if remaining > 0 {
		query = query.Where("company_id = ?", companyID)
	} else {
		// A different legacy company is an independent relationship, not a stale
		// mirror of this job. Preserve it when removing the last normal job.
		query = query.Where("company_id = ? OR company_id IS NULL", companyID)
	}
	return query.Updates(map[string]any{"company_id": nil, "job_position": nil}).Error
}

// LegacyUpdate creates or updates the first job for backward compatibility with the old PUT /jobInformation endpoint.
// Uses the new ContactCompany table instead of Contact.CompanyID.
func (s *ContactJobService) LegacyUpdate(contactID, vaultID, userID string, req dto.UpdateJobInfoRequest) (*dto.ContactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var result *dto.ContactResponse
	err := s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		var err error
		result, err = NewContactJobService(tx).legacyUpdate(contactID, vaultID, userID, req)
		return err
	})
	return result, err
}

func (s *ContactJobService) legacyUpdate(contactID, vaultID, userID string, req dto.UpdateJobInfoRequest) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	// Use the same selected employment as exports. The legacy endpoint is a
	// single-job editor; a missing company performs a title-only edit, not an
	// ignored profile write hidden by the authoritative normal Jobs records.
	loaded, err := loadContactEmployment(s.db, contactID, vaultID)
	if err != nil {
		return nil, err
	}
	var existingJob models.ContactCompany
	found := false
	projected := ContactVCardEmployments(loaded)
	if len(projected) > 0 && projected[0].JobID != 0 {
		for _, job := range loaded.ContactCompanies {
			if job.ID == projected[0].JobID {
				existingJob = job
				found = true
				break
			}
		}
	}
	if req.CompanyID == nil && found {
		id := existingJob.CompanyID
		req.CompanyID = &id
	}
	if req.CompanyID != nil {
		if err := s.verifyCompanyInVault(*req.CompanyID, vaultID); err != nil {
			return nil, err
		}
	}

	if req.CompanyID != nil {
		if found {
			existingJob.CompanyID = *req.CompanyID
			existingJob.JobPosition = strPtrOrNil(req.JobPosition)
			contact.PreferredJobID = &existingJob.ID
			if err := updateContactRecord(s.db, &existingJob, contactID, vaultID, ErrContactJobNotFound, "company_id", "job_position"); err != nil {
				return nil, err
			}
		} else {
			newJob := models.ContactCompany{
				ContactID:   contactID,
				CompanyID:   *req.CompanyID,
				JobPosition: strPtrOrNil(req.JobPosition),
			}
			if err := createContactRecord(s.db, &newJob, contactID, vaultID); err != nil {
				return nil, err
			}
			contact.PreferredJobID = &newJob.ID
		}
	}

	// Also update legacy fields on Contact for backward compatibility
	contact.CompanyID = req.CompanyID
	contact.JobPosition = strPtrOrNil(req.JobPosition)
	if err := updateContactProfile(s.db, &contact, vaultID, "company_id", "job_position", "preferred_job_id"); err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// LegacyDelete clears all jobs for a contact for backward compatibility with the old DELETE /jobInformation endpoint.
func (s *ContactJobService) legacyDelete(contactID, vaultID, userID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	// Delete all jobs from the join table
	if err := s.db.Where("contact_id = ?", contactID).Delete(&models.ContactCompany{}).Error; err != nil {
		return nil, err
	}

	// Also clear legacy fields on Contact
	contact.PreferredJobID = nil
	contact.CompanyID = nil
	contact.JobPosition = nil
	if err := updateContactProfile(s.db, &contact, vaultID, "company_id", "job_position", "preferred_job_id"); err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func toContactJobResponse(j *models.ContactCompany) dto.ContactJobResponse {
	companyName := ""
	if j.Company.ID != 0 {
		companyName = j.Company.Name
	}
	return dto.ContactJobResponse{
		ID:          j.ID,
		ContactID:   j.ContactID,
		CompanyID:   j.CompanyID,
		CompanyName: companyName,
		JobPosition: ptrToStr(j.JobPosition),
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}
}

// isSelectedJob uses the pre-mutation export order, not company ID alone.
// Call inside the contact transaction while holding the parent contact lock.
func (s *ContactJobService) isSelectedJob(contactID, vaultID string, jobID uint) (bool, error) {
	contact, err := loadContactEmployment(s.db, contactID, vaultID)
	if err != nil {
		return false, err
	}
	employments := ContactVCardEmployments(contact)
	return len(employments) > 0 && employments[0].JobID == jobID, nil
}

// Only local Jobs entry points advance the subscription conflict marker. Shared
// vCard reconciliation also runs for remote pulls and must never mark those as
// local edits. Commit the job and marker together under the parent contact lock.
func (s *ContactJobService) withLocalEmploymentChange(contactID, vaultID string, change func(*gorm.DB) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
			return err
		}
		// Freeze the pre-edit fallback for old rows before any mutable company or
		// title can change the ordering. Roll back this metadata with the edit.
		contact, err := loadContactEmployment(tx, contactID, vaultID)
		if err != nil {
			return err
		}
		projected := ContactVCardEmployments(contact)
		if len(projected) > 0 && projected[0].JobID != 0 {
			if err := tx.Model(&models.Contact{}).Where("id = ? AND vault_id = ?", contactID, vaultID).Update("preferred_job_id", projected[0].JobID).Error; err != nil {
				return err
			}
		}
		if err := change(tx); err != nil {
			return err
		}
		return markLocalEmploymentChanged(tx, contactID, vaultID)
	})
}

func (s *ContactJobService) Create(contactID, vaultID string, req dto.CreateContactJobRequest) (*dto.ContactJobResponse, error) {
	var result *dto.ContactJobResponse
	err := s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		var err error
		result, err = NewContactJobService(tx).create(contactID, vaultID, req)
		return err
	})
	return result, err
}

func (s *ContactJobService) AddEmployee(companyID uint, vaultID string, req dto.AddEmployeeRequest) (*dto.ContactJobResponse, error) {
	var result *dto.ContactJobResponse
	err := s.withLocalEmploymentChange(req.ContactID, vaultID, func(tx *gorm.DB) error {
		var err error
		result, err = NewContactJobService(tx).addEmployee(companyID, vaultID, req)
		return err
	})
	return result, err
}

func (s *ContactJobService) LegacyDelete(contactID, vaultID, userID string) (*dto.ContactResponse, error) {
	var result *dto.ContactResponse
	err := s.withLocalEmploymentChange(contactID, vaultID, func(tx *gorm.DB) error {
		var err error
		result, err = NewContactJobService(tx).legacyDelete(contactID, vaultID, userID)
		return err
	})
	return result, err
}

// Company-side deletion and contact-side edits share the same local-wins marker.
// Call only for local mutations, inside their parent-locked transaction.
func markLocalEmploymentChanged(tx *gorm.DB, contactID, vaultID string) error {
	now := time.Now()
	contact := models.Contact{ID: contactID, LastUpdatedAt: &now}
	return updateContactProfile(tx, &contact, vaultID, "last_updated_at")
}
