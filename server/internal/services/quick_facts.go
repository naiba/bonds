package services

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

const (
	QuickFactFieldText     = "text"
	QuickFactFieldNumber   = "number"
	QuickFactFieldDate     = "date"
	QuickFactFieldSelect   = "select"
	QuickFactFieldPhoto    = "photo"
	QuickFactFieldDocument = "document"
)

var (
	ErrQuickFactNotFound         = errors.New("quick fact not found")
	ErrQuickFactInvalidField     = errors.New("invalid quick fact field")
	ErrQuickFactInvalidValue     = errors.New("invalid quick fact value")
	ErrQuickFactRequiredValue    = errors.New("quick fact value required")
	ErrQuickFactFileTypeMismatch = errors.New("quick fact file type mismatch")
	ErrQuickFactTemplateMismatch = errors.New("quick fact template mismatch")
	ErrQuickFactTemplateInUse    = errors.New("quick fact template has facts")
)

type QuickFactService struct {
	db          *gorm.DB
	fileService *VaultFileService
}

func NewQuickFactService(db *gorm.DB) *QuickFactService {
	return &QuickFactService{db: db}
}

func (s *QuickFactService) SetFileService(fileService *VaultFileService) {
	s.fileService = fileService
}

func (s *QuickFactService) List(contactID, vaultID string, templateID uint) ([]dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if err := s.validateTemplateBelongsToVault(templateID, vaultID); err != nil {
		return nil, err
	}
	var facts []models.QuickFact
	if err := s.db.Preload("VaultQuickFactsTemplate").Preload("File").
		Where("contact_id = ? AND vault_quick_facts_template_id = ?", contactID, templateID).
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
	if err := s.db.Preload("VaultQuickFactsTemplate").Preload("File").
		Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
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
			FieldType:     normalizeQuickFactFieldType(template.FieldType),
			SelectOptions: parseQuickFactSelectOptions(template.SelectOptions),
			Required:      template.Required,
			HelpText:      template.HelpText,
			DefaultValue:  template.DefaultValue,
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
	template, err := s.getTemplate(templateID, vaultID)
	if err != nil {
		return nil, err
	}
	fact, err := buildQuickFactFromRequest(contactID, template, req.Content, req.ValueText, req.ValueNumber, req.ValueDate, req.ValueOption)
	if err != nil {
		return nil, err
	}
	if err := s.db.Create(fact).Error; err != nil {
		return nil, err
	}
	fact.VaultQuickFactsTemplate = *template
	resp := toQuickFactResponse(fact)
	return &resp, nil
}

func (s *QuickFactService) Update(id uint, contactID, vaultID string, templateID uint, req dto.UpdateQuickFactRequest) (*dto.QuickFactResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	fact, err := s.getFact(id, contactID, vaultID)
	if err != nil {
		return nil, err
	}
	if fact.VaultQuickFactsTemplateID != templateID {
		return nil, ErrQuickFactTemplateMismatch
	}
	if isQuickFactFileField(fact.VaultQuickFactsTemplate.FieldType) {
		return nil, ErrQuickFactInvalidField
	}
	updated, err := buildQuickFactFromRequest(contactID, &fact.VaultQuickFactsTemplate, req.Content, req.ValueText, req.ValueNumber, req.ValueDate, req.ValueOption)
	if err != nil {
		return nil, err
	}
	fact.Content = updated.Content
	fact.ValueText = updated.ValueText
	fact.ValueNumber = updated.ValueNumber
	fact.ValueDate = updated.ValueDate
	fact.ValueOption = updated.ValueOption
	fact.FileID = nil
	if err := s.db.Save(fact).Error; err != nil {
		return nil, err
	}
	resp := toQuickFactResponse(fact)
	return &resp, nil
}

func (s *QuickFactService) UploadFile(contactID, vaultID string, templateID uint, authorID string, filename string, mimeType string, size int64, data io.Reader) (*dto.QuickFactResponse, error) {
	if s.fileService == nil {
		return nil, errors.New("quick fact file service not configured")
	}
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	template, err := s.getTemplate(templateID, vaultID)
	if err != nil {
		return nil, err
	}
	fieldType := normalizeQuickFactFieldType(template.FieldType)
	if !isQuickFactFileField(fieldType) {
		return nil, ErrQuickFactInvalidField
	}
	if !quickFactMimeMatchesFieldType(mimeType, fieldType) {
		return nil, ErrQuickFactFileTypeMismatch
	}
	fileType := quickFactFileType(fieldType)
	file, err := s.fileService.Upload(vaultID, contactID, authorID, fileType, filename, mimeType, size, data)
	if err != nil {
		return nil, err
	}

	var createdFact *models.QuickFact
	err = s.db.Transaction(func(tx *gorm.DB) error {
		fact := models.QuickFact{
			VaultQuickFactsTemplateID: template.ID,
			ContactID:                 contactID,
			Content:                   file.Name,
			FileID:                    &file.ID,
		}
		if err := tx.Create(&fact).Error; err != nil {
			return err
		}
		quickFactType := "QuickFact"
		if err := tx.Model(&models.File{}).Where("id = ? AND vault_id = ?", file.ID, vaultID).Updates(map[string]interface{}{
			"fileable_type": quickFactType,
			"fileable_id":   fact.ID,
		}).Error; err != nil {
			return err
		}
		createdFact = &fact
		return nil
	})
	if err != nil {
		_ = s.fileService.ForceDeleteFile(file.ID, vaultID)
		return nil, err
	}
	createdFact.VaultQuickFactsTemplate = *template
	createdFact.File = &models.File{ID: file.ID, Name: file.Name, MimeType: file.MimeType, Type: file.Type, Size: file.Size, CreatedAt: file.CreatedAt, UpdatedAt: file.UpdatedAt}
	resp := toQuickFactResponse(createdFact)
	return &resp, nil
}

func (s *QuickFactService) ReplaceFile(id uint, contactID, vaultID string, templateID uint, authorID string, filename string, mimeType string, size int64, data io.Reader) (*dto.QuickFactResponse, error) {
	if s.fileService == nil {
		return nil, errors.New("quick fact file service not configured")
	}
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	fact, err := s.getFact(id, contactID, vaultID)
	if err != nil {
		return nil, err
	}
	if fact.VaultQuickFactsTemplateID != templateID {
		return nil, ErrQuickFactTemplateMismatch
	}
	fieldType := normalizeQuickFactFieldType(fact.VaultQuickFactsTemplate.FieldType)
	if !isQuickFactFileField(fieldType) {
		return nil, ErrQuickFactInvalidField
	}
	if !quickFactMimeMatchesFieldType(mimeType, fieldType) {
		return nil, ErrQuickFactFileTypeMismatch
	}
	fileType := quickFactFileType(fieldType)
	file, err := s.fileService.Upload(vaultID, contactID, authorID, fileType, filename, mimeType, size, data)
	if err != nil {
		return nil, err
	}
	var oldFileID *uint
	if fact.FileID != nil {
		oldFileIDValue := *fact.FileID
		oldFileID = &oldFileIDValue
	}
	fact.Content = file.Name
	fact.FileID = &file.ID
	fact.ValueText = nil
	fact.ValueNumber = nil
	fact.ValueDate = nil
	fact.ValueOption = nil
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.QuickFact{}).Where("id = ?", fact.ID).Updates(map[string]interface{}{
			"content":      fact.Content,
			"file_id":      file.ID,
			"value_text":   nil,
			"value_number": nil,
			"value_date":   nil,
			"value_option": nil,
		}).Error; err != nil {
			return err
		}
		return assignFileToQuickFact(tx, file.ID, fact.ID, vaultID)
	}); err != nil {
		_ = s.fileService.ForceDeleteFile(file.ID, vaultID)
		return nil, err
	}
	if oldFileID != nil {
		_ = s.fileService.ForceDeleteFile(*oldFileID, vaultID)
	}
	fact.File = &models.File{ID: file.ID, Name: file.Name, MimeType: file.MimeType, Type: file.Type, Size: file.Size, CreatedAt: file.CreatedAt, UpdatedAt: file.UpdatedAt}
	resp := toQuickFactResponse(fact)
	return &resp, nil
}

func (s *QuickFactService) Delete(id uint, contactID, vaultID string, templateID uint) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	fact, err := s.getFact(id, contactID, vaultID)
	if err != nil {
		return err
	}
	if fact.VaultQuickFactsTemplateID != templateID {
		return ErrQuickFactTemplateMismatch
	}
	fileID := fact.FileID
	if err := s.db.Delete(fact).Error; err != nil {
		return err
	}
	if fileID != nil && s.fileService != nil {
		return s.fileService.ForceDeleteFile(*fileID, vaultID)
	}
	return nil
}

func (s *QuickFactService) validateTemplateBelongsToVault(templateID uint, vaultID string) error {
	_, err := s.getTemplate(templateID, vaultID)
	return err
}

func (s *QuickFactService) getTemplate(templateID uint, vaultID string) (*models.VaultQuickFactsTemplate, error) {
	var template models.VaultQuickFactsTemplate
	if err := s.db.Where("id = ? AND vault_id = ?", templateID, vaultID).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactTplNotFound
		}
		return nil, err
	}
	return &template, nil
}

func (s *QuickFactService) getFact(id uint, contactID, vaultID string) (*models.QuickFact, error) {
	var fact models.QuickFact
	if err := s.db.Preload("VaultQuickFactsTemplate").Preload("File").
		Joins("JOIN vault_quick_facts_templates ON vault_quick_facts_templates.id = quick_facts.vault_quick_facts_template_id").
		Where("quick_facts.id = ? AND quick_facts.contact_id = ? AND vault_quick_facts_templates.vault_id = ?", id, contactID, vaultID).
		First(&fact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuickFactNotFound
		}
		return nil, err
	}
	return &fact, nil
}

func buildQuickFactFromRequest(contactID string, template *models.VaultQuickFactsTemplate, legacyContent string, valueText *string, valueNumber *float64, valueDate *string, valueOption *string) (*models.QuickFact, error) {
	fieldType := normalizeQuickFactFieldType(template.FieldType)
	fact := &models.QuickFact{
		VaultQuickFactsTemplateID: template.ID,
		ContactID:                 contactID,
	}
	switch fieldType {
	case QuickFactFieldText:
		value := strings.TrimSpace(legacyContent)
		if valueText != nil {
			value = strings.TrimSpace(*valueText)
		}
		if value == "" && template.DefaultValue != nil {
			value = strings.TrimSpace(*template.DefaultValue)
		}
		if value == "" && template.Required {
			return nil, ErrQuickFactRequiredValue
		}
		fact.Content = value
		if value != "" {
			fact.ValueText = &value
		}
	case QuickFactFieldNumber:
		value := valueNumber
		if value == nil && strings.TrimSpace(legacyContent) != "" {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(legacyContent), 64)
			if err != nil {
				return nil, ErrQuickFactInvalidValue
			}
			value = &parsed
		}
		if value == nil && template.DefaultValue != nil && strings.TrimSpace(*template.DefaultValue) != "" {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(*template.DefaultValue), 64)
			if err != nil {
				return nil, ErrQuickFactInvalidValue
			}
			value = &parsed
		}
		if value == nil && template.Required {
			return nil, ErrQuickFactRequiredValue
		}
		fact.ValueNumber = value
		if value != nil {
			fact.Content = strconv.FormatFloat(*value, 'f', -1, 64)
		}
	case QuickFactFieldDate:
		value := strings.TrimSpace(legacyContent)
		if valueDate != nil {
			value = strings.TrimSpace(*valueDate)
		}
		if value == "" && template.DefaultValue != nil {
			value = strings.TrimSpace(*template.DefaultValue)
		}
		if value == "" && template.Required {
			return nil, ErrQuickFactRequiredValue
		}
		if value != "" {
			if err := validateQuickFactDate(value); err != nil {
				return nil, err
			}
			fact.ValueDate = &value
		}
		fact.Content = value
	case QuickFactFieldSelect:
		value := strings.TrimSpace(legacyContent)
		if valueOption != nil {
			value = strings.TrimSpace(*valueOption)
		}
		if value == "" && template.DefaultValue != nil {
			value = strings.TrimSpace(*template.DefaultValue)
		}
		if value == "" && template.Required {
			return nil, ErrQuickFactRequiredValue
		}
		if value != "" && !quickFactOptionAllowed(value, parseQuickFactSelectOptions(template.SelectOptions)) {
			return nil, ErrQuickFactInvalidValue
		}
		fact.Content = value
		if value != "" {
			fact.ValueOption = &value
		}
	case QuickFactFieldPhoto, QuickFactFieldDocument:
		return nil, ErrQuickFactInvalidField
	default:
		return nil, ErrQuickFactInvalidField
	}
	return fact, nil
}

func validateQuickFactTemplateInput(fieldType string, options []string, defaultValue *string) (string, *string, error) {
	normalized := normalizeQuickFactFieldType(fieldType)
	if !isValidQuickFactFieldType(normalized) {
		return "", nil, ErrQuickFactInvalidField
	}
	cleanOptions := cleanQuickFactSelectOptions(options)
	if normalized == QuickFactFieldSelect {
		if len(cleanOptions) == 0 {
			return "", nil, ErrQuickFactInvalidValue
		}
		if defaultValue != nil && strings.TrimSpace(*defaultValue) != "" && !quickFactOptionAllowed(strings.TrimSpace(*defaultValue), cleanOptions) {
			return "", nil, ErrQuickFactInvalidValue
		}
	} else if len(cleanOptions) > 0 {
		return "", nil, ErrQuickFactInvalidField
	}
	if normalized == QuickFactFieldDate && defaultValue != nil && strings.TrimSpace(*defaultValue) != "" {
		if err := validateQuickFactDate(strings.TrimSpace(*defaultValue)); err != nil {
			return "", nil, err
		}
	}
	if normalized == QuickFactFieldNumber && defaultValue != nil && strings.TrimSpace(*defaultValue) != "" {
		if _, err := strconv.ParseFloat(strings.TrimSpace(*defaultValue), 64); err != nil {
			return "", nil, ErrQuickFactInvalidValue
		}
	}
	encoded := encodeQuickFactSelectOptions(cleanOptions)
	return normalized, encoded, nil
}

func normalizeQuickFactFieldType(fieldType string) string {
	fieldType = strings.TrimSpace(fieldType)
	if fieldType == "" {
		return QuickFactFieldText
	}
	return fieldType
}

func isValidQuickFactFieldType(fieldType string) bool {
	switch fieldType {
	case QuickFactFieldText, QuickFactFieldNumber, QuickFactFieldDate, QuickFactFieldSelect, QuickFactFieldPhoto, QuickFactFieldDocument:
		return true
	default:
		return false
	}
}

func isQuickFactFileField(fieldType string) bool {
	fieldType = normalizeQuickFactFieldType(fieldType)
	return fieldType == QuickFactFieldPhoto || fieldType == QuickFactFieldDocument
}

func quickFactFileType(fieldType string) string {
	if normalizeQuickFactFieldType(fieldType) == QuickFactFieldPhoto {
		return "photo"
	}
	return "document"
}

func quickFactMimeMatchesFieldType(mimeType string, fieldType string) bool {
	if normalizeQuickFactFieldType(fieldType) == QuickFactFieldPhoto {
		return mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/gif" || mimeType == "image/webp"
	}
	return mimeType == "application/pdf" || mimeType == "text/plain" || mimeType == "application/msword" || mimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

func validateQuickFactDate(value string) error {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return ErrQuickFactInvalidValue
	}
	return nil
}

func cleanQuickFactSelectOptions(options []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" || seen[option] {
			continue
		}
		seen[option] = true
		cleaned = append(cleaned, option)
	}
	return cleaned
}

func encodeQuickFactSelectOptions(options []string) *string {
	if len(options) == 0 {
		return nil
	}
	encoded := strings.Join(options, "\n")
	return &encoded
}

func parseQuickFactSelectOptions(encoded *string) []string {
	if encoded == nil || strings.TrimSpace(*encoded) == "" {
		return nil
	}
	return cleanQuickFactSelectOptions(strings.Split(*encoded, "\n"))
}

func quickFactOptionAllowed(value string, options []string) bool {
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}

func toQuickFactResponse(f *models.QuickFact) dto.QuickFactResponse {
	fieldType := normalizeQuickFactFieldType(f.VaultQuickFactsTemplate.FieldType)
	resp := dto.QuickFactResponse{
		ID:                        f.ID,
		VaultQuickFactsTemplateID: f.VaultQuickFactsTemplateID,
		ContactID:                 f.ContactID,
		Content:                   f.Content,
		FieldType:                 fieldType,
		ValueText:                 f.ValueText,
		ValueNumber:               f.ValueNumber,
		ValueDate:                 f.ValueDate,
		ValueOption:               f.ValueOption,
		FileID:                    f.FileID,
		CreatedAt:                 f.CreatedAt,
		UpdatedAt:                 f.UpdatedAt,
	}
	if f.File != nil && f.File.ID != 0 {
		resp.File = &dto.QuickFactFileResponse{
			ID:        f.File.ID,
			Name:      f.File.Name,
			MimeType:  f.File.MimeType,
			Type:      f.File.Type,
			Size:      f.File.Size,
			CreatedAt: f.File.CreatedAt,
			UpdatedAt: f.File.UpdatedAt,
		}
	}
	return resp
}
