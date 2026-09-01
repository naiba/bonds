package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const suggestPayload = `[
  {
    "lat": "51.5034",
    "lon": "-0.1276",
    "display_name": "10, Downing Street, London, SW1A 2AA, United Kingdom",
    "address": {
      "house_number": "10",
      "road": "Downing Street",
      "city": "London",
      "state": "England",
      "postcode": "SW1A 2AA",
      "country": "United Kingdom"
    }
  },
  {
    "lat": "48.2082",
    "lon": "16.3738",
    "display_name": "Vienna, Austria",
    "address": {
      "name": "Vienna",
      "town": "Vienna",
      "county": "Wien",
      "country": "Austria"
    }
  },
  {
    "lat": "51.5205",
    "lon": "-0.1571",
    "display_name": "Baker Street, Marylebone, London, United Kingdom",
    "address": {
      "name": "Baker Street",
      "suburb": "Marylebone",
      "city": "London",
      "state": "England",
      "country": "United Kingdom"
    }
  }
]`

func TestSuggestSplitsAddressIntoFormFields(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(suggestPayload))
	}))
	defer server.Close()

	suggestions, err := suggestFromURL(server.Client(), server.URL, "10 downing", "", 5, nil)
	if err != nil {
		t.Fatalf("suggestFromURL failed: %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}

	// Structured fields are what make autocomplete fill the whole form rather
	// than dumping a formatted string into line 1.
	if suggestions[0].Line1 != "10 Downing Street" {
		t.Fatalf("expected the house number folded into line 1, got %q", suggestions[0].Line1)
	}
	if suggestions[0].City != "London" || suggestions[0].Province != "England" {
		t.Fatalf("unexpected city/province: %q / %q", suggestions[0].City, suggestions[0].Province)
	}
	if suggestions[0].PostalCode != "SW1A 2AA" || suggestions[0].Country != "United Kingdom" {
		t.Fatalf("unexpected postcode/country: %q / %q", suggestions[0].PostalCode, suggestions[0].Country)
	}
	if suggestions[0].Latitude != 51.5034 || suggestions[0].Longitude != -0.1276 {
		t.Fatalf("unexpected coordinates: %v, %v", suggestions[0].Latitude, suggestions[0].Longitude)
	}

	// Which key holds the town and the region varies by country.
	if suggestions[1].City != "Vienna" || suggestions[1].Province != "Wien" {
		t.Fatalf("expected town/county to be used as city/province, got %q / %q", suggestions[1].City, suggestions[1].Province)
	}
	// The city-level result carries its own name in "name" — that is the city,
	// not a street, and must not leak into line 1.
	if suggestions[1].Line1 != "" {
		t.Fatalf("expected no street line for a city-level result, got %q", suggestions[1].Line1)
	}

	// A street-layer match puts the street in "name" with no "road"; the form
	// still deserves a line 1.
	if suggestions[2].Line1 != "Baker Street" {
		t.Fatalf("expected the street name as line 1, got %q", suggestions[2].Line1)
	}
	if suggestions[2].City != "London" {
		t.Fatalf("unexpected city for the street match: %q", suggestions[2].City)
	}

	// The autocomplete product always includes the address breakdown and always
	// answers JSON, so the search product's format/addressdetails parameters
	// must not be sent.
	for _, banned := range []string{"addressdetails", "format"} {
		if contains(gotQuery, banned) {
			t.Fatalf("search-product parameter %q sent to the autocomplete endpoint: %q", banned, gotQuery)
		}
	}
}

func TestSuggestSkipsEmptyQueryWithoutCallingProvider(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	suggestions, err := suggestFromURL(server.Client(), server.URL, "   ", "", 5, nil)
	if err != nil {
		t.Fatalf("suggestFromURL failed: %v", err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %d", len(suggestions))
	}
	if called {
		t.Fatal("a blank query must not reach the provider")
	}
}

func TestRateLimiterSpacesRequests(t *testing.T) {
	limiter := &rateLimiter{interval: 50 * time.Millisecond}

	// The first slot is free; the next two are booked behind it.
	if wait, ok := limiter.reserve(time.Second); !ok || wait != 0 {
		t.Fatalf("expected the first slot to be immediate, got wait=%v ok=%v", wait, ok)
	}
	second, ok := limiter.reserve(time.Second)
	if !ok || second <= 0 {
		t.Fatalf("expected the second caller to wait, got wait=%v ok=%v", second, ok)
	}
	third, ok := limiter.reserve(time.Second)
	if !ok || third <= second {
		t.Fatalf("expected each caller to queue behind the last, got %v then %v", second, third)
	}
}

func TestRateLimiterRefusesRatherThanQueueForever(t *testing.T) {
	limiter := &rateLimiter{interval: time.Second}
	if _, ok := limiter.reserve(time.Second); !ok {
		t.Fatal("expected the first slot to be granted")
	}
	// Waiting a full second for an autocomplete answer is worse than none.
	if _, ok := limiter.reserve(10 * time.Millisecond); ok {
		t.Fatal("expected the limiter to refuse a slot beyond the deadline")
	}
	if err := limiter.wait(10 * time.Millisecond); err != ErrGeocoderBusy {
		t.Fatalf("expected ErrGeocoderBusy, got %v", err)
	}
	// A refusal must not have consumed the slot it declined.
	if _, ok := limiter.reserve(2 * time.Second); !ok {
		t.Fatal("expected a later caller with a longer deadline to still be served")
	}
}

func TestNilRateLimiterIsUnlimited(t *testing.T) {
	var limiter *rateLimiter
	if err := limiter.wait(time.Millisecond); err != nil {
		t.Fatalf("a nil limiter must not block: %v", err)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// trippingTransport fails the test if any request is ever made through it.
type trippingTransport struct{ tripped *bool }

func (t *trippingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*t.tripped = true
	return nil, http.ErrHandlerTimeout
}

func TestNominatimSuggestRefusesWithoutTouchingTheNetwork(t *testing.T) {
	tripped := false
	g := NewNominatimGeocoder()
	g.client = &http.Client{Transport: &trippingTransport{tripped: &tripped}}

	_, err := g.Suggest("10 downing", 5)
	if err != ErrAutocompleteUnsupported {
		t.Fatalf("expected ErrAutocompleteUnsupported, got %v", err)
	}
	if tripped {
		t.Fatal("the public Nominatim instance must never receive an autocomplete request")
	}
}

func TestLocationIQSuggestUsesTheAutocompleteProduct(t *testing.T) {
	// The documented type-ahead product, not the forward-geocoding /search API.
	if locationIQAutocompleteURL != "https://api.locationiq.com/v1/autocomplete" {
		t.Fatalf("unexpected autocomplete endpoint: %q", locationIQAutocompleteURL)
	}

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(suggestPayload))
	}))
	defer server.Close()

	g := NewLocationIQGeocoder("secret-key")
	g.client = server.Client()
	g.limiter = nil
	g.autocompleteURL = server.URL

	suggestions, err := g.Suggest("10 downing", 5)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}
	for _, expected := range []string{"key=secret-key", "q=10+downing", "limit=5"} {
		if !contains(gotQuery, expected) {
			t.Fatalf("expected %q in the autocomplete request, got %q", expected, gotQuery)
		}
	}
}

func TestSuggestQueryCutAtTheDocumentedLimit(t *testing.T) {
	// LocationIQ documents a 200-character ceiling on q. The cut must land on
	// a rune boundary: "é" is two bytes, and slicing through it would send an
	// invalid byte sequence.
	long := strings.Repeat("é", 150) // 300 bytes
	cut := truncateQuery(long, suggestQueryMaxLength)
	if len(cut) > suggestQueryMaxLength {
		t.Fatalf("query still %d bytes after the cut", len(cut))
	}
	if !utf8.ValidString(cut) {
		t.Fatal("the cut split a rune")
	}
	if short := truncateQuery("10 downing", suggestQueryMaxLength); short != "10 downing" {
		t.Fatalf("short query must pass through unchanged, got %q", short)
	}
}

func TestProviderAttributionCredits(t *testing.T) {
	nominatim := NewNominatimGeocoder().Attribution()
	if len(nominatim) != 1 || nominatim[0].URL != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("Nominatim must credit OpenStreetMap with its copyright page, got %v", nominatim)
	}

	locationIQ := NewLocationIQGeocoder("k").Attribution()
	if len(locationIQ) != 2 {
		t.Fatalf("LocationIQ must credit both OpenStreetMap and itself, got %v", locationIQ)
	}
	if locationIQ[0].URL != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("LocationIQ results are OSM-derived and must say so, got %v", locationIQ[0])
	}
	if locationIQ[1].Label != "Search by LocationIQ.com" || locationIQ[1].URL != "https://locationiq.com/attribution" {
		t.Fatalf("unexpected LocationIQ credit: %v", locationIQ[1])
	}
}
