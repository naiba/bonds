package services

import (
	"errors"
	"fmt"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

func (s *AddressService) GetMapImageURL(addressID uint, contactID, vaultID string, width, height int) (string, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return "", err
	}
	var address models.Address
	if err := s.db.Where("id = ?", addressID).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrAddressNotFound
		}
		return "", err
	}

	if address.Latitude == nil || address.Longitude == nil {
		return "", errors.New("address has no coordinates")
	}

	mapURL := fmt.Sprintf(
		"https://www.openstreetmap.org/export/embed.html?bbox=%f,%f,%f,%f&layer=mapnik&marker=%f,%f",
		*address.Longitude-0.005, *address.Latitude-0.005,
		*address.Longitude+0.005, *address.Latitude+0.005,
		*address.Latitude, *address.Longitude,
	)
	return mapURL, nil
}
