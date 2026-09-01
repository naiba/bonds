package services

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// stubGeocoder records what it was asked and answers with fixed coordinates.
type stubGeocoder struct {
	queries      []string
	latitude     float64
	longitude    float64
	fail         bool
	noMatch      bool
	autocomplete bool
	// onGeocode runs while the provider is "on the wire", so tests can stage
	// what happens between an edit's commit and the geocode answer landing.
	onGeocode func()
}

func (g *stubGeocoder) Geocode(address string) (*GeocodingResult, error) {
	g.queries = append(g.queries, address)
	if g.onGeocode != nil {
		g.onGeocode()
	}
	if g.fail {
		return nil, errors.New("provider unavailable")
	}
	if g.noMatch {
		return nil, nil
	}
	return &GeocodingResult{Latitude: g.latitude, Longitude: g.longitude}, nil
}

func (g *stubGeocoder) SupportsAutocomplete() bool { return g.autocomplete }

func (g *stubGeocoder) Attribution() []ProviderAttribution {
	return []ProviderAttribution{{Label: "Test data", URL: "https://example.com/licence"}}
}

func (g *stubGeocoder) Suggest(query string, limit int) ([]GeocodingSuggestion, error) {
	return nil, nil
}

func storedCoordinates(t *testing.T, svc *AddressService, id uint) (*float64, *float64) {
	t.Helper()
	var address models.Address
	if err := svc.db.First(&address, id).Error; err != nil {
		t.Fatalf("reloading address: %v", err)
	}
	return address.Latitude, address.Longitude
}

func TestUpdateAddressKeepsCoordinatesWhenNotMoved(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(geocoder.queries) != 1 {
		t.Fatalf("expected the new address to be geocoded once, got %d calls", len(geocoder.queries))
	}

	// The address form does not send coordinates. Before this was fixed, that
	// alone wiped them on every save and nothing ever recomputed them.
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || longitude == nil {
		t.Fatal("editing an address must not erase its coordinates")
	}
	if *latitude != 51.5072 || *longitude != -0.1276 {
		t.Fatalf("unexpected coordinates: %v, %v", *latitude, *longitude)
	}
	// Unchanged text must not spend another provider request.
	if len(geocoder.queries) != 1 {
		t.Fatalf("expected no re-geocode for an unchanged address, got %d calls", len(geocoder.queries))
	}
}

func TestUpdateAddressRegeocodesWhenMoved(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	geocoder.latitude, geocoder.longitude = 48.2082, 16.3738
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Stephansplatz 1", City: "Vienna", Country: "Austria",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 48.2082 || longitude == nil || *longitude != 16.3738 {
		t.Fatalf("expected the moved address to be re-geocoded, got %v, %v", latitude, longitude)
	}
	if len(geocoder.queries) != 2 {
		t.Fatalf("expected exactly one re-geocode, got %d calls", len(geocoder.queries))
	}
	if geocoder.queries[1] != "Stephansplatz 1, Vienna, Austria" {
		t.Fatalf("unexpected geocoding query: %q", geocoder.queries[1])
	}
}

func TestUpdateAddressPrefersCallerSuppliedCoordinates(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// A client that picked a suggestion already knows the coordinates, so the
	// provider should not be asked again.
	latitude, longitude := 48.2082, 16.3738
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Stephansplatz 1", City: "Vienna", Country: "Austria",
		Latitude: &latitude, Longitude: &longitude,
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	storedLat, storedLon := storedCoordinates(t, svc, created.ID)
	if storedLat == nil || *storedLat != 48.2082 || storedLon == nil || *storedLon != 16.3738 {
		t.Fatalf("expected the supplied coordinates to be kept, got %v, %v", storedLat, storedLon)
	}
	if len(geocoder.queries) != 1 {
		t.Fatalf("expected no extra geocode, got %d calls", len(geocoder.queries))
	}
}

func TestSuggestReportsLookupDisabledWithoutGeocoder(t *testing.T) {
	svc, _, _ := setupAddressTest(t)

	suggestions, enabled, err := svc.Suggest("downing", 5)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if enabled {
		t.Fatal("lookup must report itself disabled when no provider is configured")
	}
	if suggestions == nil {
		t.Fatal("expected an empty slice rather than nil")
	}
}

func TestOutwardCodeKeepsOnlyTheDistrict(t *testing.T) {
	cases := map[string]string{
		"SW1A 2AA":   "SW1A",  // UK: the outward half covers a neighbourhood
		"sw1a 2aa":   "sw1a",  // case is the caller's business, not ours
		"EC1A 1BB ":  "EC1A",  // trailing space
		"94103-1234": "94103", // ZIP+4 reduces to the ZIP
		"94103":      "94103", // already district-sized
		"75008":      "75008", // no separator, left alone
		"":           "",      // nothing to reduce
		"   ":        "",      // whitespace is nothing
	}
	for input, want := range cases {
		if got := outwardCode(input); got != want {
			t.Errorf("outwardCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLocalityPrecisionNeverSendsTheStreet(t *testing.T) {
	address := &models.Address{
		Line1:      strPtr("221B Baker Street"),
		Line2:      strPtr("Flat 3"),
		City:       strPtr("London"),
		Province:   strPtr("Greater London"),
		PostalCode: strPtr("NW1 6XE"),
		Country:    strPtr("United Kingdom"),
	}

	exact := geocodeQuery(address, GeocodingPrecisionExact)
	if exact != "221B Baker Street, London, Greater London, NW1 6XE, United Kingdom" {
		t.Fatalf("unexpected exact query: %q", exact)
	}

	locality := geocodeQuery(address, GeocodingPrecisionLocality)
	if strings.Contains(locality, "Baker Street") || strings.Contains(locality, "221B") {
		t.Fatalf("the street must never leave the server at locality precision: %q", locality)
	}
	if strings.Contains(locality, "6XE") {
		t.Fatalf("the inward postcode identifies a building and must be dropped: %q", locality)
	}
	if locality != "NW1, London, Greater London, United Kingdom" {
		t.Fatalf("unexpected locality query: %q", locality)
	}
}

func TestLocalityPrecisionFallsBackToTheTown(t *testing.T) {
	address := &models.Address{
		Line1:   strPtr("12 Rue de Rivoli"),
		City:    strPtr("Paris"),
		Country: strPtr("France"),
	}
	// No postcode: the town has to carry the answer on its own.
	if got := geocodeQuery(address, GeocodingPrecisionLocality); got != "Paris, France" {
		t.Fatalf("unexpected locality query: %q", got)
	}
}

func TestLocalityPrecisionIsUsedWhenGeocoding(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5237, longitude: -0.1585}
	svc.SetGeocoder(geocoder)
	svc.SetGeocodingPrecision(GeocodingPrecisionLocality)

	if _, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "221B Baker Street", City: "London", PostalCode: "NW1 6XE", Country: "United Kingdom",
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(geocoder.queries) != 1 {
		t.Fatalf("expected one geocode, got %d", len(geocoder.queries))
	}
	if strings.Contains(geocoder.queries[0], "Baker Street") {
		t.Fatalf("the street reached the provider: %q", geocoder.queries[0])
	}
	if geocoder.queries[0] != "NW1, London, United Kingdom" {
		t.Fatalf("unexpected query: %q", geocoder.queries[0])
	}
}

func TestLocalityPrecisionSkipsRegeocodeWhenOnlyTheStreetChanges(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5237, longitude: -0.1585}
	svc.SetGeocoder(geocoder)
	svc.SetGeocodingPrecision(GeocodingPrecisionLocality)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "221B Baker Street", City: "London", PostalCode: "NW1 6XE", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Moving down the same street does not change the district, so there is
	// nothing new to ask the provider.
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "224 Baker Street", City: "London", PostalCode: "NW1 6XE", Country: "United Kingdom",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(geocoder.queries) != 1 {
		t.Fatalf("expected no second geocode, got %d", len(geocoder.queries))
	}
}

func TestUpdateAddressDropsCoordinatesWhenGeocodingFails(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if lat, _ := storedCoordinates(t, svc, created.ID); lat == nil {
		t.Fatal("expected the new address to be geocoded")
	}

	// The address moves, and the provider cannot answer. Keeping the old
	// coordinates would leave this address pinned to London on the map.
	geocoder.fail = true
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Stephansplatz 1", City: "Vienna", Country: "Austria",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude != nil || longitude != nil {
		t.Fatalf("a moved address must not keep its old coordinates when geocoding fails, got %v, %v", latitude, longitude)
	}
}

func TestUpdateAddressDropsCoordinatesWhenProviderFindsNothing(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	geocoder.noMatch = true
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Nowhere At All", City: "Nowhereville", Country: "Atlantis",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if latitude, _ := storedCoordinates(t, svc, created.ID); latitude != nil {
		t.Fatalf("expected no coordinates for an unresolvable address, got %v", *latitude)
	}
}
func TestSuggestRefusedWhenProviderForbidsAutocomplete(t *testing.T) {
	svc, _, _ := setupAddressTest(t)
	// The public Nominatim instance forbids autocomplete outright.
	svc.SetGeocoder(&stubGeocoder{autocomplete: false})

	suggestions, enabled, err := svc.Suggest("10 downing", 5)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if enabled {
		t.Fatal("autocomplete must report itself unavailable when the provider forbids it")
	}
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %d", len(suggestions))
	}
}

func TestSuggestRefusedAtLocalityPrecision(t *testing.T) {
	svc, _, _ := setupAddressTest(t)
	geocoder := &stubGeocoder{autocomplete: true}
	svc.SetGeocoder(geocoder)
	svc.SetGeocodingPrecision(GeocodingPrecisionLocality)

	// Locality mode promises no street address leaves the server. Autocomplete
	// would send the partial street line the reader is typing, so it is
	// withdrawn rather than quietly weakened.
	_, enabled, err := svc.Suggest("221b baker street", 5)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if enabled {
		t.Fatal("autocomplete must be unavailable at locality precision")
	}
	if len(geocoder.queries) != 0 {
		t.Fatalf("nothing may reach the provider at locality precision, got %v", geocoder.queries)
	}
}

func TestSuggestWorksWhenProviderPermitsAndPrecisionIsExact(t *testing.T) {
	svc, _, _ := setupAddressTest(t)
	svc.SetGeocoder(&stubGeocoder{autocomplete: true})
	svc.SetGeocodingPrecision(GeocodingPrecisionExact)

	_, enabled, err := svc.Suggest("10 downing", 5)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if !enabled {
		t.Fatal("expected autocomplete to be available")
	}
}

func TestAttributionFollowsProviderEvenWhenAutocompleteIsUnavailable(t *testing.T) {
	svc, _, _ := setupAddressTest(t)

	// No provider: nothing is shown, so nothing needs crediting.
	if credits := svc.Attribution(); len(credits) != 0 {
		t.Fatalf("expected no attribution without a geocoder, got %v", credits)
	}

	svc.SetGeocoder(&stubGeocoder{autocomplete: true})
	credits := svc.Attribution()
	if len(credits) != 1 || credits[0].Label != "Test data" || credits[0].URL != "https://example.com/licence" {
		t.Fatalf("expected the provider's credit passed through, got %v", credits)
	}

	// Locality precision withdraws autocomplete, but forward geocoding still
	// creates coordinates that may be displayed elsewhere and still need credit.
	svc.SetGeocodingPrecision(GeocodingPrecisionLocality)
	if credits := svc.Attribution(); len(credits) != 1 || credits[0].Label != "Test data" {
		t.Fatalf("expected provider attribution at locality precision, got %v", credits)
	}
}

// urlErrorGeocoder fails the way a real provider does over HTTP: with a
// transport error that carries the full request URL, query string and all.
type urlErrorGeocoder struct{}

func (g *urlErrorGeocoder) Geocode(address string) (*GeocodingResult, error) {
	return nil, &url.Error{
		Op:  "Get",
		URL: "https://us1.locationiq.com/v1/search?key=super-secret-api-key&q=" + url.QueryEscape(address),
		Err: errors.New("connection refused"),
	}
}

func (g *urlErrorGeocoder) Suggest(query string, limit int) ([]GeocodingSuggestion, error) {
	return nil, ErrAutocompleteUnsupported
}

func (g *urlErrorGeocoder) SupportsAutocomplete() bool { return false }

func (g *urlErrorGeocoder) Attribution() []ProviderAttribution { return nil }

func TestGeocodeFailureLogLeavesTheAddressOut(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	svc.SetGeocoder(&urlErrorGeocoder{})

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	logged := buf.String()
	if logged == "" {
		t.Fatal("expected the geocoding failure to be logged")
	}
	// Neither the query nor the request URL may appear: in exact mode both are
	// a contact's complete home address.
	// The API key rides in the same URL, so it must go down with it.
	for _, fragment := range []string{"Downing", "London", "United Kingdom", "q=", "super-secret-api-key"} {
		if strings.Contains(logged, fragment) {
			t.Fatalf("log line leaks the address (%q): %s", fragment, logged)
		}
	}
	if !strings.Contains(logged, fmt.Sprintf("address %d", created.ID)) {
		t.Fatalf("log line should identify the address by ID: %s", logged)
	}
	if !strings.Contains(logged, "connection refused") {
		t.Fatalf("log line should keep the underlying cause: %s", logged)
	}
}

// PUT is full-replace, so a read-modify-write API client echoes the whole
// object back — coordinates included. Echoed coordinates on a moved address
// are a stale copy of what the server said, not an instruction to keep the
// old pin, and treating them as authoritative would pin the moved address to
// its old location permanently (every subsequent echo re-supplies them).
func TestUpdateAddressTreatsEchoedCoordinatesAsStale(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// The provider now answers for the new city.
	geocoder.latitude, geocoder.longitude = 48.2082, 16.3738
	london := 51.5072
	londonLongitude := -0.1276
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "Vienna", Country: "Austria",
		Latitude: &london, Longitude: &londonLongitude,
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 48.2082 || longitude == nil || *longitude != 16.3738 {
		t.Fatalf("a moved address with echoed coordinates must be re-geocoded, got %v, %v", latitude, longitude)
	}

	// Coordinates that DIFFER from what the server handed out are a deliberate
	// override and still win.
	custom, customLongitude := 40.0, -70.0
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "Berlin", Country: "Germany",
		Latitude: &custom, Longitude: &customLongitude,
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	latitude, longitude = storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 40.0 {
		t.Fatalf("caller-chosen coordinates must win, got %v", latitude)
	}
}

// Trailing whitespace or a change of case would be asked of the geocoder as
// the same question — it is not a move, and must not cost the pin (nor spend
// a provider request), even when the provider is down.
func TestUpdateAddressKeepsCoordinatesOnCosmeticEdit(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London ", Country: "UNITED KINGDOM",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// The provider goes down; the cosmetic edit must not need it.
	geocoder.fail = true
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || longitude == nil {
		t.Fatal("a cosmetic edit erased the coordinates")
	}
	if len(geocoder.queries) != 1 {
		t.Fatalf("a cosmetic edit must not spend a provider request, got %d calls", len(geocoder.queries))
	}
}

// An address created while the provider was down has no pin, and "the stored
// coordinates are already right" is no reason to skip re-geocoding when
// nothing is stored: a plain save must be able to heal it.
func TestUpdateAddressRetriesGeocodeWhenPinMissing(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276, fail: true}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if latitude, _ := storedCoordinates(t, svc, created.ID); latitude != nil {
		t.Fatal("precondition: the failed create should have stored no coordinates")
	}

	geocoder.fail = false
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 51.5072 || longitude == nil {
		t.Fatalf("an unchanged save should have healed the missing pin, got %v, %v", latitude, longitude)
	}
}

// The geocode answer arrives after the edit's transaction committed, and the
// row may have been edited again while the provider was thinking. The late
// answer is for a question the row no longer asks, and must not be stored.
func TestLateGeocodeAnswerDoesNotClobberANewerEdit(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// While Vienna's geocode is "on the wire", the address moves on to Berlin.
	geocoder.latitude, geocoder.longitude = 48.2082, 16.3738
	geocoder.onGeocode = func() {
		geocoder.onGeocode = nil
		if err := svc.db.Model(&models.Address{}).Where("id = ?", created.ID).
			Updates(map[string]any{"city": "Berlin", "country": "Germany", "latitude": 52.52, "longitude": 13.405}).Error; err != nil {
			t.Fatalf("staging the concurrent edit: %v", err)
		}
	}
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "10 Downing Street", City: "Vienna", Country: "Austria",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 52.52 || longitude == nil || *longitude != 13.405 {
		t.Fatalf("Vienna's late answer clobbered Berlin's coordinates: %v, %v", latitude, longitude)
	}
}

// Even a newer edit that lands after the provider answer has been accepted
// but immediately before its coordinates are written must win. This stages
// that edit in GORM's update callback: the optimistic updated_at predicate is
// already built, then the row version changes before the SQL executes.
func TestGeocodeCoordinateWriteIsAtomicWithAddressVersion(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "10 Downing Street", City: "London", Country: "United Kingdom",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	const callbackName = "test:edit_before_geocode_coordinate_write"
	fired := false
	if err := svc.db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if fired || len(tx.Statement.Selects) != 2 ||
			tx.Statement.Selects[0] != "latitude" || tx.Statement.Selects[1] != "longitude" {
			return
		}
		fired = true
		newerVersion := time.Now().UTC().Add(time.Hour)
		if err := tx.Session(&gorm.Session{SkipHooks: true}).Exec(
			"UPDATE addresses SET city = ?, country = ?, latitude = ?, longitude = ?, updated_at = ? WHERE id = ?",
			"Berlin", "Germany", 52.52, 13.405, newerVersion, created.ID,
		).Error; err != nil {
			t.Fatalf("staging edit immediately before coordinate write: %v", err)
		}
	}); err != nil {
		t.Fatalf("registering update callback: %v", err)
	}
	defer func() {
		if err := svc.db.Callback().Update().Remove(callbackName); err != nil {
			t.Fatalf("removing update callback: %v", err)
		}
	}()

	geocoder.latitude, geocoder.longitude = 48.2082, 16.3738
	if _, err := svc.Update(created.ID, contactID, vaultID, dto.UpdateAddressRequest{
		Line1: "Stephansplatz 1", City: "Vienna", Country: "Austria",
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !fired {
		t.Fatal("the concurrent-edit callback did not intercept the coordinate write")
	}

	latitude, longitude := storedCoordinates(t, svc, created.ID)
	if latitude == nil || *latitude != 52.52 || longitude == nil || *longitude != 13.405 {
		t.Fatalf("Vienna's coordinate write clobbered the newer Berlin edit: %v, %v", latitude, longitude)
	}
}

// POST with coordinates the caller already knows must store them as given —
// spending a provider request to second-guess them would make Create and
// Update disagree about whose coordinates win.
func TestCreateAddressKeepsCallerCoordinates(t *testing.T) {
	svc, contactID, vaultID := setupAddressTest(t)
	geocoder := &stubGeocoder{latitude: 51.5072, longitude: -0.1276}
	svc.SetGeocoder(geocoder)

	latitude, longitude := 48.8566, 2.3522
	created, err := svc.Create(contactID, vaultID, dto.CreateAddressRequest{
		Line1: "48 Rue de Rivoli", City: "Paris", Country: "France",
		Latitude: &latitude, Longitude: &longitude,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	stored, storedLongitude := storedCoordinates(t, svc, created.ID)
	if stored == nil || *stored != 48.8566 || storedLongitude == nil || *storedLongitude != 2.3522 {
		t.Fatalf("caller coordinates must be stored as given, got %v, %v", stored, storedLongitude)
	}
	if len(geocoder.queries) != 0 {
		t.Fatalf("no provider request should be spent on known coordinates, got %d", len(geocoder.queries))
	}
}
