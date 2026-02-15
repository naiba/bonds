package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *UserManagementService) Get(id, accountID string) (*dto.UserManagementResponse, error) {
	var user models.User
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrManagedUserNotFound
		}
		return nil, err
	}
	resp := toUserManagementResponse(&user)
	return &resp, nil
}
