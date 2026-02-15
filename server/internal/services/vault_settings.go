package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrVaultUserNotFound    = errors.New("vault user not found")
	ErrUserEmailNotFound    = errors.New("user with this email not found")
	ErrUserAlreadyInVault   = errors.New("user already in vault")
	ErrCannotRemoveSelf     = errors.New("cannot remove yourself from vault")
	ErrLabelNotFound        = errors.New("label not found")
	ErrDateTypeNotFound     = errors.New("important date type not found")
	ErrCannotDeleteDefault  = errors.New("cannot delete default item")
	ErrMoodParamNotFound    = errors.New("mood tracking parameter not found")
	ErrLifeCategoryNotFound = errors.New("life event category not found")
	ErrLifeTypeNotFound     = errors.New("life event type not found")
	ErrQuickFactTplNotFound = errors.New("quick fact template not found")
)

type VaultSettingsService struct {
	db *gorm.DB
}

func NewVaultSettingsService(db *gorm.DB) *VaultSettingsService {
	return &VaultSettingsService{db: db}
}

func (s *VaultSettingsService) Get(vaultID string) (*dto.VaultSettingsResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	return toVaultSettingsResponse(&vault), nil
}

func (s *VaultSettingsService) Update(vaultID string, req dto.UpdateVaultSettingsRequest) (*dto.VaultSettingsResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	vault.Name = req.Name
	desc := req.Description
	vault.Description = &desc
	if err := s.db.Save(&vault).Error; err != nil {
		return nil, err
	}
	return toVaultSettingsResponse(&vault), nil
}

func (s *VaultSettingsService) UpdateVisibility(vaultID string, req dto.UpdateTabVisibilityRequest) (*dto.VaultSettingsResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if req.ShowGroupTab != nil {
		updates["show_group_tab"] = *req.ShowGroupTab
	}
	if req.ShowTasksTab != nil {
		updates["show_tasks_tab"] = *req.ShowTasksTab
	}
	if req.ShowFilesTab != nil {
		updates["show_files_tab"] = *req.ShowFilesTab
	}
	if req.ShowJournalTab != nil {
		updates["show_journal_tab"] = *req.ShowJournalTab
	}
	if req.ShowCompaniesTab != nil {
		updates["show_companies_tab"] = *req.ShowCompaniesTab
	}
	if req.ShowReportsTab != nil {
		updates["show_reports_tab"] = *req.ShowReportsTab
	}
	if req.ShowCalendarTab != nil {
		updates["show_calendar_tab"] = *req.ShowCalendarTab
	}
	if len(updates) > 0 {
		if err := s.db.Model(&vault).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
			return nil, err
		}
	}
	return toVaultSettingsResponse(&vault), nil
}

func (s *VaultSettingsService) UpdateDefaultTemplate(vaultID string, req dto.UpdateDefaultTemplateRequest) (*dto.VaultSettingsResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	if err := s.db.Model(&vault).Update("default_template_id", req.DefaultTemplateID).Error; err != nil {
		return nil, err
	}
	vault.DefaultTemplateID = req.DefaultTemplateID
	return toVaultSettingsResponse(&vault), nil
}

func toVaultSettingsResponse(v *models.Vault) *dto.VaultSettingsResponse {
	desc := ""
	if v.Description != nil {
		desc = *v.Description
	}
	return &dto.VaultSettingsResponse{
		ID:                v.ID,
		Name:              v.Name,
		Description:       desc,
		DefaultTemplateID: v.DefaultTemplateID,
		ShowGroupTab:      v.ShowGroupTab,
		ShowTasksTab:      v.ShowTasksTab,
		ShowFilesTab:      v.ShowFilesTab,
		ShowJournalTab:    v.ShowJournalTab,
		ShowCompaniesTab:  v.ShowCompaniesTab,
		ShowReportsTab:    v.ShowReportsTab,
		ShowCalendarTab:   v.ShowCalendarTab,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}
