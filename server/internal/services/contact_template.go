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

func (s *ContactTemplateService) UpdateTemplate(contactID, vaultID, userID string, req dto.UpdateContactTemplateRequest) (*dto.ContactResponse, error) {
	if err := validateContactTemplateBelongsToVault(s.db, req.TemplateID, vaultID); err != nil {
		return nil, err
	}
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.TemplateID = req.TemplateID
	if err := updateContactProfile(s.db, &contact, vaultID, "template_id"); err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func validateContactTemplateBelongsToVault(db *gorm.DB, templateID *uint, vaultID string) error {
	if templateID == nil {
		return nil
	}
	var count int64
	if err := db.Model(&models.VaultContactTemplate{}).
		Where("id = ? AND vault_id = ?", *templateID, vaultID).
		Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return ErrContactLayoutNotFound
	}
	return nil
}
