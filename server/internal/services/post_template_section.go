package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPostTemplateSectionNotFound = errors.New("post template section not found")
var ErrPostTemplateNotFound = errors.New("post template not found")

type PostTemplateSectionService struct {
	db *gorm.DB
}

func NewPostTemplateSectionService(db *gorm.DB) *PostTemplateSectionService {
	return &PostTemplateSectionService{db: db}
}

func (s *PostTemplateSectionService) List(accountID string, templateID uint) ([]dto.PostTemplateSectionResponse, error) {
	var tpl models.PostTemplate
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostTemplateNotFound
		}
		return nil, err
	}
	var sections []models.PostTemplateSection
	if err := s.db.Where("post_template_id = ?", templateID).Order("position ASC, id ASC").Find(&sections).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PostTemplateSectionResponse, len(sections))
	for i, s := range sections {
		result[i] = toPostTemplateSectionResponse(&s)
	}
	return result, nil
}

func (s *PostTemplateSectionService) Create(accountID string, templateID uint, req dto.CreatePostTemplateSectionRequest) (*dto.PostTemplateSectionResponse, error) {
	var tpl models.PostTemplate
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostTemplateNotFound
		}
		return nil, err
	}
	section := models.PostTemplateSection{
		PostTemplateID: templateID,
		Label:          strPtrOrNil(req.Label),
		Position:       req.Position,
	}
	if err := s.db.Create(&section).Error; err != nil {
		return nil, err
	}
	resp := toPostTemplateSectionResponse(&section)
	return &resp, nil
}

func (s *PostTemplateSectionService) Update(accountID string, templateID uint, sectionID uint, req dto.UpdatePostTemplateSectionRequest) (*dto.PostTemplateSectionResponse, error) {
	var tpl models.PostTemplate
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostTemplateNotFound
		}
		return nil, err
	}
	var section models.PostTemplateSection
	if err := s.db.Where("id = ? AND post_template_id = ?", sectionID, templateID).First(&section).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostTemplateSectionNotFound
		}
		return nil, err
	}
	section.Label = strPtrOrNil(req.Label)
	section.Position = req.Position
	if err := s.db.Save(&section).Error; err != nil {
		return nil, err
	}
	resp := toPostTemplateSectionResponse(&section)
	return &resp, nil
}

func (s *PostTemplateSectionService) Delete(accountID string, templateID uint, sectionID uint) error {
	var tpl models.PostTemplate
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostTemplateNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND post_template_id = ?", sectionID, templateID).Delete(&models.PostTemplateSection{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPostTemplateSectionNotFound
	}
	return nil
}

func (s *PostTemplateSectionService) UpdatePosition(accountID string, templateID uint, sectionID uint, position int) error {
	var tpl models.PostTemplate
	if err := s.db.Where("id = ? AND account_id = ?", templateID, accountID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostTemplateNotFound
		}
		return err
	}
	result := s.db.Model(&models.PostTemplateSection{}).Where("id = ? AND post_template_id = ?", sectionID, templateID).Update("position", position)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPostTemplateSectionNotFound
	}
	return nil
}

func toPostTemplateSectionResponse(s *models.PostTemplateSection) dto.PostTemplateSectionResponse {
	return dto.PostTemplateSectionResponse{
		ID:             s.ID,
		PostTemplateID: s.PostTemplateID,
		Label:          ptrToStr(s.Label),
		Position:       s.Position,
		CanBeDeleted:   s.CanBeDeleted,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
