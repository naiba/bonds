package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultUsersService struct {
	db *gorm.DB
}

func NewVaultUsersService(db *gorm.DB) *VaultUsersService {
	return &VaultUsersService{db: db}
}

func (s *VaultUsersService) List(vaultID string) ([]dto.VaultUserResponse, error) {
	var uvs []models.UserVault
	if err := s.db.Where("vault_id = ?", vaultID).Find(&uvs).Error; err != nil {
		return nil, err
	}
	result := make([]dto.VaultUserResponse, 0, len(uvs))
	for _, uv := range uvs {
		var user models.User
		if err := s.db.First(&user, "id = ?", uv.UserID).Error; err != nil {
			continue
		}
		result = append(result, toVaultUserResponse(&uv, &user))
	}
	return result, nil
}

func (s *VaultUsersService) Add(vaultID string, req dto.AddVaultUserRequest) (*dto.VaultUserResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserEmailNotFound
		}
		return nil, err
	}

	// SECURITY: 防止跨 Account 拉人——目标用户必须与 vault 属于同一 Account。
	var vault models.Vault
	if err := s.db.Select("account_id").First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, err
	}
	if user.AccountID != vault.AccountID {
		return nil, ErrUserEmailNotFound
	}

	var existing models.UserVault
	err := s.db.Where("user_id = ? AND vault_id = ?", user.ID, vaultID).First(&existing).Error
	if err == nil {
		return nil, ErrUserAlreadyInVault
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	uv := models.UserVault{
		UserID:     user.ID,
		VaultID:    vaultID,
		Permission: req.Permission,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&uv).Error; err != nil {
			return err
		}
		return scheduleAllVaultUserRemindersForNewMember(tx, vaultID, user.ID, time.Now())
	}); err != nil {
		return nil, err
	}
	resp := toVaultUserResponse(&uv, &user)
	return &resp, nil
}

func (s *VaultUsersService) UpdatePermission(id uint, vaultID string, req dto.UpdateVaultUserPermRequest) (*dto.VaultUserResponse, error) {
	var uv models.UserVault
	var user models.User
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockVaultMemberships(tx, vaultID); err != nil {
			return err
		}
		if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&uv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVaultUserNotFound
			}
			return err
		}
		if uv.Permission == models.PermissionManager && req.Permission != models.PermissionManager {
			if err := ensureAnotherVaultManager(tx, vaultID, uv.ID); err != nil {
				return err
			}
		}
		if err := tx.Model(&uv).Update("permission", req.Permission).Error; err != nil {
			return err
		}
		uv.Permission = req.Permission
		return tx.First(&user, "id = ?", uv.UserID).Error
	}); err != nil {
		return nil, err
	}
	resp := toVaultUserResponse(&uv, &user)
	return &resp, nil
}

func (s *VaultUsersService) Remove(id uint, vaultID, currentUserID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockVaultMemberships(tx, vaultID); err != nil {
			return err
		}
		var uv models.UserVault
		if err := tx.Where("id = ? AND vault_id = ?", id, vaultID).First(&uv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVaultUserNotFound
			}
			return err
		}
		if uv.UserID == currentUserID {
			return ErrCannotRemoveSelf
		}
		if uv.Permission == models.PermissionManager {
			if err := ensureAnotherVaultManager(tx, vaultID, uv.ID); err != nil {
				return err
			}
		}
		return tx.Delete(&uv).Error
	})
}

func toVaultUserResponse(uv *models.UserVault, u *models.User) dto.VaultUserResponse {
	return dto.VaultUserResponse{
		ID:         uv.ID,
		UserID:     uv.UserID,
		Email:      u.Email,
		FirstName:  ptrToStr(u.FirstName),
		LastName:   ptrToStr(u.LastName),
		Disabled:   u.Disabled,
		Permission: uv.Permission,
	}
}
