package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type MostConsultedService struct {
	db *gorm.DB
}

func NewMostConsultedService(db *gorm.DB) *MostConsultedService {
	return &MostConsultedService{db: db}
}

func (s *MostConsultedService) List(vaultID, userID string) ([]dto.MostConsultedContactItem, error) {
	type row struct {
		ContactID     string  `gorm:"column:contact_id"`
		FirstName     *string `gorm:"column:first_name"`
		LastName      *string `gorm:"column:last_name"`
		NumberOfViews int     `gorm:"column:number_of_views"`
	}

	var rows []row
	err := s.db.Model(&models.ContactVaultUser{}).
		Select("contact_vault_user.contact_id, contacts.first_name, contacts.last_name, contact_vault_user.number_of_views").
		Joins("JOIN contacts ON contacts.id = contact_vault_user.contact_id").
		Where("contact_vault_user.vault_id = ? AND contact_vault_user.user_id = ? AND contact_vault_user.number_of_views > 0", vaultID, userID).
		Order("contact_vault_user.number_of_views DESC").
		Limit(10).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.MostConsultedContactItem, len(rows))
	for i, r := range rows {
		result[i] = dto.MostConsultedContactItem{
			ContactID:     r.ContactID,
			FirstName:     ptrToStr(r.FirstName),
			LastName:      ptrToStr(r.LastName),
			NumberOfViews: r.NumberOfViews,
		}
	}
	return result, nil
}
