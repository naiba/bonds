package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrPasswordMismatch = errors.New("password does not match")
var ErrAccountHasExternalVaultMembers = errors.New("account cancellation would orphan a shared vault member")

type AccountCancelService struct {
	db *gorm.DB
}

func NewAccountCancelService(db *gorm.DB) *AccountCancelService {
	return &AccountCancelService{db: db}
}

func (s *AccountCancelService) Cancel(userID, accountID, password string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if user.Password != nil {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
			return ErrPasswordMismatch
		}
	}

	return companyEmploymentTransaction(s.db, func(tx *gorm.DB) error {
		// A user may belong to several accounts. Cancel only the selected account;
		// never delete another account's identity or vault memberships.
		var members []models.AccountMembership
		if err := tx.Where("account_id = ?", accountID).Find(&members).Error; err != nil {
			return err
		}
		var vaults []models.Vault
		if err := tx.Where("account_id = ?", accountID).Order("id ASC").Find(&vaults).Error; err != nil {
			return err
		}
		// Reuse deleteVaultCascade (same package) to properly clean up all
		// vault child tables — fixes the same incomplete-deletion bug as DeleteVault.
		for _, v := range vaults {
			if err := deleteVaultCascade(tx, v.ID); err != nil {
				return err
			}
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&models.Invitation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&models.AccountMembership{}).Error; err != nil {
			return err
		}
		for _, member := range members {
			var owned models.User
			if err := tx.First(&owned, "id = ?", member.UserID).Error; err != nil {
				return err
			}
			if owned.AccountID != accountID {
				continue
			}
			var other models.AccountMembership
			err := tx.Where("user_id = ?", owned.ID).Order("id ASC").First(&other).Error
			if err == nil {
				if err := tx.Model(&owned).Updates(map[string]interface{}{"account_id": other.AccountID, "is_account_administrator": other.IsAdmin}).Error; err != nil {
					return err
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var externalVaults int64
			if err := tx.Model(&models.UserVault{}).Where("user_id = ?", owned.ID).Count(&externalVaults).Error; err != nil {
				return err
			}
			if externalVaults != 0 {
				return ErrAccountHasExternalVaultMembers
			}
			if err := tx.Where("user_id = ?", owned.ID).Delete(&models.UserNotificationChannel{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&owned).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ?", accountID).Delete(&models.Account{}).Error
	})
}
