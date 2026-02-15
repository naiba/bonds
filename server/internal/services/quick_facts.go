package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrQuickFactNotFound = errors.New("quick fact not found")

type QuickFactService struct {
	db *gorm.DB
}

func NewQuickFactService(db *gorm.DB) *QuickFactService {
	return &QuickFactService{db: db}
}

func (s *QuickFactService) List(contactID, vaultID string, templateID uint) ([]dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var facts []models.QuickFact
	if err := s.db.Where("contact_id = ? AND vault_quick_facts_template_id = ?", contactID, templateID).
		Order("created_at DESC").Find(&facts).Error; err != nil {
		return nil, err
	}
	result := make([]dto.QuickFactResponse, len(facts))
	for i, f := range facts {
		result[i] = toQuickFactResponse(&f)
	}
	return result, nil
}

func (s *QuickFactService) Create(contactID, vaultID string, templateID uint, req dto.CreateQuickFactRequest) (*dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	fact := models.QuickFact{
		VaultQuickFactsTemplateID: templateID,
		ContactID:                 contactID,
		Content:                   req.Content,
	}
	if err := s.db.Create(&fact).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactResponse(&fact)
	return &resp, nil
}

func (s *QuickFactService) Update(id uint, contactID, vaultID string, req dto.UpdateQuickFactRequest) (*dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var fact models.QuickFact
	if err := s.db.First(&fact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactNotFound
		}
		return nil, err
	}
	fact.Content = req.Content
	if err := s.db.Save(&fact).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactResponse(&fact)
	return &resp, nil
}

func (s *QuickFactService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var fact models.QuickFact
	if err := s.db.First(&fact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuickFactNotFound
		}
		return err
	}
	return s.db.Delete(&fact).Error
}

func toQuickFactResponse(f *models.QuickFact) dto.QuickFactResponse {
	return dto.QuickFactResponse{
		ID:                        f.ID,
		VaultQuickFactsTemplateID: f.VaultQuickFactsTemplateID,
		ContactID:                 f.ContactID,
		Content:                   f.Content,
		CreatedAt:                 f.CreatedAt,
		UpdatedAt:                 f.UpdatedAt,
	}
}
