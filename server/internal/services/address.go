package services

import (
	"errors"
	"time"

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

func (s *AddressService) List(contactID, vaultID string) ([]dto.AddressResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var pivots []models.ContactAddress
	if err := s.db.Where("contact_id = ?", contactID).Find(&pivots).Error; err != nil {
		return nil, err
	}
	if len(pivots) == 0 {
		return []dto.AddressResponse{}, nil
	}

	addressIDs := make([]uint, len(pivots))
	pivotByAddr := make(map[uint]models.ContactAddress)
	for i, p := range pivots {
		addressIDs[i] = p.AddressID
		pivotByAddr[p.AddressID] = p
	}

	var addresses []models.Address
	if err := s.db.Where("id IN ?", addressIDs).Find(&addresses).Error; err != nil {
		return nil, err
	}

	result := make([]dto.AddressResponse, len(addresses))
	for i, a := range addresses {
		p := pivotByAddr[a.ID]
		result[i] = toAddressResponse(&a, p.IsPastAddress, p.DateFrom, p.DateTo)
	}
	return result, nil
}

func (s *AddressService) Create(contactID, vaultID string, req dto.CreateAddressRequest) (*dto.AddressResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
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

	// A non-null DateTo implies the contact has moved out — auto-flip
	// IsPastAddress to true so the two fields can't disagree silently.
	isPast := req.IsPastAddress || req.DateTo != nil

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&address).Error; err != nil {
			return err
		}
		pivot := models.ContactAddress{
			ContactID:     contactID,
			AddressID:     address.ID,
			IsPastAddress: isPast,
			DateFrom:      req.DateFrom,
			DateTo:        req.DateTo,
		}
		// Honor the GORM zero-value-bool trap for the false-explicit case
		// (per AGENTS.md). Create skips false; a separate Update locks it in.
		if err := tx.Create(&pivot).Error; err != nil {
			return err
		}
		if !isPast {
			return tx.Model(&pivot).Update("is_past_address", false).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.tryGeocode(&address)

	if s.feedRecorder != nil {
		entityType := "Address"
		s.feedRecorder.Record(contactID, "", ActionAddressAdded, "Added an address", &address.ID, &entityType)
	}

	resp := toAddressResponse(&address, isPast, req.DateFrom, req.DateTo)
	return &resp, nil
}

func (s *AddressService) Update(id uint, contactID, vaultID string, req dto.UpdateAddressRequest) (*dto.AddressResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
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

	isPast := req.IsPastAddress || req.DateTo != nil
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&address).Error; err != nil {
			return err
		}
		pivot.IsPastAddress = isPast
		pivot.DateFrom = req.DateFrom
		pivot.DateTo = req.DateTo
		return tx.Save(&pivot).Error
	})
	if err != nil {
		return nil, err
	}

	resp := toAddressResponse(&address, isPast, req.DateFrom, req.DateTo)
	return &resp, nil
}

func (s *AddressService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
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

func toAddressResponse(a *models.Address, isPastAddress bool, dateFrom, dateTo *time.Time) dto.AddressResponse {
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
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}
