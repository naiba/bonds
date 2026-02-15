package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func (s *ReportService) AddressesByCity(vaultID, city string) ([]dto.AddressContactItem, error) {
	type row struct {
		ContactID string  `gorm:"column:contact_id"`
		FirstName *string `gorm:"column:first_name"`
		LastName  *string `gorm:"column:last_name"`
		City      *string `gorm:"column:city"`
		Province  *string `gorm:"column:province"`
		Country   *string `gorm:"column:country"`
	}

	var rows []row
	err := s.db.Model(&models.Address{}).
		Select("contact_address.contact_id, contacts.first_name, contacts.last_name, addresses.city, addresses.province, addresses.country").
		Joins("JOIN contact_address ON contact_address.address_id = addresses.id").
		Joins("JOIN contacts ON contacts.id = contact_address.contact_id").
		Where("addresses.vault_id = ? AND addresses.city = ?", vaultID, city).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.AddressContactItem, len(rows))
	for i, r := range rows {
		result[i] = dto.AddressContactItem{
			ContactID: r.ContactID,
			FirstName: ptrToStr(r.FirstName),
			LastName:  ptrToStr(r.LastName),
			City:      ptrToStr(r.City),
			Province:  ptrToStr(r.Province),
			Country:   ptrToStr(r.Country),
		}
	}
	return result, nil
}

func (s *ReportService) AddressesByCountry(vaultID, country string) ([]dto.AddressContactItem, error) {
	type row struct {
		ContactID string  `gorm:"column:contact_id"`
		FirstName *string `gorm:"column:first_name"`
		LastName  *string `gorm:"column:last_name"`
		City      *string `gorm:"column:city"`
		Province  *string `gorm:"column:province"`
		Country   *string `gorm:"column:country"`
	}

	var rows []row
	err := s.db.Model(&models.Address{}).
		Select("contact_address.contact_id, contacts.first_name, contacts.last_name, addresses.city, addresses.province, addresses.country").
		Joins("JOIN contact_address ON contact_address.address_id = addresses.id").
		Joins("JOIN contacts ON contacts.id = contact_address.contact_id").
		Where("addresses.vault_id = ? AND addresses.country = ?", vaultID, country).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]dto.AddressContactItem, len(rows))
	for i, r := range rows {
		result[i] = dto.AddressContactItem{
			ContactID: r.ContactID,
			FirstName: ptrToStr(r.FirstName),
			LastName:  ptrToStr(r.LastName),
			City:      ptrToStr(r.City),
			Province:  ptrToStr(r.Province),
			Country:   ptrToStr(r.Country),
		}
	}
	return result, nil
}
