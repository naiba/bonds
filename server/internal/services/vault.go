package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrVaultNotFound    = errors.New("vault not found")
	ErrVaultForbidden   = errors.New("vault access forbidden")
	ErrInsufficientPerm = errors.New("insufficient permissions")
)

type VaultService struct {
	db *gorm.DB
}

func NewVaultService(db *gorm.DB) *VaultService {
	return &VaultService{db: db}
}

func (s *VaultService) ListVaults(userID string) ([]dto.VaultResponse, error) {
	var userVaults []models.UserVault
	if err := s.db.Where("user_id = ?", userID).Find(&userVaults).Error; err != nil {
		return nil, err
	}

	vaultIDs := make([]string, len(userVaults))
	for i, uv := range userVaults {
		vaultIDs[i] = uv.VaultID
	}

	if len(vaultIDs) == 0 {
		return []dto.VaultResponse{}, nil
	}

	var vaults []models.Vault
	if err := s.db.Where("id IN ?", vaultIDs).Find(&vaults).Error; err != nil {
		return nil, err
	}

	result := make([]dto.VaultResponse, len(vaults))
	for i, v := range vaults {
		result[i] = toVaultResponse(&v)
	}
	return result, nil
}

func (s *VaultService) CreateVault(accountID, userID string, req dto.CreateVaultRequest, locale string) (*dto.VaultResponse, error) {
	desc := req.Description
	vault := models.Vault{
		AccountID:   accountID,
		Name:        req.Name,
		Description: &desc,
		Type:        "personal",
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&vault).Error; err != nil {
			return err
		}
		userVault := models.UserVault{
			UserID:     userID,
			VaultID:    vault.ID,
			Permission: models.PermissionManager,
		}
		if err := tx.Create(&userVault).Error; err != nil {
			return err
		}
		return models.SeedVaultDefaults(tx, vault.ID, locale)
	})
	if err != nil {
		return nil, err
	}

	resp := toVaultResponse(&vault)
	return &resp, nil
}

func (s *VaultService) GetVault(vaultID string) (*dto.VaultResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultNotFound
		}
		return nil, err
	}
	resp := toVaultResponse(&vault)
	return &resp, nil
}

func (s *VaultService) UpdateVault(vaultID string, req dto.UpdateVaultRequest) (*dto.VaultResponse, error) {
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

	resp := toVaultResponse(&vault)
	return &resp, nil
}

func (s *VaultService) DeleteVault(vaultID string) error {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultNotFound
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("vault_id = ?", vaultID).Delete(&models.UserVault{}).Error; err != nil {
			return err
		}
		if err := tx.Where("vault_id = ?", vaultID).Delete(&models.Contact{}).Error; err != nil {
			return err
		}
		return tx.Delete(&vault).Error
	})
}

func (s *VaultService) CheckUserVaultAccess(userID, vaultID string, requiredPerm int) error {
	var uv models.UserVault
	if err := s.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultForbidden
		}
		return err
	}
	if uv.Permission > requiredPerm {
		return ErrInsufficientPerm
	}
	return nil
}

func toVaultResponse(v *models.Vault) dto.VaultResponse {
	desc := ""
	if v.Description != nil {
		desc = *v.Description
	}
	return dto.VaultResponse{
		ID:          v.ID,
		AccountID:   v.AccountID,
		Name:        v.Name,
		Description: desc,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}
