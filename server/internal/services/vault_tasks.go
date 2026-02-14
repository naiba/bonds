package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultTaskService struct {
	db *gorm.DB
}

func NewVaultTaskService(db *gorm.DB) *VaultTaskService {
	return &VaultTaskService{db: db}
}

func (s *VaultTaskService) List(vaultID string) ([]dto.VaultTaskResponse, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Select("id").Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}
	if len(contactIDs) == 0 {
		return []dto.VaultTaskResponse{}, nil
	}

	var tasks []models.ContactTask
	if err := s.db.Where("contact_id IN ?", contactIDs).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultTaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toVaultTaskResponse(&t)
	}
	return result, nil
}

func toVaultTaskResponse(t *models.ContactTask) dto.VaultTaskResponse {
	desc := ""
	if t.Description != nil {
		desc = *t.Description
	}
	return dto.VaultTaskResponse{
		ID:          t.ID,
		ContactID:   t.ContactID,
		AuthorName:  t.AuthorName,
		Label:       t.Label,
		Description: desc,
		Completed:   t.Completed,
		CompletedAt: t.CompletedAt,
		DueAt:       t.DueAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
