package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrPasswordMismatch = errors.New("password does not match")

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

	return s.db.Transaction(func(tx *gorm.DB) error {
		var vaults []models.Vault
		if err := tx.Where("account_id = ?", accountID).Find(&vaults).Error; err != nil {
			return err
		}
		for _, v := range vaults {
			if err := tx.Where("vault_id = ?", v.ID).Delete(&models.UserVault{}).Error; err != nil {
				return err
			}
			if err := tx.Where("vault_id = ?", v.ID).Delete(&models.Contact{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&models.Vault{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&models.User{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", accountID).Delete(&models.Account{}).Error
	})
}
