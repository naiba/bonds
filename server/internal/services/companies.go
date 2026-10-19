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
	db *gorm.DB
}

func NewCompanyService(db *gorm.DB) *CompanyService {
	return &CompanyService{db: db}
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

func (s *CompanyService) Update(id uint, vaultID string, req dto.UpdateCompanyRequest) (*dto.CompanyResponse, error) {
	var company models.Company
	err := companyEmploymentTransaction(s.db, func(tx *gorm.DB) error {
		// Rename changes every employee's vCard. Lock the same parents as deletion,
		// then read the company and commit its name together with local-wins markers.
		contactIDs, err := lockCompanyEmploymentContacts(tx, id, vaultID)
		if err != nil {
			return err
		}
		if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		renamed := company.Name != req.Name
		company.Name = req.Name
		company.Type = strPtrOrNil(req.Type)
		if err := tx.Save(&company).Error; err != nil {
			return err
		}
		if renamed {
			for _, contactID := range contactIDs {
				if err := markLocalEmploymentChanged(tx, contactID, vaultID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp := toCompanyResponse(&company)
	return &resp, nil
}

// Delete removes a company and cleans up related ContactCompany rows and legacy Contact.CompanyID references.
func (s *CompanyService) Delete(id uint, vaultID string) error {
	return companyEmploymentTransaction(s.db, func(tx *gorm.DB) error {
		var company models.Company
		if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		// Company deletion is also a local employment edit. Discover both normal
		// jobs and legacy-only owners, then use the same parent lock order as Jobs
		// and merge. Cleanup and the local-wins marker must commit together.
		contactIDs, err := lockCompanyEmploymentContacts(tx, id, vaultID)
		if err != nil {
			return err
		}
		if err := tx.Where("company_id = ?", id).Delete(&models.ContactCompany{}).Error; err != nil {
			return err
		}
		for _, contactID := range contactIDs {
			if err := clearLegacyJobSelection(tx, contactID, vaultID, id); err != nil {
				return err
			}
			if err := markLocalEmploymentChanged(tx, contactID, vaultID); err != nil {
				return err
			}
		}
		return tx.Delete(&company).Error
	})
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

// Company rename and deletion affect both normal jobs and legacy-only owners.
// Keep the parent lock order shared with Jobs and contact merge operations.
func companyEmploymentContactIDs(tx *gorm.DB, companyID uint, vaultID string) ([]string, error) {
	var ids []string
	employees := tx.Model(&models.ContactCompany{}).Select("contact_id").Where("company_id = ?", companyID)
	err := tx.Model(&models.Contact{}).Where("vault_id = ?", vaultID).Where("company_id = ? OR id IN (?)", companyID, employees).Pluck("id", &ids).Error
	return ids, err
}

var errCompanyEmployeesChanged = errors.New("company employee set changed while locking")

// Discovery is not a stable employee set. Lock parents first, then the company
// (the same order as association writers), and recheck while new links are
// blocked. If new parents appeared, release everything before acquiring them;
// acquiring more parent locks while holding the company would invert the order.
func lockCompanyEmploymentContacts(tx *gorm.DB, companyID uint, vaultID string) ([]string, error) {
	ids, err := companyEmploymentContactIDs(tx, companyID, vaultID)
	if err != nil {
		return nil, err
	}
	if err := lockContactsBelongToVault(tx, ids, vaultID); err != nil {
		return nil, err
	}
	var company models.Company
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).First(&company, "id = ? AND vault_id = ?", companyID, vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCompanyNotFound
		}
		return nil, err
	}
	actual, err := companyEmploymentContactIDs(tx, companyID, vaultID)
	if err != nil {
		return nil, err
	}
	locked := map[string]bool{}
	for _, id := range ids {
		locked[id] = true
	}
	for _, id := range actual {
		if !locked[id] {
			return nil, errCompanyEmployeesChanged
		}
	}
	return actual, nil
}

func companyEmploymentTransaction(db *gorm.DB, change func(*gorm.DB) error) error {
	for {
		if err := db.Statement.Context.Err(); err != nil {
			return err
		}
		err := db.Transaction(change)
		// Only this read-only discovery race is retried; all other errors and all
		// mutation failures are returned. The failed attempt has already rolled back.
		if !errors.Is(err, errCompanyEmployeesChanged) {
			return err
		}
	}
}

// Call inside the contact transaction, after its parent lock. KEY SHARE permits
// independent employee writes while excluding employer deletion/key changes.
func lockCompanyForEmployment(tx *gorm.DB, companyID uint, vaultID string) error {
	var company models.Company
	if err := tx.Clauses(clause.Locking{Strength: "KEY SHARE"}).First(&company, "id = ? AND vault_id = ?", companyID, vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCompanyNotFound
		}
		return err
	}
	return nil
}
