package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrContactPromotionNotAllowed = errors.New("contact promotion not allowed")

func validateContactPromotionRequest(db *gorm.DB, contact *models.Contact, req dto.UpdateContactRequest) error {
	wantsToList := req.Listed != nil && *req.Listed && !contact.Listed
	wantsToClearVerification := req.NeedsVerification != nil && !*req.NeedsVerification && contact.NeedsVerification

	if !wantsToList && !wantsToClearVerification {
		return nil
	}

	var shadowContactCount int64
	if err := db.Model(&models.UserVault{}).
		Where("vault_id = ? AND contact_id = ?", contact.VaultID, contact.ID).
		Count(&shadowContactCount).Error; err != nil {
		return err
	}
	isShadowContact := shadowContactCount > 0 || !contact.CanBeDeleted
	isHiddenRelationshipPlaceholder := !contact.Listed && contact.NeedsVerification && contact.CanBeDeleted

	if wantsToList {
		if isShadowContact {
			return ErrContactPromotionNotAllowed
		}
		if isHiddenRelationshipPlaceholder {
			if !wantsToClearVerification {
				return ErrContactPromotionNotAllowed
			}
			return nil
		}
		return nil
	}

	if wantsToClearVerification && !contact.Listed && (isShadowContact || isHiddenRelationshipPlaceholder) {
		return ErrContactPromotionNotAllowed
	}

	return nil
}
