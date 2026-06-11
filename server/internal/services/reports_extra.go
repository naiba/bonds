package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func (s *ReportService) AddressesByCity(vaultID, city, userID string) ([]dto.AddressContactItem, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	type row struct {
		ContactID  string  `gorm:"column:contact_id"`
		VaultID    string  `gorm:"column:vault_id"`
		FirstName  *string `gorm:"column:first_name"`
		LastName   *string `gorm:"column:last_name"`
		MiddleName *string `gorm:"column:middle_name"`
		Nickname   *string `gorm:"column:nickname"`
		MaidenName *string `gorm:"column:maiden_name"`
		Prefix     *string `gorm:"column:prefix"`
		Suffix     *string `gorm:"column:suffix"`
		City       *string `gorm:"column:city"`
		Province   *string `gorm:"column:province"`
		Country    *string `gorm:"column:country"`
	}

	var rows []row
	queryErr := s.db.Model(&models.Address{}).
		Select("contact_address.contact_id, contacts.vault_id, contacts.first_name, contacts.last_name, contacts.middle_name, contacts.nickname, contacts.maiden_name, contacts.prefix, contacts.suffix, addresses.city, addresses.province, addresses.country").
		Joins("JOIN contact_address ON contact_address.address_id = addresses.id").
		Joins("JOIN contacts ON contacts.id = contact_address.contact_id").
		Where("addresses.vault_id = ? AND addresses.city = ?", vaultID, city).
		Scan(&rows).Error
	if queryErr != nil {
		return nil, queryErr
	}

	result := make([]dto.AddressContactItem, len(rows))
	for i, r := range rows {
		contact := models.Contact{VaultID: r.VaultID, FirstName: r.FirstName, LastName: r.LastName, MiddleName: r.MiddleName, Nickname: r.Nickname, MaidenName: r.MaidenName, Prefix: r.Prefix, Suffix: r.Suffix}
		contactName, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		result[i] = dto.AddressContactItem{
			ContactID:   r.ContactID,
			ContactName: contactName,
			FirstName:   ptrToStr(r.FirstName),
			LastName:    ptrToStr(r.LastName),
			MiddleName:  ptrToStr(r.MiddleName),
			Nickname:    ptrToStr(r.Nickname),
			MaidenName:  ptrToStr(r.MaidenName),
			Prefix:      ptrToStr(r.Prefix),
			Suffix:      ptrToStr(r.Suffix),
			City:        ptrToStr(r.City),
			Province:    ptrToStr(r.Province),
			Country:     ptrToStr(r.Country),
		}
	}
	return result, nil
}

func (s *ReportService) AddressesByCountry(vaultID, country, userID string) ([]dto.AddressContactItem, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	type row struct {
		ContactID  string  `gorm:"column:contact_id"`
		VaultID    string  `gorm:"column:vault_id"`
		FirstName  *string `gorm:"column:first_name"`
		LastName   *string `gorm:"column:last_name"`
		MiddleName *string `gorm:"column:middle_name"`
		Nickname   *string `gorm:"column:nickname"`
		MaidenName *string `gorm:"column:maiden_name"`
		Prefix     *string `gorm:"column:prefix"`
		Suffix     *string `gorm:"column:suffix"`
		City       *string `gorm:"column:city"`
		Province   *string `gorm:"column:province"`
		Country    *string `gorm:"column:country"`
	}

	var rows []row
	queryErr := s.db.Model(&models.Address{}).
		Select("contact_address.contact_id, contacts.vault_id, contacts.first_name, contacts.last_name, contacts.middle_name, contacts.nickname, contacts.maiden_name, contacts.prefix, contacts.suffix, addresses.city, addresses.province, addresses.country").
		Joins("JOIN contact_address ON contact_address.address_id = addresses.id").
		Joins("JOIN contacts ON contacts.id = contact_address.contact_id").
		Where("addresses.vault_id = ? AND addresses.country = ?", vaultID, country).
		Scan(&rows).Error
	if queryErr != nil {
		return nil, queryErr
	}

	result := make([]dto.AddressContactItem, len(rows))
	for i, r := range rows {
		contact := models.Contact{VaultID: r.VaultID, FirstName: r.FirstName, LastName: r.LastName, MiddleName: r.MiddleName, Nickname: r.Nickname, MaidenName: r.MaidenName, Prefix: r.Prefix, Suffix: r.Suffix}
		contactName, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		result[i] = dto.AddressContactItem{
			ContactID:   r.ContactID,
			ContactName: contactName,
			FirstName:   ptrToStr(r.FirstName),
			LastName:    ptrToStr(r.LastName),
			MiddleName:  ptrToStr(r.MiddleName),
			Nickname:    ptrToStr(r.Nickname),
			MaidenName:  ptrToStr(r.MaidenName),
			Prefix:      ptrToStr(r.Prefix),
			Suffix:      ptrToStr(r.Suffix),
			City:        ptrToStr(r.City),
			Province:    ptrToStr(r.Province),
			Country:     ptrToStr(r.Country),
		}
	}
	return result, nil
}
