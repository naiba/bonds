package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ContactTabService struct {
	db *gorm.DB
}

func NewContactTabService(db *gorm.DB) *ContactTabService {
	return &ContactTabService{db: db}
}

func (s *ContactTabService) GetTabs(contactID, vaultID string) (*dto.ContactTabsResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var tmpl models.Template

	if contact.TemplateID != nil {
		if err := s.db.First(&tmpl, *contact.TemplateID).Error; err != nil {
			return nil, err
		}
	} else {
		// Fall back to the account's default (non-deletable) template
		var vault models.Vault
		if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
			return nil, err
		}
		if err := s.db.Where("account_id = ? AND can_be_deleted = ?", vault.AccountID, false).First(&tmpl).Error; err != nil {
			return nil, err
		}
	}

	var pages []models.TemplatePage
	if err := s.db.Where("template_id = ?", tmpl.ID).Order("position ASC").Find(&pages).Error; err != nil {
		return nil, err
	}

	result := &dto.ContactTabsResponse{
		TemplateID:   tmpl.ID,
		TemplateName: derefStr(tmpl.Name),
		Pages:        make([]dto.ContactTabPage, len(pages)),
	}

	for i, page := range pages {
		var pivots []models.ModuleTemplatePage
		if err := s.db.Where("template_page_id = ?", page.ID).Order("position ASC").Find(&pivots).Error; err != nil {
			return nil, err
		}

		modules := make([]dto.ContactTabModule, 0, len(pivots))
		for _, pivot := range pivots {
			var mod models.Module
			if err := s.db.First(&mod, pivot.ModuleID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return nil, err
			}
			pos := 0
			if pivot.Position != nil {
				pos = *pivot.Position
			}
			modules = append(modules, dto.ContactTabModule{
				ID:       mod.ID,
				Name:     derefStr(mod.Name),
				Type:     derefStr(mod.Type),
				Position: pos,
			})
		}

		pos := 0
		if page.Position != nil {
			pos = *page.Position
		}
		result.Pages[i] = dto.ContactTabPage{
			ID:       page.ID,
			Name:     derefStr(page.Name),
			Slug:     page.Slug,
			Position: pos,
			Type:     derefStr(page.Type),
			Modules:  modules,
		}
	}

	return result, nil
}

func derefStr(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}
