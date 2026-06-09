package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultQuickFactTemplateService struct {
	db        *gorm.DB
	uploadDir string
}

func NewVaultQuickFactTemplateService(db *gorm.DB) *VaultQuickFactTemplateService {
	return &VaultQuickFactTemplateService{db: db}
}

func (s *VaultQuickFactTemplateService) SetUploadDir(uploadDir string) {
	s.uploadDir = uploadDir
}

func (s *VaultQuickFactTemplateService) List(vaultID string) ([]dto.QuickFactTemplateResponse, error) {
	var tpls []models.VaultQuickFactsTemplate
	if err := s.db.Where("vault_id = ?", vaultID).Order("position ASC").Find(&tpls).Error; err != nil {
		return nil, err
	}
	result := make([]dto.QuickFactTemplateResponse, len(tpls))
	for i, t := range tpls {
		result[i] = toQuickFactTemplateResponse(&t)
	}
	return result, nil
}

func (s *VaultQuickFactTemplateService) Create(vaultID string, req dto.CreateQuickFactTemplateRequest) (*dto.QuickFactTemplateResponse, error) {
	fieldType, selectOptions, err := validateQuickFactTemplateInput(req.FieldType, req.SelectOptions, req.DefaultValue)
	if err != nil {
		return nil, err
	}
	label := req.Label
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	tpl := models.VaultQuickFactsTemplate{
		VaultID:       vaultID,
		Label:         &label,
		FieldType:     fieldType,
		SelectOptions: selectOptions,
		Required:      req.Required,
		HelpText:      req.HelpText,
		DefaultValue:  req.DefaultValue,
		Position:      position,
	}
	if err := s.db.Create(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) Update(id uint, vaultID string, req dto.UpdateQuickFactTemplateRequest) (*dto.QuickFactTemplateResponse, error) {
	var tpl models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactTplNotFound
		}
		return nil, err
	}
	fieldType, selectOptions, err := validateQuickFactTemplateInput(req.FieldType, req.SelectOptions, req.DefaultValue)
	if err != nil {
		return nil, err
	}
	if err := s.validateExistingFactsRemainCompatible(&tpl, fieldType, selectOptions, req.Required); err != nil {
		return nil, err
	}
	label := req.Label
	tpl.Label = &label
	tpl.FieldType = fieldType
	tpl.SelectOptions = selectOptions
	tpl.Required = req.Required
	tpl.HelpText = req.HelpText
	tpl.DefaultValue = req.DefaultValue
	if req.Position != nil {
		tpl.Position = *req.Position
	}
	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) validateExistingFactsRemainCompatible(tpl *models.VaultQuickFactsTemplate, nextFieldType string, nextSelectOptions *string, nextRequired bool) error {
	var factCount int64
	if err := s.db.Model(&models.QuickFact{}).Where("vault_quick_facts_template_id = ?", tpl.ID).Count(&factCount).Error; err != nil {
		return err
	}
	if factCount == 0 {
		return nil
	}
	if normalizeQuickFactFieldType(tpl.FieldType) != normalizeQuickFactFieldType(nextFieldType) {
		return ErrQuickFactTemplateInUse
	}
	fieldType := normalizeQuickFactFieldType(tpl.FieldType)
	if nextRequired {
		var facts []models.QuickFact
		if err := s.db.Where("vault_quick_facts_template_id = ?", tpl.ID).Find(&facts).Error; err != nil {
			return err
		}
		for _, fact := range facts {
			if !quickFactHasRequiredValue(fact, fieldType) {
				return ErrQuickFactTemplateInUse
			}
		}
	}
	if fieldType != QuickFactFieldSelect {
		return nil
	}
	nextOptions := parseQuickFactSelectOptions(nextSelectOptions)
	var facts []models.QuickFact
	if err := s.db.Where("vault_quick_facts_template_id = ?", tpl.ID).Find(&facts).Error; err != nil {
		return err
	}
	for _, fact := range facts {
		if fact.ValueOption != nil && strings.TrimSpace(*fact.ValueOption) != "" && !quickFactOptionAllowed(*fact.ValueOption, nextOptions) {
			return ErrQuickFactTemplateInUse
		}
		if fact.ValueOption == nil && strings.TrimSpace(fact.Content) != "" && !quickFactOptionAllowed(fact.Content, nextOptions) {
			return ErrQuickFactTemplateInUse
		}
	}
	return nil
}

func quickFactHasRequiredValue(fact models.QuickFact, fieldType string) bool {
	content := strings.TrimSpace(fact.Content)
	switch normalizeQuickFactFieldType(fieldType) {
	case QuickFactFieldText:
		return (fact.ValueText != nil && strings.TrimSpace(*fact.ValueText) != "") || content != ""
	case QuickFactFieldNumber:
		return fact.ValueNumber != nil || content != ""
	case QuickFactFieldDate:
		return (fact.ValueDate != nil && strings.TrimSpace(*fact.ValueDate) != "") || content != ""
	case QuickFactFieldSelect:
		return (fact.ValueOption != nil && strings.TrimSpace(*fact.ValueOption) != "") || content != ""
	case QuickFactFieldPhoto, QuickFactFieldDocument:
		return fact.FileID != nil || content != ""
	default:
		return content != ""
	}
}

func (s *VaultQuickFactTemplateService) UpdatePosition(id uint, vaultID string, position int) (*dto.QuickFactTemplateResponse, error) {
	var tpl models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactTplNotFound
		}
		return nil, err
	}
	tpl.Position = position
	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactTemplateResponse(&tpl)
	return &resp, nil
}

func (s *VaultQuickFactTemplateService) Delete(id uint, vaultID string) error {
	return s.DeleteWithUploadDir(id, vaultID, s.uploadDir)
}

func (s *VaultQuickFactTemplateService) DeleteWithUploadDir(id uint, vaultID string, uploadDir string) error {
	var files []models.File
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var tpl models.VaultQuickFactsTemplate
		if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&tpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQuickFactTplNotFound
			}
			return err
		}
		if err := tx.Joins("JOIN quick_facts ON quick_facts.file_id = files.id").
			Where("quick_facts.vault_quick_facts_template_id = ?", id).
			Find(&files).Error; err != nil {
			return err
		}
		if err := tx.Where("vault_quick_facts_template_id = ?", id).Delete(&models.QuickFact{}).Error; err != nil {
			return err
		}
		if len(files) > 0 {
			fileIDs := make([]uint, len(files))
			for i := range files {
				fileIDs[i] = files[i].ID
			}
			if err := tx.Where("id IN ?", fileIDs).Delete(&models.File{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&tpl).Error
	})
	if err != nil {
		return err
	}
	if uploadDir != "" {
		for _, file := range files {
			_ = os.Remove(filepath.Join(uploadDir, file.UUID))
		}
	}
	return nil
}

func toQuickFactTemplateResponse(t *models.VaultQuickFactsTemplate) dto.QuickFactTemplateResponse {
	return dto.QuickFactTemplateResponse{
		ID:            t.ID,
		Label:         ptrToStr(t.Label),
		FieldType:     normalizeQuickFactFieldType(t.FieldType),
		SelectOptions: parseQuickFactSelectOptions(t.SelectOptions),
		Required:      t.Required,
		HelpText:      t.HelpText,
		DefaultValue:  t.DefaultValue,
		Position:      t.Position,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}
