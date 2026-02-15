package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrTemplateNotFound = errors.New("template not found")
var ErrTemplatePageNotFound = errors.New("template page not found")
var ErrTemplatePageCannotBeDeleted = errors.New("template page cannot be deleted")

type TemplatePageService struct {
	db *gorm.DB
}

func NewTemplatePageService(db *gorm.DB) *TemplatePageService {
	return &TemplatePageService{db: db}
}

func (s *TemplatePageService) validateTemplateOwnership(templateID uint, accountID string) error {
	var tmpl models.Template
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplateNotFound
		}
		return err
	}
	return nil
}

func (s *TemplatePageService) List(templateID uint, accountID string) ([]dto.TemplatePageResponse, error) {
	if err := s.validateTemplateOwnership(templateID, accountID); err != nil {
		return nil, err
	}
	var pages []models.TemplatePage
	if err := s.db.Where("template_id = ?", templateID).Order("position ASC, id ASC").Find(&pages).Error; err != nil {
		return nil, err
	}
	result := make([]dto.TemplatePageResponse, len(pages))
	for i, p := range pages {
		result[i] = toTemplatePageResponse(&p)
	}
	return result, nil
}

func (s *TemplatePageService) Get(pageID uint, accountID string) (*dto.TemplatePageResponse, error) {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplatePageNotFound
		}
		return nil, err
	}
	if page.Template.AccountID != accountID {
		return nil, ErrTemplatePageNotFound
	}
	resp := toTemplatePageResponse(&page)
	return &resp, nil
}

func (s *TemplatePageService) Create(templateID uint, accountID string, req dto.CreateTemplatePageRequest) (*dto.TemplatePageResponse, error) {
	if err := s.validateTemplateOwnership(templateID, accountID); err != nil {
		return nil, err
	}
	page := models.TemplatePage{
		TemplateID: templateID,
		Name:       strPtrOrNil(req.Name),
		Slug:       req.Slug,
		Position:   &req.Position,
		Type:       strPtrOrNil(req.Type),
	}
	if err := s.db.Create(&page).Error; err != nil {
		return nil, err
	}
	resp := toTemplatePageResponse(&page)
	return &resp, nil
}

func (s *TemplatePageService) Update(pageID uint, accountID string, req dto.UpdateTemplatePageRequest) (*dto.TemplatePageResponse, error) {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplatePageNotFound
		}
		return nil, err
	}
	if page.Template.AccountID != accountID {
		return nil, ErrTemplatePageNotFound
	}
	page.Name = strPtrOrNil(req.Name)
	if req.Slug != "" {
		page.Slug = req.Slug
	}
	page.Position = &req.Position
	page.Type = strPtrOrNil(req.Type)
	if err := s.db.Save(&page).Error; err != nil {
		return nil, err
	}
	resp := toTemplatePageResponse(&page)
	return &resp, nil
}

func (s *TemplatePageService) Delete(pageID uint, accountID string) error {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplatePageNotFound
		}
		return err
	}
	if page.Template.AccountID != accountID {
		return ErrTemplatePageNotFound
	}
	if !page.CanBeDeleted {
		return ErrTemplatePageCannotBeDeleted
	}
	return s.db.Delete(&page).Error
}

func (s *TemplatePageService) UpdatePosition(pageID uint, accountID string, position int) error {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplatePageNotFound
		}
		return err
	}
	if page.Template.AccountID != accountID {
		return ErrTemplatePageNotFound
	}
	return s.db.Model(&page).Update("position", position).Error
}

func (s *TemplatePageService) ListModules(pageID uint, accountID string) ([]dto.TemplatePageModuleResponse, error) {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplatePageNotFound
		}
		return nil, err
	}
	if page.Template.AccountID != accountID {
		return nil, ErrTemplatePageNotFound
	}

	var entries []models.ModuleTemplatePage
	if err := s.db.Where("template_page_id = ?", pageID).Order("position ASC").Find(&entries).Error; err != nil {
		return nil, err
	}

	result := make([]dto.TemplatePageModuleResponse, 0, len(entries))
	for _, e := range entries {
		var mod models.Module
		if err := s.db.Where("id = ?", e.ModuleID).First(&mod).Error; err != nil {
			continue
		}
		pos := 0
		if e.Position != nil {
			pos = *e.Position
		}
		result = append(result, dto.TemplatePageModuleResponse{
			ModuleID:       e.ModuleID,
			TemplatePageID: e.TemplatePageID,
			Position:       pos,
			ModuleName:     ptrToStr(mod.Name),
			ModuleType:     ptrToStr(mod.Type),
			CreatedAt:      e.CreatedAt,
		})
	}
	return result, nil
}

func (s *TemplatePageService) AddModule(pageID uint, accountID string, req dto.AddModuleToPageRequest) error {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplatePageNotFound
		}
		return err
	}
	if page.Template.AccountID != accountID {
		return ErrTemplatePageNotFound
	}

	entry := models.ModuleTemplatePage{
		TemplatePageID: pageID,
		ModuleID:       req.ModuleID,
		Position:       &req.Position,
	}
	return s.db.Create(&entry).Error
}

func (s *TemplatePageService) RemoveModule(pageID, moduleID uint, accountID string) error {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplatePageNotFound
		}
		return err
	}
	if page.Template.AccountID != accountID {
		return ErrTemplatePageNotFound
	}

	result := s.db.Where("template_page_id = ? AND module_id = ?", pageID, moduleID).Delete(&models.ModuleTemplatePage{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTemplatePageNotFound
	}
	return nil
}

func (s *TemplatePageService) UpdateModulePosition(pageID, moduleID uint, accountID string, position int) error {
	var page models.TemplatePage
	if err := s.db.Preload("Template").Where("id = ?", pageID).First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTemplatePageNotFound
		}
		return err
	}
	if page.Template.AccountID != accountID {
		return ErrTemplatePageNotFound
	}

	result := s.db.Model(&models.ModuleTemplatePage{}).
		Where("template_page_id = ? AND module_id = ?", pageID, moduleID).
		Update("position", position)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTemplatePageNotFound
	}
	return nil
}

func toTemplatePageResponse(p *models.TemplatePage) dto.TemplatePageResponse {
	pos := 0
	if p.Position != nil {
		pos = *p.Position
	}
	return dto.TemplatePageResponse{
		ID:                 p.ID,
		TemplateID:         p.TemplateID,
		Name:               ptrToStr(p.Name),
		NameTranslationKey: ptrToStr(p.NameTranslationKey),
		Slug:               p.Slug,
		Position:           pos,
		Type:               ptrToStr(p.Type),
		CanBeDeleted:       p.CanBeDeleted,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
