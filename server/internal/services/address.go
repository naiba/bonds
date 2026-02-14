package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrAddressNotFound = errors.New("address not found")

type AddressService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
	geocoder     Geocoder
}

func NewAddressService(db *gorm.DB) *AddressService {
	return &AddressService{db: db}
}

func (s *AddressService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *AddressService) SetGeocoder(g Geocoder) {
	s.geocoder = g
}

func (s *AddressService) List(contactID string) ([]dto.AddressResponse, error) {
	var pivots []models.ContactAddress
	if err := s.db.Where("contact_id = ?", contactID).Find(&pivots).Error; err != nil {
		return nil, err
	}
	if len(pivots) == 0 {
		return []dto.AddressResponse{}, nil
	}

	addressIDs := make([]uint, len(pivots))
	pastMap := make(map[uint]bool)
	for i, p := range pivots {
		addressIDs[i] = p.AddressID
		pastMap[p.AddressID] = p.IsPastAddress
	}

	var addresses []models.Address
	if err := s.db.Where("id IN ?", addressIDs).Find(&addresses).Error; err != nil {
		return nil, err
	}

	result := make([]dto.AddressResponse, len(addresses))
	for i, a := range addresses {
		result[i] = toAddressResponse(&a, pastMap[a.ID])
	}
	return result, nil
}

func (s *AddressService) Create(contactID, vaultID string, req dto.CreateAddressRequest) (*dto.AddressResponse, error) {
	address := models.Address{
		VaultID:       vaultID,
		Line1:         strPtrOrNil(req.Line1),
		Line2:         strPtrOrNil(req.Line2),
		City:          strPtrOrNil(req.City),
		Province:      strPtrOrNil(req.Province),
		PostalCode:    strPtrOrNil(req.PostalCode),
		Country:       strPtrOrNil(req.Country),
		AddressTypeID: req.AddressTypeID,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&address).Error; err != nil {
			return err
		}
		pivot := models.ContactAddress{
			ContactID:     contactID,
			AddressID:     address.ID,
			IsPastAddress: req.IsPastAddress,
		}
		return tx.Create(&pivot).Error
	})
	if err != nil {
		return nil, err
	}

	s.tryGeocode(&address)

	if s.feedRecorder != nil {
		entityType := "Address"
		s.feedRecorder.Record(contactID, "", ActionAddressAdded, "Added an address", &address.ID, &entityType)
	}

	resp := toAddressResponse(&address, req.IsPastAddress)
	return &resp, nil
}

func (s *AddressService) Update(id uint, contactID string, req dto.UpdateAddressRequest) (*dto.AddressResponse, error) {
	var pivot models.ContactAddress
	if err := s.db.Where("address_id = ? AND contact_id = ?", id, contactID).First(&pivot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAddressNotFound
		}
		return nil, err
	}

	var address models.Address
	if err := s.db.First(&address, id).Error; err != nil {
		return nil, ErrAddressNotFound
	}

	address.Line1 = strPtrOrNil(req.Line1)
	address.Line2 = strPtrOrNil(req.Line2)
	address.City = strPtrOrNil(req.City)
	address.Province = strPtrOrNil(req.Province)
	address.PostalCode = strPtrOrNil(req.PostalCode)
	address.Country = strPtrOrNil(req.Country)
	address.AddressTypeID = req.AddressTypeID
	address.Latitude = req.Latitude
	address.Longitude = req.Longitude

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&address).Error; err != nil {
			return err
		}
		pivot.IsPastAddress = req.IsPastAddress
		return tx.Save(&pivot).Error
	})
	if err != nil {
		return nil, err
	}

	resp := toAddressResponse(&address, req.IsPastAddress)
	return &resp, nil
}

func (s *AddressService) Delete(id uint, contactID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("address_id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactAddress{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAddressNotFound
		}
		return tx.Where("id = ?", id).Delete(&models.Address{}).Error
	})
}

func (s *AddressService) tryGeocode(address *models.Address) {
	if s.geocoder == nil {
		return
	}
	parts := []string{}
	for _, p := range []*string{address.Line1, address.City, address.Province, address.PostalCode, address.Country} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	if len(parts) == 0 {
		return
	}
	query := ""
	for i, p := range parts {
		if i > 0 {
			query += ", "
		}
		query += p
	}
	result, err := s.geocoder.Geocode(query)
	if err != nil || result == nil {
		return
	}
	address.Latitude = &result.Latitude
	address.Longitude = &result.Longitude
	s.db.Model(address).Select("latitude", "longitude").Updates(address)
}

func toAddressResponse(a *models.Address, isPastAddress bool) dto.AddressResponse {
	return dto.AddressResponse{
		ID:            a.ID,
		VaultID:       a.VaultID,
		Line1:         ptrToStr(a.Line1),
		Line2:         ptrToStr(a.Line2),
		City:          ptrToStr(a.City),
		Province:      ptrToStr(a.Province),
		PostalCode:    ptrToStr(a.PostalCode),
		Country:       ptrToStr(a.Country),
		AddressTypeID: a.AddressTypeID,
		Latitude:      a.Latitude,
		Longitude:     a.Longitude,
		IsPastAddress: isPastAddress,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}
