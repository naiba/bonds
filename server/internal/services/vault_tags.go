package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultTagService struct {
	db *gorm.DB
}

func NewVaultTagService(db *gorm.DB) *VaultTagService {
	return &VaultTagService{db: db}
}

func (s *VaultTagService) List(vaultID string) ([]dto.TagResponse, error) {
	var tags []models.Tag
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&tags).Error; err != nil {
		return nil, err
	}
	result := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		result[i] = toTagResponse(&t)
	}
	return result, nil
}

func (s *VaultTagService) Create(vaultID string, req dto.CreateTagRequest) (*dto.TagResponse, error) {
	tag := models.Tag{
		VaultID: vaultID,
		Name:    req.Name,
		Slug:    generateSlug(req.Name),
	}
	if err := s.db.Create(&tag).Error; err != nil {
		return nil, err
	}
	resp := toTagResponse(&tag)
	return &resp, nil
}

func (s *VaultTagService) Update(id uint, vaultID string, req dto.UpdateTagRequest) (*dto.TagResponse, error) {
	var tag models.Tag
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	tag.Name = req.Name
	tag.Slug = generateSlug(req.Name)
	if err := s.db.Save(&tag).Error; err != nil {
		return nil, err
	}
	resp := toTagResponse(&tag)
	return &resp, nil
}

func (s *VaultTagService) Delete(id uint, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.Tag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}

func toTagResponse(t *models.Tag) dto.TagResponse {
	return dto.TagResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
