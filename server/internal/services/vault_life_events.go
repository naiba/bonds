package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultLifeEventService struct {
	db *gorm.DB
}

func NewVaultLifeEventService(db *gorm.DB) *VaultLifeEventService {
	return &VaultLifeEventService{db: db}
}

func (s *VaultLifeEventService) ListCategories(vaultID string) ([]dto.LifeEventCategoryResponse, error) {
	var cats []models.LifeEventCategory
	if err := s.db.Where("vault_id = ?", vaultID).Preload("LifeEventTypes").Order("position ASC").Find(&cats).Error; err != nil {
		return nil, err
	}
	result := make([]dto.LifeEventCategoryResponse, len(cats))
	for i, c := range cats {
		result[i] = toLifeEventCategoryResponse(&c)
	}
	return result, nil
}

func (s *VaultLifeEventService) CreateCategory(vaultID string, req dto.CreateLifeEventCategoryRequest) (*dto.LifeEventCategoryResponse, error) {
	label := req.Label
	cat := models.LifeEventCategory{
		VaultID:      vaultID,
		Label:        &label,
		Position:     req.Position,
		CanBeDeleted: true,
	}
	if err := s.db.Create(&cat).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventCategoryResponse(&cat)
	return &resp, nil
}

func (s *VaultLifeEventService) UpdateCategory(id uint, vaultID string, req dto.UpdateLifeEventCategoryRequest) (*dto.LifeEventCategoryResponse, error) {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeCategoryNotFound
		}
		return nil, err
	}
	label := req.Label
	cat.Label = &label
	cat.Position = req.Position
	if err := s.db.Save(&cat).Error; err != nil {
		return nil, err
	}
	if err := s.db.Preload("LifeEventTypes").First(&cat, "id = ?", cat.ID).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventCategoryResponse(&cat)
	return &resp, nil
}

func (s *VaultLifeEventService) UpdateCategoryPosition(id uint, vaultID string, position int) (*dto.LifeEventCategoryResponse, error) {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeCategoryNotFound
		}
		return nil, err
	}
	cat.Position = &position
	if err := s.db.Save(&cat).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventCategoryResponse(&cat)
	return &resp, nil
}

func (s *VaultLifeEventService) DeleteCategory(id uint, vaultID string) error {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeCategoryNotFound
		}
		return err
	}
	if !cat.CanBeDeleted {
		return ErrCannotDeleteDefault
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("life_event_category_id = ?", id).Delete(&models.LifeEventType{}).Error; err != nil {
			return err
		}
		return tx.Delete(&cat).Error
	})
}

func (s *VaultLifeEventService) CreateType(categoryID uint, vaultID string, req dto.CreateLifeEventTypeRequest) (*dto.LifeEventTypeResponse, error) {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", categoryID, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeCategoryNotFound
		}
		return nil, err
	}
	label := req.Label
	lt := models.LifeEventType{
		LifeEventCategoryID: categoryID,
		Label:               &label,
		Position:            req.Position,
		CanBeDeleted:        true,
	}
	if err := s.db.Create(&lt).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventTypeResponse(&lt)
	return &resp, nil
}

func (s *VaultLifeEventService) UpdateType(typeID, categoryID uint, vaultID string, req dto.UpdateLifeEventTypeRequest) (*dto.LifeEventTypeResponse, error) {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", categoryID, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeCategoryNotFound
		}
		return nil, err
	}
	var lt models.LifeEventType
	if err := s.db.Where("id = ? AND life_event_category_id = ?", typeID, categoryID).First(&lt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeTypeNotFound
		}
		return nil, err
	}
	label := req.Label
	lt.Label = &label
	lt.Position = req.Position
	if err := s.db.Save(&lt).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventTypeResponse(&lt)
	return &resp, nil
}

func (s *VaultLifeEventService) UpdateTypePosition(typeID, categoryID uint, vaultID string, position int) (*dto.LifeEventTypeResponse, error) {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", categoryID, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeCategoryNotFound
		}
		return nil, err
	}
	var lt models.LifeEventType
	if err := s.db.Where("id = ? AND life_event_category_id = ?", typeID, categoryID).First(&lt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeTypeNotFound
		}
		return nil, err
	}
	lt.Position = &position
	if err := s.db.Save(&lt).Error; err != nil {
		return nil, err
	}
	resp := toLifeEventTypeResponse(&lt)
	return &resp, nil
}

func (s *VaultLifeEventService) DeleteType(typeID, categoryID uint, vaultID string) error {
	var cat models.LifeEventCategory
	if err := s.db.Where("id = ? AND vault_id = ?", categoryID, vaultID).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeCategoryNotFound
		}
		return err
	}
	var lt models.LifeEventType
	if err := s.db.Where("id = ? AND life_event_category_id = ?", typeID, categoryID).First(&lt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeTypeNotFound
		}
		return err
	}
	if !lt.CanBeDeleted {
		return ErrCannotDeleteDefault
	}
	return s.db.Delete(&lt).Error
}

func toLifeEventCategoryResponse(c *models.LifeEventCategory) dto.LifeEventCategoryResponse {
	types := make([]dto.LifeEventTypeResponse, len(c.LifeEventTypes))
	for i, t := range c.LifeEventTypes {
		types[i] = toLifeEventTypeResponse(&t)
	}
	return dto.LifeEventCategoryResponse{
		ID:           c.ID,
		Label:        ptrToStr(c.Label),
		CanBeDeleted: c.CanBeDeleted,
		Position:     c.Position,
		Types:        types,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func toLifeEventTypeResponse(t *models.LifeEventType) dto.LifeEventTypeResponse {
	return dto.LifeEventTypeResponse{
		ID:           t.ID,
		CategoryID:   t.LifeEventCategoryID,
		Label:        ptrToStr(t.Label),
		CanBeDeleted: t.CanBeDeleted,
		Position:     t.Position,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
