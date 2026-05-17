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
	if err := s.validateTemplateBelongsToVault(templateID, vaultID); err != nil {
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

func (s *QuickFactService) ListAll(contactID, vaultID string) ([]dto.QuickFactGroupResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}

	var templates []models.VaultQuickFactsTemplate
	if err := s.db.Where("vault_id = ?", vaultID).Order("position ASC, id ASC").Find(&templates).Error; err != nil {
		return nil, err
	}

	var facts []models.QuickFact
	if err := s.db.Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
		Where("quick_facts.contact_id = ? AND vault_quick_facts_templates.vault_id = ?", contactID, vaultID).
		Order("quick_facts.created_at DESC").
		Find(&facts).Error; err != nil {
		return nil, err
	}

	factsByTemplateID := make(map[uint][]dto.QuickFactResponse, len(templates))
	for _, fact := range facts {
		factsByTemplateID[fact.VaultQuickFactsTemplateID] = append(factsByTemplateID[fact.VaultQuickFactsTemplateID], toQuickFactResponse(&fact))
	}

	groups := make([]dto.QuickFactGroupResponse, len(templates))
	for i, template := range templates {
		groups[i] = dto.QuickFactGroupResponse{
			TemplateID:    template.ID,
			TemplateLabel: ptrToStr(template.Label),
			Position:      template.Position,
			Facts:         factsByTemplateID[template.ID],
		}
		if groups[i].Facts == nil {
			groups[i].Facts = []dto.QuickFactResponse{}
		}
	}
	return groups, nil
}

func (s *QuickFactService) Create(contactID, vaultID string, templateID uint, req dto.CreateQuickFactRequest) (*dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if err := s.validateTemplateBelongsToVault(templateID, vaultID); err != nil {
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
	if err := s.db.Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
		Where("quick_facts.id = ? AND quick_facts.contact_id = ? AND vault_quick_facts_templates.vault_id = ?", id, contactID, vaultID).
		First(&fact).Error; err != nil {
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
	if err := s.db.Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
		Where("quick_facts.id = ? AND quick_facts.contact_id = ? AND vault_quick_facts_templates.vault_id = ?", id, contactID, vaultID).
		First(&fact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuickFactNotFound
		}
		return err
	}
	return s.db.Delete(&fact).Error
}

func (s *QuickFactService) validateTemplateBelongsToVault(templateID uint, vaultID string) error {
	var template models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", templateID, vaultID).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQuickFactTplNotFound
		}
		return err
	}
	return nil
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
