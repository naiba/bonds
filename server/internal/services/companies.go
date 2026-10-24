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
	if err := companyEmploymentTransaction(s.db, vaultID, func(tx *gorm.DB) error {
		if err := lockCompanyResolution(tx, vaultID); err != nil {
			return err
		}
		return tx.Create(&company).Error
	}); err != nil {
		return nil, err
	}
	resp := toCompanyResponse(&company)
	return &resp, nil
}

// lockCompanyContacts locks the owners of every physical company reference,
// including historical cross-vault links and tombstones. Cleanup by company ID
// touches all of them; filtering locks by vault would invert contact -> job
// ordering against a concurrent legacy/job deletion. Notification recipients
// remain restricted to live contacts in the company's vault.
func lockCompanyContacts(db *gorm.DB, companyID uint, vaultID string) ([]string, error) {
	var contacts []struct {
		ID        string
		VaultID   string
		DeletedAt gorm.DeletedAt
	}
	if err := companyContactQuery(db, companyID).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Select("id", "vault_id", "deleted_at").Find(&contacts).Error; err != nil {
		return nil, err
	}
	if err := tryCompanyResolution(db, vaultID); err != nil {
		return nil, err
	}
	// NO KEY UPDATE permits a resolver's foreign-key KEY SHARE lock. A pull
	// can commit a new owner after discovery and release its name lock before
	// our try succeeds (#297). Revalidate under resolution exclusion; never
	// append contact locks here, which could invert the global ID lock order.
	// Roll back the entire transaction and rediscover if membership changed.
	var currentIDs []string
	if err := companyContactQuery(db, companyID).Pluck("id", &currentIDs).Error; err != nil {
		return nil, err
	}
	if len(currentIDs) != len(contacts) {
		return nil, errCompanyContactsChanged
	}
	for i, id := range currentIDs {
		if id != contacts[i].ID {
			return nil, errCompanyContactsChanged
		}
	}
	ids := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		if contact.VaultID == vaultID && !contact.DeletedAt.Valid {
			ids = append(ids, contact.ID)
		}
	}
	return ids, nil
}

func companyContactQuery(db *gorm.DB, companyID uint) *gorm.DB {
	jobs := db.Model(&models.ContactCompany{}).Select("contact_id").Where("company_id = ?", companyID)
	return db.Unscoped().Model(&models.Contact{}).
		Where("company_id = ? OR id IN (?)", companyID, jobs).Order("id ASC")
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
	err := companyEmploymentTransaction(s.db, vaultID, func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: companyWriteLockStrength}).Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		// Only the name appears in vCards; changing company type needs no push.
		// Reset recipients on retry in case another writer already renamed it.
		contactIDs = nil
		if company.Name != req.Name {
			var err error
			contactIDs, err = lockCompanyContacts(tx, id, vaultID)
			if err != nil {
				return err
			}
		} else if err := tryCompanyResolution(tx, vaultID); err != nil {
			return err
		}
		company.Name = req.Name
		company.Type = strPtrOrNil(req.Type)
		if err := tx.Save(&company).Error; err != nil {
			return err
		}
		return markContactJobsChanged(tx, contactIDs, vaultID)
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
	err := companyEmploymentTransaction(s.db, vaultID, func(tx *gorm.DB) error {
		var company models.Company
		if err := tx.Clauses(clause.Locking{Strength: companyWriteLockStrength}).Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCompanyNotFound
			}
			return err
		}
		// Lock company -> contacts (in ID order) -> jobs, matching association
		// writers and contact cleanup. The helper also freezes name resolution
		// and revalidates recipients before any destructive write.
		var err error
		contactIDs, err = lockCompanyContacts(tx, id, vaultID)
		if err != nil {
			return err
		}
		var removedJobIDs []uint
		if err := tx.Model(&models.ContactCompany{}).Where("company_id = ?", id).Pluck("id", &removedJobIDs).Error; err != nil {
			return err
		}
		if err := tx.Where("company_id = ?", id).Delete(&models.ContactCompany{}).Error; err != nil {
			return err
		}
		for _, contactID := range contactIDs {
			if err := clearLegacyJobSelection(tx, contactID, vaultID, id); err != nil {
				return err
			}
		}
		// A removed job cannot remain selected, including historical foreign or
		// tombstone owners. Clear only references to the jobs removed here.
		if len(removedJobIDs) > 0 {
			if err := tx.Unscoped().Model(&models.Contact{}).Where("preferred_job_id IN ?", removedJobIDs).Update("preferred_job_id", nil).Error; err != nil {
				return err
			}
		}
		// Soft-deleted merge sources retain their profile but must not keep a
		// foreign key to the company being removed.
		if err := tx.Unscoped().Model(&models.Contact{}).Where("company_id = ?", id).Update("company_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Delete(&company).Error; err != nil {
			return err
		}
		return markContactJobsChanged(tx, contactIDs, vaultID)
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

var errCompanyResolutionBusy = errors.New("company candidates are being resolved")
var errCompanyContactsChanged = errors.New("company references changed during discovery")

// Company names are not unique identities. Serialize candidate membership
// (create/rename/delete) with automatic resolution without adding a constraint.
// Call within the writer's transaction. SQLite already serializes writes.
func lockCompanyResolution(tx *gorm.DB, vaultID string) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "bonds:vcard-company:"+vaultID).Error
}

// A resolver holds a contact and the vault name lock before taking a compatible
// company KEY SHARE lock. Never wait for its name lock while holding company or
// contact locks: release this transaction first, then rediscover the owners.
func tryCompanyResolution(tx *gorm.DB, vaultID string) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	var acquired bool
	if err := tx.Raw("SELECT pg_try_advisory_xact_lock(hashtextextended(?, 0))", "bonds:vcard-company:"+vaultID).Scan(&acquired).Error; err != nil {
		return err
	}
	if !acquired {
		return errCompanyResolutionBusy
	}
	return nil
}

func companyEmploymentTransaction(db *gorm.DB, vaultID string, change func(*gorm.DB) error) error {
	for {
		if err := db.Statement.Context.Err(); err != nil {
			return err
		}
		err := db.Transaction(change)
		if errors.Is(err, errCompanyContactsChanged) {
			continue
		}
		if !errors.Is(err, errCompanyResolutionBusy) {
			return err
		}
		if err := db.Transaction(func(tx *gorm.DB) error { return lockCompanyResolution(tx, vaultID) }); err != nil {
			return err
		}
	}
}
