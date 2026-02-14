package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type GeocodingResult struct {
	Latitude  float64
	Longitude float64
}

type Geocoder interface {
	Geocode(address string) (*GeocodingResult, error)
}

type NominatimGeocoder struct {
	client *http.Client
}

func NewNominatimGeocoder() *NominatimGeocoder {
	return &NominatimGeocoder{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *NominatimGeocoder) Geocode(address string) (*GeocodingResult, error) {
	return geocodeFromURL(g.client, "https://nominatim.openstreetmap.org/search", address, "")
}

type LocationIQGeocoder struct {
	client *http.Client
	apiKey string
}

func NewLocationIQGeocoder(apiKey string) *LocationIQGeocoder {
	return &LocationIQGeocoder{
		client: &http.Client{Timeout: 10 * time.Second},
		apiKey: apiKey,
	}
}

func (g *LocationIQGeocoder) Geocode(address string) (*GeocodingResult, error) {
	return geocodeFromURL(g.client, "https://us1.locationiq.com/v1/search", address, g.apiKey)
}

func NewGeocoder(provider, apiKey string) Geocoder {
	switch provider {
	case "locationiq":
		return NewLocationIQGeocoder(apiKey)
	default:
		return NewNominatimGeocoder()
	}
}

type nominatimResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

func geocodeFromURL(client *http.Client, baseURL, address, apiKey string) (*GeocodingResult, error) {
	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "json")
	params.Set("limit", "1")
	if apiKey != "" {
		params.Set("key", apiKey)
	}

	reqURL := baseURL + "?" + params.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create geocoding request: %w", err)
	}
	req.Header.Set("User-Agent", "Bonds/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding returned status %d", resp.StatusCode)
	}

	var results []nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse longitude: %w", err)
	}

	return &GeocodingResult{Latitude: lat, Longitude: lon}, nil
}
