package services

import (
	"sort"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

// MapReport returns everything needed to draw the vault on a map.
//
// It answers in two resolutions on purpose. Points are the geocoded addresses,
// which is the map most people picture — but geocoding is optional, off by
// default, and never backfilled, so a vault imported from a phone address book
// has none of them. Countries is derived from the address text alone and is
// therefore always populated, which lets the page draw a country choropleth
// even when not a single coordinate exists.
//
// Addresses are counted the same way the Overview stat card counts them (every
// address in the vault, listed or not) so the two numbers reconcile.
func (s *ReportService) MapReport(vaultID, userID string) (*dto.MapReportResponse, error) {
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	var addresses []models.Address
	if err := s.db.Where("vault_id = ?", vaultID).Find(&addresses).Error; err != nil {
		return nil, err
	}

	response := &dto.MapReportResponse{
		TotalAddresses: len(addresses),
		Points:         []dto.MapPoint{},
		Countries:      []dto.MapCountryItem{},
		Attribution:    []dto.AddressAttribution{},
	}
	if len(addresses) == 0 {
		return response, nil
	}

	addressIDs := make([]uint, len(addresses))
	for i, address := range addresses {
		addressIDs[i] = address.ID
	}

	// Residents of each address, joined through the pivot. The contacts table is
	// joined manually, so its soft-delete predicate has to be spelled out.
	type residentRow struct {
		AddressID  uint    `gorm:"column:address_id"`
		ContactID  string  `gorm:"column:contact_id"`
		VaultID    string  `gorm:"column:vault_id"`
		FirstName  *string `gorm:"column:first_name"`
		LastName   *string `gorm:"column:last_name"`
		MiddleName *string `gorm:"column:middle_name"`
		Nickname   *string `gorm:"column:nickname"`
		MaidenName *string `gorm:"column:maiden_name"`
		Prefix     *string `gorm:"column:prefix"`
		Suffix     *string `gorm:"column:suffix"`
	}
	var residents []residentRow
	if err := s.db.Table("contact_address").
		Select("contact_address.address_id, contact_address.contact_id, contacts.vault_id, contacts.first_name, contacts.last_name, contacts.middle_name, contacts.nickname, contacts.maiden_name, contacts.prefix, contacts.suffix").
		Joins("JOIN contacts ON contacts.id = contact_address.contact_id").
		Where("contact_address.address_id IN ? AND contacts.deleted_at IS NULL AND contacts.listed = ?", addressIDs, true).
		Scan(&residents).Error; err != nil {
		return nil, err
	}

	byAddress := make(map[uint][]dto.MapContactItem, len(addresses))
	for i := range residents {
		resident := residents[i]
		contact := models.Contact{
			VaultID: resident.VaultID, FirstName: resident.FirstName, LastName: resident.LastName,
			MiddleName: resident.MiddleName, Nickname: resident.Nickname, MaidenName: resident.MaidenName,
			Prefix: resident.Prefix, Suffix: resident.Suffix,
		}
		name, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		byAddress[resident.AddressID] = append(byAddress[resident.AddressID], dto.MapContactItem{
			ContactID:   resident.ContactID,
			ContactName: name,
		})
	}

	type countryAgg struct {
		addresses int
		geocoded  int
		contacts  map[string]struct{}
	}
	countries := make(map[string]*countryAgg)

	for _, address := range addresses {
		country := ptrToStr(address.Country)
		agg, ok := countries[country]
		if !ok {
			agg = &countryAgg{contacts: make(map[string]struct{})}
			countries[country] = agg
		}
		agg.addresses++
		for _, contact := range byAddress[address.ID] {
			agg.contacts[contact.ContactID] = struct{}{}
		}

		if address.Latitude == nil || address.Longitude == nil {
			continue
		}
		agg.geocoded++
		response.GeocodedCount++

		residents := byAddress[address.ID]
		sort.Slice(residents, func(i, j int) bool { return residents[i].ContactName < residents[j].ContactName })
		if residents == nil {
			residents = []dto.MapContactItem{}
		}
		response.Points = append(response.Points, dto.MapPoint{
			AddressID: address.ID,
			Latitude:  *address.Latitude,
			Longitude: *address.Longitude,
			City:      ptrToStr(address.City),
			Province:  ptrToStr(address.Province),
			Country:   country,
			Contacts:  residents,
		})
	}

	for country, agg := range countries {
		contactIDs := make([]string, 0, len(agg.contacts))
		for contactID := range agg.contacts {
			contactIDs = append(contactIDs, contactID)
		}
		sort.Strings(contactIDs)
		response.Countries = append(response.Countries, dto.MapCountryItem{
			Country:      country,
			AddressCount: agg.addresses,
			ContactCount: len(agg.contacts),
			ContactIDs:   contactIDs,
			Geocoded:     agg.geocoded,
		})
	}
	sort.Slice(response.Countries, func(i, j int) bool {
		if response.Countries[i].AddressCount != response.Countries[j].AddressCount {
			return response.Countries[i].AddressCount > response.Countries[j].AddressCount
		}
		return response.Countries[i].Country < response.Countries[j].Country
	})
	sort.Slice(response.Points, func(i, j int) bool { return response.Points[i].AddressID < response.Points[j].AddressID })

	return response, nil
}
