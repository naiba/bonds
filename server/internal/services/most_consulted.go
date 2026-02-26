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
		MiddleName    *string `gorm:"column:middle_name"`
		Nickname      *string `gorm:"column:nickname"`
		MaidenName    *string `gorm:"column:maiden_name"`
		Prefix        *string `gorm:"column:prefix"`
		Suffix        *string `gorm:"column:suffix"`
		NumberOfViews int     `gorm:"column:number_of_views"`
	}

	var rows []row
	err := s.db.Model(&models.ContactVaultUser{}).
		Select("contact_vault_user.contact_id, contacts.first_name, contacts.last_name, contacts.middle_name, contacts.nickname, contacts.maiden_name, contacts.prefix, contacts.suffix, contact_vault_user.number_of_views").
		Joins("JOIN contacts ON contacts.id = contact_vault_user.contact_id").
		Where("contact_vault_user.vault_id = ? AND contact_vault_user.user_id = ? AND contact_vault_user.number_of_views > 0", vaultID, userID).
		// 手动 JOIN 不会触发 GORM 软删除自动过滤，必须显式排除已删除和已归档的联系人。
		Where("contacts.deleted_at IS NULL AND contacts.listed = ?", true).
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
			MiddleName:    ptrToStr(r.MiddleName),
			Nickname:      ptrToStr(r.Nickname),
			MaidenName:    ptrToStr(r.MaidenName),
			Prefix:        ptrToStr(r.Prefix),
			Suffix:        ptrToStr(r.Suffix),
			NumberOfViews: r.NumberOfViews,
		}
	}
	return result, nil
}
