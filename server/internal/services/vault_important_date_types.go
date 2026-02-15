package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultImportantDateTypeService struct {
	db *gorm.DB
}

func NewVaultImportantDateTypeService(db *gorm.DB) *VaultImportantDateTypeService {
	return &VaultImportantDateTypeService{db: db}
}

func (s *VaultImportantDateTypeService) List(vaultID string) ([]dto.ImportantDateTypeResponse, error) {
	var types []models.ContactImportantDateType
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at ASC").Find(&types).Error; err != nil {
		return nil, err
	}
	result := make([]dto.ImportantDateTypeResponse, len(types))
	for i, t := range types {
		result[i] = toImportantDateTypeResponse(&t)
	}
	return result, nil
}

func (s *VaultImportantDateTypeService) Create(vaultID string, req dto.CreateImportantDateTypeRequest) (*dto.ImportantDateTypeResponse, error) {
	dt := models.ContactImportantDateType{
		VaultID:      vaultID,
		Label:        req.Label,
		CanBeDeleted: true,
	}
	if err := s.db.Create(&dt).Error; err != nil {
		return nil, err
	}
	resp := toImportantDateTypeResponse(&dt)
	return &resp, nil
}

func (s *VaultImportantDateTypeService) Update(id uint, vaultID string, req dto.UpdateImportantDateTypeRequest) (*dto.ImportantDateTypeResponse, error) {
	var dt models.ContactImportantDateType
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&dt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDateTypeNotFound
		}
		return nil, err
	}
	dt.Label = req.Label
	if err := s.db.Save(&dt).Error; err != nil {
		return nil, err
	}
	resp := toImportantDateTypeResponse(&dt)
	return &resp, nil
}

func (s *VaultImportantDateTypeService) Delete(id uint, vaultID string) error {
	var dt models.ContactImportantDateType
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&dt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDateTypeNotFound
		}
		return err
	}
	if !dt.CanBeDeleted {
		return ErrCannotDeleteDefault
	}
	return s.db.Delete(&dt).Error
}

func toImportantDateTypeResponse(dt *models.ContactImportantDateType) dto.ImportantDateTypeResponse {
	return dto.ImportantDateTypeResponse{
		ID:           dt.ID,
		Label:        dt.Label,
		InternalType: ptrToStr(dt.InternalType),
		CanBeDeleted: dt.CanBeDeleted,
		CreatedAt:    dt.CreatedAt,
		UpdatedAt:    dt.UpdatedAt,
	}
}
