package services

import (
	"errors"

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
	var count int64
	if err := s.db.Model(&models.Company{}).Where("id = ? AND vault_id = ?", companyID, vaultID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrCompanyNotFound
	}
	return nil
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
func (s *ContactJobService) Create(contactID, vaultID string, req dto.CreateContactJobRequest) (*dto.ContactJobResponse, error) {
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
	if err := s.db.Create(&job).Error; err != nil {
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

	job.CompanyID = req.CompanyID
	job.JobPosition = strPtrOrNil(req.JobPosition)
	if err := s.db.Save(&job).Error; err != nil {
		return nil, err
	}

	if err := s.db.Preload("Company").First(&job, job.ID).Error; err != nil {
		return nil, err
	}

	resp := toContactJobResponse(&job)
	return &resp, nil
}

// Delete removes a specific job entry by its ID.
func (s *ContactJobService) Delete(contactID, vaultID string, jobID uint) error {
	if err := s.verifyContactInVault(contactID, vaultID); err != nil {
		return err
	}

	result := s.db.Where("id = ? AND contact_id = ?", jobID, contactID).Delete(&models.ContactCompany{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactJobNotFound
	}
	return nil
}

// AddEmployee creates a ContactCompany entry from the company side (VaultCompanies page).
func (s *ContactJobService) AddEmployee(companyID uint, vaultID string, req dto.AddEmployeeRequest) (*dto.ContactJobResponse, error) {
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
	if err := s.db.Create(&job).Error; err != nil {
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
	// Verify company belongs to vault
	if err := s.verifyCompanyInVault(companyID, vaultID); err != nil {
		return err
	}

	result := s.db.Where("company_id = ? AND contact_id = ?", companyID, contactID).Delete(&models.ContactCompany{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactJobNotFound
	}
	return nil
}

// LegacyUpdate creates or updates the first job for backward compatibility with the old PUT /jobInformation endpoint.
// Uses the new ContactCompany table instead of Contact.CompanyID.
func (s *ContactJobService) LegacyUpdate(contactID, vaultID string, req dto.UpdateJobInfoRequest) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	// Find existing first job or create new one
	var existingJob models.ContactCompany
	found := true
	if err := s.db.Where("contact_id = ?", contactID).Order("id ASC").First(&existingJob).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			found = false
		} else {
			return nil, err
		}
	}

	if req.CompanyID != nil {
		if found {
			existingJob.CompanyID = *req.CompanyID
			existingJob.JobPosition = strPtrOrNil(req.JobPosition)
			if err := s.db.Save(&existingJob).Error; err != nil {
				return nil, err
			}
		} else {
			newJob := models.ContactCompany{
				ContactID:   contactID,
				CompanyID:   *req.CompanyID,
				JobPosition: strPtrOrNil(req.JobPosition),
			}
			if err := s.db.Create(&newJob).Error; err != nil {
				return nil, err
			}
		}
	}

	// Also update legacy fields on Contact for backward compatibility
	contact.CompanyID = req.CompanyID
	contact.JobPosition = strPtrOrNil(req.JobPosition)
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

// LegacyDelete clears all jobs for a contact for backward compatibility with the old DELETE /jobInformation endpoint.
func (s *ContactJobService) LegacyDelete(contactID, vaultID string) (*dto.ContactResponse, error) {
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
	contact.CompanyID = nil
	contact.JobPosition = nil
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
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
