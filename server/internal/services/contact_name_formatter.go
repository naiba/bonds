package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/utils"
	"gorm.io/gorm"
)

type contactNameFormatter struct {
	db              *gorm.DB
	userNameOrder   string
	vaultNameOrders map[string]string
}

func newContactNameFormatter(db *gorm.DB, userID string) (*contactNameFormatter, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}
	userNameOrder, err := getUserNameOrder(db, userID)
	if err != nil {
		return nil, err
	}
	return &contactNameFormatter{
		db:              db,
		userNameOrder:   userNameOrder,
		vaultNameOrders: make(map[string]string),
	}, nil
}

func (f *contactNameFormatter) format(contact *models.Contact, fallback string) (string, error) {
	if contact == nil {
		return fallback, nil
	}
	nameOrder, err := f.nameOrderForVault(contact.VaultID)
	if err != nil {
		return "", err
	}
	return utils.FormatContactName(nameOrder, contact, fallback), nil
}

func (f *contactNameFormatter) nameOrderForVault(vaultID string) (string, error) {
	if nameOrder, ok := f.vaultNameOrders[vaultID]; ok {
		return nameOrder, nil
	}
	var vault models.Vault
	if err := f.db.Select("id", "name_order").First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrVaultNotFound
		}
		return "", err
	}
	nameOrder := effectiveVaultNameOrder(&vault, f.userNameOrder)
	f.vaultNameOrders[vaultID] = nameOrder
	return nameOrder, nil
}
