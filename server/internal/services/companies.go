package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCompanyNotFound = errors.New("company not found")

type CompanyService struct {
	db             *gorm.DB
	davPushService *DavPushService
}

func NewCompanyService(db *gorm.DB) *CompanyService {
	return &CompanyService{db: db}
}

func (s *CompanyService) SetDavPushService(push *DavPushService) {
	s.davPushService = push
}

func (s *CompanyService) List(vaultID, userID string) ([]dto.CompanyResponse, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	var companies []models.Company
	// BUG FIX (#55): Must Preload ContactCompanies+Contact so the Employees column
	// in the frontend list is populated, not just in the single-Get detail view.
	// GORM retains join rows but omits soft-deleted contacts; filter them before employee formatting.
	if err := s.db.Preload("ContactCompanies", func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN contacts ON contacts.id = contact_companies.contact_id AND contacts.deleted_at IS NULL")
	}).Preload("ContactCompanies.Contact").Where("vault_id = ?", vaultID).Order("name ASC").Find(&companies).Error; err != nil {
		return nil, err
	}
	result := make([]dto.CompanyResponse, len(companies))
	for i, c := range companies {
		resp, err := toCompanyResponseWithEmployees(&c, formatter)
		if err != nil {
			return nil, err
		}
		result[i] = resp
	}
	return result, nil
}

// ListForContact queries the ContactCompany join table to find all companies for a contact.
// Replaces the old approach of reading Contact.CompanyID directly.
func (s *CompanyService) ListForContact(contactID, vaultID string) ([]dto.CompanyResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []dto.CompanyResponse{}, nil
		}
		return nil, err
	}

	var jobs []models.ContactCompany
	if err := s.db.Preload("Company").Where("contact_id = ?", contactID).Find(&jobs).Error; err != nil {
		return nil, err
	}

	result := make([]dto.CompanyResponse, 0, len(jobs))
	for _, j := range jobs {
		if j.Company.ID != 0 {
			result = append(result, toCompanyResponse(&j.Company))
		}
	}
	return result, nil
}

// Get returns a single company with its employees loaded via the ContactCompany join table.
// SECURITY: WHERE vault_id 防止跨 vault IDOR，勿删。
func (s *CompanyService) Get(id uint, vaultID, userID string) (*dto.CompanyResponse, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	var company models.Company
	// GORM retains join rows but omits soft-deleted contacts; filter them before employee formatting.
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).
		Preload("ContactCompanies", func(db *gorm.DB) *gorm.DB {
			return db.Joins("JOIN contacts ON contacts.id = contact_companies.contact_id AND contacts.deleted_at IS NULL")
		}).Preload("ContactCompanies.Contact").
		First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCompanyNotFound
		}
		return nil, err
	}
	resp, err := toCompanyResponseWithEmployees(&company, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CompanyService) Create(vaultID string, req dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
	company := models.Company{
		VaultID: vaultID,
		Name:    req.Name,
		Type:    strPtrOrNil(req.Type),
	}
	if err := s.db.Create(&company).Error; err != nil {
		return nil, err
	}
	resp := toCompanyResponse(&company)
	return &resp, nil
}

// companyContactIDs includes both job history and legacy-only contacts. Selecting
// from contacts deduplicates employees with multiple jobs at the same company.
func companyContactIDs(db *gorm.DB, companyID uint, vaultID string, lockContacts bool) ([]string, error) {
	var contacts []struct {
		ID        string
		DeletedAt gorm.DeletedAt
	}
	jobs := db.Model(&models.ContactCompany{}).Select("contact_id").Where("company_id = ?", companyID)
	query := db.Model(&models.Contact{}).Where("vault_id = ?", vaultID).
		Where("company_id = ? OR id IN (?)", companyID, jobs).Order("id ASC")
	if lockContacts {
		// Tombstones still enforce their company FK. Lock them for cleanup,
		// but do not send notifications for contacts already deleted.
		query = query.Unscoped().Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
	}
	if err := query.Select("id", "deleted_at").Find(&contacts).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		if !contact.DeletedAt.Valid {
			ids = append(ids, contact.ID)
		}
	}
	return ids, nil
}

func (s *CompanyService) pushContactChanges(contactIDs []string, vaultID string) {
	if s.davPushService == nil || len(contactIDs) == 0 {
		return
	}
	// Capture IDs before deletion, but export fresh cards only after commit. One
	// worker per company change avoids spawning a goroutine for every employee.
	s.davPushService.PushContactChangesAsync(contactIDs, vaultID)
}

func (s *CompanyService) Update(id uint, vaultID string, req dto.UpdateCompanyRequest) (*dto.CompanyResponse, error) {
	var company models.Company
	var contactIDs []string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: companyWriteLockStrength}).Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		// Hold the company lock through recipient discovery and commit so a new
		// association cannot push the old name after being missed by this batch.
		// Only the name appears in vCards; changing company type needs no push.
		if s.davPushService != nil && company.Name != req.Name {
			var err error
			contactIDs, err = companyContactIDs(tx, id, vaultID, false)
			if err != nil {
				return err
			}
		}
		company.Name = req.Name
		company.Type = strPtrOrNil(req.Type)
		return tx.Save(&company).Error
	})
	if err != nil {
		return nil, err
	}
	s.pushContactChanges(contactIDs, vaultID)
	resp := toCompanyResponse(&company)
	return &resp, nil
}

// Delete removes a company and cleans up related ContactCompany rows and legacy Contact.CompanyID references.
func (s *CompanyService) Delete(id uint, vaultID string) error {
	var contactIDs []string
	// Cleanup must be atomic: otherwise a failed company delete loses jobs and
	// subscribers can observe a partially deleted organization.
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var company models.Company
		if err := tx.Clauses(clause.Locking{Strength: companyWriteLockStrength}).Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		// Lock company -> contacts (in ID order) -> jobs, matching association
		// writers and contact cleanup. Collecting recipients without the company
		// lock can miss an employee that pushes before this deletion commits.
		var err error
		contactIDs, err = companyContactIDs(tx, id, vaultID, true)
		if err != nil {
			return err
		}
		if err := tx.Where("company_id = ?", id).Delete(&models.ContactCompany{}).Error; err != nil {
			return err
		}
		// Soft-deleted merge sources retain their profile but must not keep a
		// foreign key to the company being removed.
		if err := tx.Unscoped().Model(&models.Contact{}).Where("company_id = ?", id).Update("company_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&company).Error
	})
	if err != nil {
		return err
	}
	s.pushContactChanges(contactIDs, vaultID)
	return nil
}

func toCompanyResponse(c *models.Company) dto.CompanyResponse {
	return dto.CompanyResponse{
		ID:        c.ID,
		VaultID:   c.VaultID,
		Name:      c.Name,
		Type:      ptrToStr(c.Type),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// toCompanyResponseWithEmployees builds CompanyResponse with employee briefs from the ContactCompany join table.
func toCompanyResponseWithEmployees(c *models.Company, formatter *contactNameFormatter) (dto.CompanyResponse, error) {
	resp := toCompanyResponse(c)
	contacts := make([]dto.CompanyContactBrief, len(c.ContactCompanies))
	for i, cc := range c.ContactCompanies {
		name, err := formatter.format(&cc.Contact, "")
		if err != nil {
			return dto.CompanyResponse{}, err
		}
		contacts[i] = dto.CompanyContactBrief{
			ID:          cc.Contact.ID,
			Name:        name,
			FirstName:   ptrToStr(cc.Contact.FirstName),
			LastName:    ptrToStr(cc.Contact.LastName),
			JobPosition: ptrToStr(cc.JobPosition),
			JobID:       cc.ID,
		}
	}
	resp.Contacts = contacts
	return resp, nil
}
