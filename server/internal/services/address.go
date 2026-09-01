package services

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrAddressNotFound = errors.New("address not found")

type AddressService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder

	geocodingMu        sync.RWMutex
	geocoder           Geocoder
	geocodingPrecision string
}

func NewAddressService(db *gorm.DB) *AddressService {
	return &AddressService{db: db}
}

func (s *AddressService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *AddressService) SetGeocoder(g Geocoder) {
	s.geocodingMu.Lock()
	defer s.geocodingMu.Unlock()
	s.geocoder = g
}

// SetGeocodingPrecision decides how much of an address leaves the server when
// it is geocoded. Anything other than GeocodingPrecisionLocality behaves as
// exact, which is the historical behaviour.
func (s *AddressService) SetGeocodingPrecision(precision string) {
	s.geocodingMu.Lock()
	defer s.geocodingMu.Unlock()
	s.geocodingPrecision = normalizeGeocodingPrecision(precision)
}

// ConfigureGeocoding atomically replaces every runtime setting used for an
// outbound lookup. Admin updates can change the provider, key and privacy
// precision together, so no request may observe a half-reloaded combination.
func (s *AddressService) ConfigureGeocoding(g Geocoder, precision string) {
	s.geocodingMu.Lock()
	defer s.geocodingMu.Unlock()
	s.geocoder = g
	s.geocodingPrecision = normalizeGeocodingPrecision(precision)
}

type geocodingRuntime struct {
	geocoder  Geocoder
	precision string
}

func (s *AddressService) geocodingSnapshot() geocodingRuntime {
	s.geocodingMu.RLock()
	defer s.geocodingMu.RUnlock()
	return geocodingRuntime{
		geocoder:  s.geocoder,
		precision: normalizeGeocodingPrecision(s.geocodingPrecision),
	}
}

func normalizeGeocodingPrecision(precision string) string {
	if precision == GeocodingPrecisionLocality {
		return GeocodingPrecisionLocality
	}
	return GeocodingPrecisionExact
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

	// Coordinates the caller already knows are kept as given — spending a
	// provider request to second-guess them would make POST and PUT disagree
	// about whose coordinates win.
	if address.Latitude == nil || address.Longitude == nil {
		s.tryGeocode(&address, s.geocodingSnapshot())
	}

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
	runtime := s.geocodingSnapshot()

	// Remember where the address was before it is overwritten, so an edit that
	// does not move it keeps its coordinates. The address form does not send
	// latitude or longitude, so assigning them straight from the request would
	// erase the geocode on every save and nothing would ever recompute it.
	previousQuery := geocodeQuery(&address, runtime.precision)
	previousLatitude, previousLongitude := address.Latitude, address.Longitude

	address.Line1 = strPtrOrNil(req.Line1)
	address.Line2 = strPtrOrNil(req.Line2)
	address.City = strPtrOrNil(req.City)
	address.Province = strPtrOrNil(req.Province)
	address.PostalCode = strPtrOrNil(req.PostalCode)
	address.Country = strPtrOrNil(req.Country)
	address.AddressTypeID = req.AddressTypeID

	// A cosmetic edit is not a move: trailing whitespace or a change of case
	// would be asked of the geocoder as the same question, so it must not cost
	// the stored coordinates (nor a provider request to recompute them).
	movedElsewhere := !sameGeocodeQuery(geocodeQuery(&address, runtime.precision), previousQuery)

	// Coordinates in the request only count as caller-supplied when they say
	// something the server did not already say. PUT is full-replace, so a
	// read-modify-write client echoes the whole object back — including the
	// coordinates it was handed. If the address text moved but the coordinates
	// are the old pair verbatim, that is a stale echo, not an instruction to
	// pin a Vienna address at London forever.
	echoedCoordinates := req.Latitude != nil && req.Longitude != nil &&
		previousLatitude != nil && previousLongitude != nil &&
		*req.Latitude == *previousLatitude && *req.Longitude == *previousLongitude
	coordinatesGiven := req.Latitude != nil && req.Longitude != nil &&
		!(movedElsewhere && echoedCoordinates)
	switch {
	case coordinatesGiven:
		address.Latitude = req.Latitude
		address.Longitude = req.Longitude
	case movedElsewhere:
		// The address now describes somewhere else, so the old coordinates are
		// simply wrong. Drop them before re-geocoding: keeping them until a
		// replacement arrives would leave the address pinned to its previous
		// location whenever the provider errors or finds nothing, which is a
		// worse answer than having no pin at all.
		address.Latitude, address.Longitude = nil, nil
	default:
		address.Latitude = previousLatitude
		address.Longitude = previousLongitude
	}

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

	// Re-geocode when the address actually moved — and also when it has no
	// coordinates at all, because "arrive back at the coordinates already
	// stored" is no reason to skip when nothing is stored: an address created
	// while the provider was down would otherwise never get a pin without the
	// user mangling its text and changing it back.
	if !coordinatesGiven && (movedElsewhere || address.Latitude == nil || address.Longitude == nil) {
		s.tryGeocode(&address, runtime)
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

// Geocoding precision settings.
const (
	// GeocodingPrecisionExact sends the whole address, giving a pin on the
	// doorstep. This is the default, and what Bonds has always done.
	GeocodingPrecisionExact = "exact"
	// GeocodingPrecisionLocality sends only the district: the outward half of a
	// postcode where there is one, otherwise the town, plus region and country.
	// The map still shows where someone lives to within a neighbourhood, but no
	// contact's street address is ever sent to a third-party geocoder.
	GeocodingPrecisionLocality = "locality"
)

// geocodeQuery renders the address as the single line a geocoder is asked
// about. Line 2 is left out deliberately: flat and building numbers are noise
// to a geocoder and routinely cost it the match.
func geocodeQuery(address *models.Address, precision string) string {
	if precision == GeocodingPrecisionLocality {
		return localityQuery(address)
	}
	parts := []string{}
	for _, p := range []*string{address.Line1, address.City, address.Province, address.PostalCode, address.Country} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	return strings.Join(parts, ", ")
}

// localityQuery describes the address only as far as its district.
//
// The street line is dropped entirely. The postcode is cut back to its outward
// part — "SW1A 2AA" becomes "SW1A", which covers a neighbourhood rather than a
// building — and where there is no postcode the town stands in for it.
func localityQuery(address *models.Address) string {
	parts := []string{}
	if outward := outwardCode(ptrToStr(address.PostalCode)); outward != "" {
		parts = append(parts, outward)
	}
	for _, p := range []*string{address.City, address.Province, address.Country} {
		if p != nil && *p != "" {
			parts = append(parts, *p)
		}
	}
	return strings.Join(parts, ", ")
}

// outwardCode reduces a postcode to the part that identifies a district rather
// than a delivery point. UK postcodes split on a space ("SW1A 2AA" -> "SW1A");
// ZIP+4 splits on a hyphen ("94103-1234" -> "94103"). Anything with neither
// separator is already district-sized and is returned unchanged.
func outwardCode(postalCode string) string {
	trimmed := strings.TrimSpace(postalCode)
	if trimmed == "" {
		return ""
	}
	if index := strings.LastIndex(trimmed, " "); index > 0 {
		return strings.TrimSpace(trimmed[:index])
	}
	if index := strings.Index(trimmed, "-"); index > 0 {
		return strings.TrimSpace(trimmed[:index])
	}
	return trimmed
}

// sameGeocodeQuery reports whether two geocoding queries would ask the
// provider the same question: case and whitespace do not change the answer,
// so they do not count as a move. Whitespace is removed entirely rather than
// collapsed, because a trimmed field otherwise leaves a stranded separator
// ("London ," versus "London,") that a word-level comparison still sees.
func sameGeocodeQuery(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), ""), strings.Join(strings.Fields(b), ""))
}

func (s *AddressService) tryGeocode(address *models.Address, runtime geocodingRuntime) {
	if runtime.geocoder == nil {
		return
	}
	query := geocodeQuery(address, runtime.precision)
	if query == "" {
		return
	}
	result, err := runtime.geocoder.Geocode(query)
	if err != nil {
		// Worth a line in the log: a misconfigured provider or a blocked IP is
		// otherwise completely invisible, and the only symptom is addresses
		// quietly never getting coordinates. The query itself stays out of the
		// log, though — in exact mode it is a contact's complete home address,
		// and the address ID identifies the row just as well.
		log.Printf("geocoding address %d via %T failed: %v", address.ID, runtime.geocoder, redactGeocodeError(err))
		return
	}
	if result == nil {
		return
	}
	// The provider can take seconds to answer, and this runs after the edit's
	// transaction committed — the row may already have been edited again. Make
	// the address version we geocoded part of the UPDATE itself so a newer edit
	// cannot land between a separate check and this write.
	stored := s.db.Model(&models.Address{}).
		Where("id = ? AND updated_at = ?", address.ID, address.UpdatedAt).
		Select("latitude", "longitude").
		Updates(map[string]any{
			"latitude":  result.Latitude,
			"longitude": result.Longitude,
		})
	if stored.Error != nil {
		log.Printf("storing geocode for address %d failed: %v", address.ID, stored.Error)
		return
	}
	if stored.RowsAffected == 0 {
		return
	}
	address.Latitude = &result.Latitude
	address.Longitude = &result.Longitude
}

// redactGeocodeError strips the request URL from a geocoding failure before
// it reaches the log. A transport error carries the full URL it was for, and
// that URL embeds the geocoding query — the same address the log line above
// deliberately leaves out.
func redactGeocodeError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s request failed: %w", urlErr.Op, urlErr.Err)
	}
	return err
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

// autocompleteAvailable reports whether address lookup may be offered at all.
//
// Two things can withdraw it, and both are about what may leave the server
// rather than about whether it would work:
//
//   - The provider's terms. The public Nominatim instance forbids autocomplete,
//     so it is never driven as a type-ahead however slowly requests are paced.
//   - Locality precision. That mode exists so a contact's street address is
//     never sent to a geocoder, and autocomplete would send exactly that: the
//     partial line the reader is typing. There is no way to answer a street
//     query at district precision, so the feature is withdrawn rather than
//     quietly weakened.
func autocompleteAvailable(runtime geocodingRuntime) bool {
	if runtime.geocoder == nil {
		return false
	}
	if runtime.precision == GeocodingPrecisionLocality {
		return false
	}
	return runtime.geocoder.SupportsAutocomplete()
}

// Attribution lists the data credits that must be shown wherever this
// provider's data is displayed. Forward geocoding can still create stored
// coordinates when autocomplete is unavailable (public Nominatim and locality
// precision), so attribution must not be tied to the suggestion control.
func (s *AddressService) Attribution() []dto.AddressAttribution {
	runtime := s.geocodingSnapshot()
	if runtime.geocoder == nil {
		return []dto.AddressAttribution{}
	}
	credits := runtime.geocoder.Attribution()
	attribution := make([]dto.AddressAttribution, 0, len(credits))
	for _, credit := range credits {
		attribution = append(attribution, dto.AddressAttribution{Label: credit.Label, URL: credit.URL})
	}
	return attribution
}

// Suggest returns candidate addresses for a partial query, for the address
// form's autocomplete.
//
// The second return value reports whether lookup is available at all: an
// instance with no geocoding provider configured is not an error, it simply has
// no suggestions to offer, and the caller should hide the control rather than
// show an empty one.
func (s *AddressService) Suggest(query string, limit int) ([]dto.AddressSuggestionItem, bool, error) {
	runtime := s.geocodingSnapshot()
	if !autocompleteAvailable(runtime) {
		return []dto.AddressSuggestionItem{}, false, nil
	}
	if strings.TrimSpace(query) == "" {
		return []dto.AddressSuggestionItem{}, true, nil
	}

	results, err := runtime.geocoder.Suggest(query, limit)
	if err != nil {
		return nil, true, err
	}

	suggestions := make([]dto.AddressSuggestionItem, 0, len(results))
	for _, result := range results {
		latitude, longitude := result.Latitude, result.Longitude
		suggestions = append(suggestions, dto.AddressSuggestionItem{
			Label:      result.Label,
			Line1:      result.Line1,
			City:       result.City,
			Province:   result.Province,
			PostalCode: result.PostalCode,
			Country:    result.Country,
			Latitude:   &latitude,
			Longitude:  &longitude,
		})
	}
	return suggestions, true, nil
}
