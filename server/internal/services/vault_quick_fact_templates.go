package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultQuickFactTemplateService struct {
	db *gorm.DB
}

func NewVaultQuickFactTemplateService(db *gorm.DB) *VaultQuickFactTemplateService {
	return &VaultQuickFactTemplateService{db: db}
}

func (s *VaultQuickFactTemplateService) List(vaultID string) ([]dto.QuickFactTemplateResponse, error) {
	var tpls []models.VaultQuickFactsTemplate
	if err := s.db.Where("vault_id = ?", vaultID).Order("position ASC").Find(&tpls).Error; err != nil {
		return nil, err
	}
	result := make([]dto.QuickFactTemplateResponse, len(tpls))
	for i, t := range tpls {
		result[i] = toQuickFactTemplateResponse(&t)
	}
	return result, nil
}

func (s *VaultQuickFactTemplateService) Create(vaultID string, req dto.CreateQuickFactTemplateRequest) (*dto.QuickFactTemplateResponse, error) {
	label := req.Label
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	tpl := models.VaultQuickFactsTemplate{
		VaultID:  vaultID,
		Label:    &label,
		Position: position,
	}
	if err := s.db.Create(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) Update(id uint, vaultID string, req dto.UpdateQuickFactTemplateRequest) (*dto.QuickFactTemplateResponse, error) {
	var tpl models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactTplNotFound
		}
		return nil, err
	}
	label := req.Label
	tpl.Label = &label
	if req.Position != nil {
		tpl.Position = *req.Position
	}
	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) UpdatePosition(id uint, vaultID string, position int) (*dto.QuickFactTemplateResponse, error) {
	var tpl models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactTplNotFound
		}
		return nil, err
	}
	tpl.Position = position
	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) Delete(id uint, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.VaultQuickFactsTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrQuickFactTplNotFound
	}
	return nil
}

func toQuickFactTemplateResponse(t *models.VaultQuickFactsTemplate) dto.QuickFactTemplateResponse {
	return dto.QuickFactTemplateResponse{
		ID:        t.ID,
		Label:     ptrToStr(t.Label),
		Position:  t.Position,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
