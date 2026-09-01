package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoapifyGeocoderSearchAndAutocomplete(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("apiKey") != "geo-key" {
			t.Errorf("missing Geoapify apiKey: %q", r.URL.RawQuery)
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("missing JSON format: %q", r.URL.RawQuery)
		}
		if r.Header.Get("User-Agent") != "Bonds/1.0" {
			t.Errorf("unexpected User-Agent: %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"formatted":"10 Downing Street, London, United Kingdom","address_line1":"10 Downing Street","city":"London","state":"England","postcode":"SW1A 2AA","country":"United Kingdom","lat":51.5034,"lon":-0.1276}]}`))
	}))
	defer server.Close()

	geocoder := NewGeoapifyGeocoder("geo-key")
	geocoder.client = server.Client()
	geocoder.limiter = nil
	geocoder.searchURL = server.URL
	geocoder.autocompleteURL = server.URL
	result, err := geocoder.Geocode("10 Downing Street")
	if err != nil || result == nil || result.Latitude != 51.5034 || result.Longitude != -0.1276 {
		t.Fatalf("Geoapify geocode mismatch: result=%+v err=%v", result, err)
	}
	suggestions, err := geocoder.Suggest("10 Down", 4)
	if err != nil || len(suggestions) != 1 {
		t.Fatalf("Geoapify suggestions: %+v err=%v", suggestions, err)
	}
	if suggestions[0].Line1 != "10 Downing Street" || suggestions[0].City != "London" || suggestions[0].Country != "United Kingdom" {
		t.Fatalf("Geoapify suggestion mapping mismatch: %+v", suggestions[0])
	}
	if requests != 2 {
		t.Fatalf("expected two Geoapify requests, got %d", requests)
	}
	if credits := geocoder.Attribution(); len(credits) != 2 || credits[1].Label != "Powered by Geoapify" {
		t.Fatalf("Geoapify attribution mismatch: %+v", credits)
	}
}

func TestPhotonGeocoderCustomEndpointAndMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" || r.URL.Query().Get("q") == "" || r.URL.Query().Get("limit") == "" {
			t.Errorf("unexpected Photon request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"features":[{"geometry":{"coordinates":[13.405,52.52]},"properties":{"name":"Alexanderplatz","street":"Alexanderplatz","city":"Berlin","state":"Berlin","postcode":"10178","country":"Germany"}}]}`))
	}))
	defer server.Close()

	geocoder, err := NewPhotonGeocoder(server.URL + "/")
	if err != nil {
		t.Fatalf("construct custom Photon: %v", err)
	}
	geocoder.client = server.Client()
	result, err := geocoder.Geocode("Alexanderplatz Berlin")
	if err != nil || result == nil || result.Latitude != 52.52 || result.Longitude != 13.405 {
		t.Fatalf("Photon geocode mismatch: result=%+v err=%v", result, err)
	}
	suggestions, err := geocoder.Suggest("Alexander", 5)
	if err != nil || len(suggestions) != 1 {
		t.Fatalf("Photon suggestions: %+v err=%v", suggestions, err)
	}
	if suggestions[0].City != "Berlin" || suggestions[0].Country != "Germany" {
		t.Fatalf("Photon mapping mismatch: %+v", suggestions[0])
	}
	if credits := geocoder.Attribution(); len(credits) != 2 || credits[1].Label != "Powered by Photon" {
		t.Fatalf("Photon attribution mismatch: %+v", credits)
	}
}

func TestPhotonGeocoderURLValidationAndPublicLimit(t *testing.T) {
	invalid := []string{"", "photon.local", "ftp://photon.local", "https://user:pass@photon.local", "https://photon.local?x=1", "https://photon.local/#fragment"}
	for _, value := range invalid {
		if _, err := NewPhotonGeocoder(value); err == nil {
			t.Errorf("expected invalid Photon URL %q to fail", value)
		}
	}
	public, err := NewPhotonGeocoder(PhotonPublicBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if public.limiter == nil || public.limiter.interval != photonPublicRequestInterval {
		t.Fatalf("public Photon endpoint must use conservative pacing: %+v", public.limiter)
	}
}
