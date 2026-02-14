package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNominatimGeocoderSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Bonds/1.0" {
			t.Errorf("Expected User-Agent 'Bonds/1.0', got '%s'", r.Header.Get("User-Agent"))
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("Expected non-empty query parameter 'q'")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"lat":"40.7128","lon":"-74.0060"}]`))
	}))
	defer server.Close()

	geocoder := &NominatimGeocoder{client: server.Client()}
	result, err := geocodeFromURL(geocoder.client, server.URL, "New York", "")
	if err != nil {
		t.Fatalf("Geocode failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Latitude != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %f", result.Latitude)
	}
	if result.Longitude != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %f", result.Longitude)
	}
}

func TestGeocoderEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	geocoder := &NominatimGeocoder{client: server.Client()}
	result, err := geocodeFromURL(geocoder.client, server.URL, "nonexistent place xyz", "")
	if err != nil {
		t.Fatalf("Geocode failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result for empty response, got %+v", result)
	}
}

func TestGeocoderMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	geocoder := &NominatimGeocoder{client: server.Client()}
	_, err := geocodeFromURL(geocoder.client, server.URL, "test", "")
	if err == nil {
		t.Error("Expected error for malformed JSON")
	}
}

func TestGeocoderServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	geocoder := &NominatimGeocoder{client: server.Client()}
	_, err := geocodeFromURL(geocoder.client, server.URL, "test", "")
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestLocationIQGeocoderPassesAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key != "test-api-key" {
			t.Errorf("Expected API key 'test-api-key', got '%s'", key)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"lat":"51.5074","lon":"-0.1278"}]`))
	}))
	defer server.Close()

	geocoder := &LocationIQGeocoder{client: server.Client(), apiKey: "test-api-key"}
	result, err := geocodeFromURL(geocoder.client, server.URL, "London", geocoder.apiKey)
	if err != nil {
		t.Fatalf("Geocode failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Latitude != 51.5074 {
		t.Errorf("Expected latitude 51.5074, got %f", result.Latitude)
	}
}

func TestNewGeocoderFactory(t *testing.T) {
	g := NewGeocoder("nominatim", "")
	if _, ok := g.(*NominatimGeocoder); !ok {
		t.Error("Expected NominatimGeocoder for default provider")
	}

	g = NewGeocoder("locationiq", "some-key")
	if _, ok := g.(*LocationIQGeocoder); !ok {
		t.Error("Expected LocationIQGeocoder for 'locationiq' provider")
	}

	g = NewGeocoder("unknown", "")
	if _, ok := g.(*NominatimGeocoder); !ok {
		t.Error("Expected NominatimGeocoder for unknown provider")
	}
}
