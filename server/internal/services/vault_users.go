package services

import (
	"errors"

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
	if err := s.db.Create(&uv).Error; err != nil {
		return nil, err
	}
	resp := toVaultUserResponse(&uv, &user)
	return &resp, nil
}

func (s *VaultUsersService) UpdatePermission(id uint, vaultID string, req dto.UpdateVaultUserPermRequest) (*dto.VaultUserResponse, error) {
	var uv models.UserVault
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&uv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVaultUserNotFound
		}
		return nil, err
	}
	uv.Permission = req.Permission
	if err := s.db.Save(&uv).Error; err != nil {
		return nil, err
	}
	var user models.User
	if err := s.db.First(&user, "id = ?", uv.UserID).Error; err != nil {
		return nil, err
	}
	resp := toVaultUserResponse(&uv, &user)
	return &resp, nil
}

func (s *VaultUsersService) Remove(id uint, vaultID, currentUserID string) error {
	var uv models.UserVault
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&uv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVaultUserNotFound
		}
		return err
	}
	if uv.UserID == currentUserID {
		return ErrCannotRemoveSelf
	}
	return s.db.Delete(&uv).Error
}

func toVaultUserResponse(uv *models.UserVault, u *models.User) dto.VaultUserResponse {
	return dto.VaultUserResponse{
		ID:         uv.ID,
		UserID:     uv.UserID,
		Email:      u.Email,
		FirstName:  ptrToStr(u.FirstName),
		LastName:   ptrToStr(u.LastName),
		Permission: uv.Permission,
	}
}
