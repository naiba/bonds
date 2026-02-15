package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ContactSortService struct {
	db *gorm.DB
}

func NewContactSortService(db *gorm.DB) *ContactSortService {
	return &ContactSortService{db: db}
}

func (s *ContactSortService) UpdateSort(userID string, req dto.UpdateContactSortRequest) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	user.ContactSortOrder = req.SortOrder
	return s.db.Save(&user).Error
}
