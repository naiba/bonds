package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrCompanyNotFound = errors.New("company not found")

type CompanyService struct {
	db *gorm.DB
}

func NewCompanyService(db *gorm.DB) *CompanyService {
	return &CompanyService{db: db}
}

func (s *CompanyService) List(vaultID string) ([]dto.CompanyResponse, error) {
	var companies []models.Company
	if err := s.db.Where("vault_id = ?", vaultID).Order("name ASC").Find(&companies).Error; err != nil {
		return nil, err
	}
	result := make([]dto.CompanyResponse, len(companies))
	for i, c := range companies {
		result[i] = toCompanyResponse(&c)
	}
	return result, nil
}


func (s *CompanyService) ListForContact(contactID, vaultID string) ([]dto.CompanyResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []dto.CompanyResponse{}, nil
		}
		return nil, err
	}
	if contact.CompanyID == nil {
		return []dto.CompanyResponse{}, nil
	}
	var company models.Company
	if err := s.db.First(&company, *contact.CompanyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []dto.CompanyResponse{}, nil
		}
		return nil, err
	}
	return []dto.CompanyResponse{toCompanyResponse(&company)}, nil
}

func (s *CompanyService) Get(id uint) (*dto.CompanyResponse, error) {
	var company models.Company
	if err := s.db.Preload("Contacts").First(&company, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCompanyNotFound
		}
		return nil, err
	}
	resp := toCompanyResponseWithContacts(&company)
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
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCompanyNotFound
		}
		return nil, err
	}
	company.Name = req.Name
	company.Type = strPtrOrNil(req.Type)
	if err := s.db.Save(&company).Error; err != nil {
		return nil, err
	}
	resp := toCompanyResponse(&company)
	return &resp, nil
}

func (s *CompanyService) Delete(id uint, vaultID string) error {
	var company models.Company
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCompanyNotFound
		}
		return err
	}
	// Null out CompanyID on associated contacts
	if err := s.db.Model(&models.Contact{}).Where("company_id = ?", id).Update("company_id", nil).Error; err != nil {
		return err
	}
	if err := s.db.Delete(&company).Error; err != nil {
		return err
	}
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

func toCompanyResponseWithContacts(c *models.Company) dto.CompanyResponse {
	resp := toCompanyResponse(c)
	contacts := make([]dto.CompanyContactBrief, len(c.Contacts))
	for i, ct := range c.Contacts {
		contacts[i] = dto.CompanyContactBrief{
			ID:        ct.ID,
			FirstName: ptrToStr(ct.FirstName),
			LastName:  ptrToStr(ct.LastName),
		}
	}
	resp.Contacts = contacts
	return resp
}
