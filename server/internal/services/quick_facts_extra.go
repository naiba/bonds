package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *QuickFactService) ToggleShowQuickFacts(contactID, vaultID string) (bool, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return false, err
	}
	var contact models.Contact
	if err := s.db.Where("id = ?", contactID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrContactNotFound
		}
		return false, err
	}
	newValue := !contact.ShowQuickFacts
	if err := s.db.Model(&contact).Update("show_quick_facts", newValue).Error; err != nil {
		return false, err
	}
	return newValue, nil
}
