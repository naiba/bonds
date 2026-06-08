package models

import (
	"errors"

	"github.com/naiba/bonds/internal/i18n"
	"gorm.io/gorm"
)

const (
	defaultGiftContactModuleType           = "gifts"
	defaultGiftContactModuleTranslationKey = "seed.modules.gifts"
)

// BackfillGiftContactModules makes the shipped contact Gifts module visible on
// accounts that pre-date issue #162. Only the non-deletable default template is
// touched so user-created templates keep their existing module layout.
func BackfillGiftContactModules(db *gorm.DB) error {
	var accountIDs []string
	if err := db.Model(&Account{}).Pluck("id", &accountIDs).Error; err != nil {
		return err
	}

	for _, accountID := range accountIDs {
		if err := db.Transaction(func(tx *gorm.DB) error {
			giftModule, defaultGiftModules, err := ensureDefaultGiftContactModule(tx, accountID)
			if err != nil {
				return err
			}
			return bindDefaultGiftContactModule(tx, accountID, giftModule.ID, defaultGiftModules)
		}); err != nil {
			return err
		}
	}

	return nil
}

func ensureDefaultGiftContactModule(tx *gorm.DB, accountID string) (*Module, []Module, error) {
	var modules []Module
	if err := tx.Where(
		"account_id = ? AND type = ? AND name_translation_key = ?",
		accountID,
		defaultGiftContactModuleType,
		defaultGiftContactModuleTranslationKey,
	).Order("id ASC").Find(&modules).Error; err != nil {
		return nil, nil, err
	}

	if len(modules) > 0 {
		module := modules[0]
		if module.CanBeDeleted {
			if err := tx.Model(&Module{}).Where("id = ?", module.ID).Update("can_be_deleted", false).Error; err != nil {
				return nil, nil, err
			}
			module.CanBeDeleted = false
		}
		return &module, modules, nil
	}

	module := Module{
		AccountID:          accountID,
		Name:               strPtr(i18n.T("en", defaultGiftContactModuleTranslationKey)),
		NameTranslationKey: strPtr(defaultGiftContactModuleTranslationKey),
		Type:               strPtr(defaultGiftContactModuleType),
	}
	if err := tx.Create(&module).Error; err != nil {
		return nil, nil, err
	}
	if err := tx.Model(&module).Update("can_be_deleted", false).Error; err != nil {
		return nil, nil, err
	}
	module.CanBeDeleted = false

	return &module, []Module{module}, nil
}

func bindDefaultGiftContactModule(tx *gorm.DB, accountID string, moduleID uint, defaultGiftModules []Module) error {
	var tmpl Template
	if err := tx.Where("account_id = ? AND can_be_deleted = ?", accountID, false).Order("id ASC").First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	var informationPage TemplatePage
	if err := tx.Where("template_id = ? AND slug = ?", tmpl.ID, "information").First(&informationPage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	moduleIDs := make([]uint, 0, len(defaultGiftModules))
	for _, module := range defaultGiftModules {
		moduleIDs = append(moduleIDs, module.ID)
	}

	var existingBindingCount int64
	if err := tx.Model(&ModuleTemplatePage{}).
		Where("template_page_id = ? AND module_id IN ?", informationPage.ID, moduleIDs).
		Count(&existingBindingCount).Error; err != nil {
		return err
	}
	if existingBindingCount > 0 {
		return nil
	}

	var maxPosition int
	if err := tx.Model(&ModuleTemplatePage{}).
		Where("template_page_id = ?", informationPage.ID).
		Select("COALESCE(MAX(position), 0)").
		Scan(&maxPosition).Error; err != nil {
		return err
	}
	position := maxPosition + 1

	return tx.Create(&ModuleTemplatePage{
		TemplatePageID: informationPage.ID,
		ModuleID:       moduleID,
		Position:       &position,
	}).Error
}
