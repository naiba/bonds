package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ContactTemplateService struct {
	db *gorm.DB
}

func NewContactTemplateService(db *gorm.DB) *ContactTemplateService {
	return &ContactTemplateService{db: db}
}

func (s *ContactTemplateService) UpdateTemplate(contactID, vaultID string, req dto.UpdateContactTemplateRequest) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.TemplateID = req.TemplateID
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}
