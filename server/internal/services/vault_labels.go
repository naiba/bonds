package services

import (
	"errors"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultLabelService struct {
	db *gorm.DB
}

func NewVaultLabelService(db *gorm.DB) *VaultLabelService {
	return &VaultLabelService{db: db}
}

func (s *VaultLabelService) List(vaultID string) ([]dto.LabelResponse, error) {
	var labels []models.Label
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&labels).Error; err != nil {
		return nil, err
	}
	result := make([]dto.LabelResponse, len(labels))
	for i, l := range labels {
		result[i] = toLabelResponse(&l)
	}
	return result, nil
}

func (s *VaultLabelService) Create(vaultID string, req dto.CreateLabelRequest) (*dto.LabelResponse, error) {
	label := models.Label{
		VaultID:     vaultID,
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: strPtrOrNil(req.Description),
		BgColor:     req.BgColor,
		TextColor:   req.TextColor,
	}
	if label.BgColor == "" {
		label.BgColor = "bg-zinc-200"
	}
	if label.TextColor == "" {
		label.TextColor = "text-zinc-700"
	}
	if err := s.db.Create(&label).Error; err != nil {
		return nil, err
	}
	resp := toLabelResponse(&label)
	return &resp, nil
}

func (s *VaultLabelService) Update(id uint, vaultID string, req dto.UpdateLabelRequest) (*dto.LabelResponse, error) {
	var label models.Label
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&label).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}
	label.Name = req.Name
	label.Slug = generateSlug(req.Name)
	label.Description = strPtrOrNil(req.Description)
	if req.BgColor != "" {
		label.BgColor = req.BgColor
	}
	if req.TextColor != "" {
		label.TextColor = req.TextColor
	}
	if err := s.db.Save(&label).Error; err != nil {
		return nil, err
	}
	resp := toLabelResponse(&label)
	return &resp, nil
}

func (s *VaultLabelService) Delete(id uint, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.Label{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLabelNotFound
	}
	return nil
}

func generateSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

func toLabelResponse(l *models.Label) dto.LabelResponse {
	return dto.LabelResponse{
		ID:          l.ID,
		Name:        l.Name,
		Slug:        l.Slug,
		Description: ptrToStr(l.Description),
		BgColor:     l.BgColor,
		TextColor:   l.TextColor,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
