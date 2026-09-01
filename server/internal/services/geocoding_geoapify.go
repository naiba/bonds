package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	geoapifySearchURL       = "https://api.geoapify.com/v1/geocode/search"
	geoapifyAutocompleteURL = "https://api.geoapify.com/v1/geocode/autocomplete"
)

type GeoapifyGeocoder struct {
	client          *http.Client
	apiKey          string
	limiter         *rateLimiter
	searchURL       string
	autocompleteURL string
}

func NewGeoapifyGeocoder(apiKey string) *GeoapifyGeocoder {
	return &GeoapifyGeocoder{
		client:          &http.Client{Timeout: 10 * time.Second},
		apiKey:          apiKey,
		limiter:         &rateLimiter{interval: 200 * time.Millisecond},
		searchURL:       geoapifySearchURL,
		autocompleteURL: geoapifyAutocompleteURL,
	}
}

func (g *GeoapifyGeocoder) Geocode(address string) (*GeocodingResult, error) {
	results, err := g.fetch(g.searchURL, address, 1, time.Minute)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &GeocodingResult{Latitude: results[0].Lat, Longitude: results[0].Lon}, nil
}

func (g *GeoapifyGeocoder) Suggest(query string, limit int) ([]GeocodingSuggestion, error) {
	query = truncateQuery(strings.TrimSpace(query), suggestQueryMaxLength)
	if query == "" {
		return []GeocodingSuggestion{}, nil
	}
	limit = normalizeSuggestionLimit(limit)
	results, err := g.fetch(g.autocompleteURL, query, limit, suggestWait)
	if err != nil {
		return nil, err
	}
	suggestions := make([]GeocodingSuggestion, 0, len(results))
	for _, result := range results {
		line1 := result.AddressLine1
		if line1 == "" {
			line1 = strings.TrimSpace(strings.Join([]string{result.HouseNumber, result.Street}, " "))
		}
		if line1 == "" {
			line1 = result.Name
		}
		suggestions = append(suggestions, GeocodingSuggestion{
			Label:      firstNonEmpty(result.Formatted, result.AddressLine1, result.Name),
			Line1:      line1,
			City:       firstNonEmpty(result.City, result.Town, result.Village, result.Suburb),
			Province:   firstNonEmpty(result.State, result.County),
			PostalCode: result.Postcode,
			Country:    result.Country,
			Latitude:   result.Lat,
			Longitude:  result.Lon,
		})
	}
	return suggestions, nil
}

func (g *GeoapifyGeocoder) SupportsAutocomplete() bool { return true }

func (g *GeoapifyGeocoder) Attribution() []ProviderAttribution {
	return []ProviderAttribution{
		osmAttribution,
		{Label: "Powered by Geoapify", URL: "https://www.geoapify.com/"},
	}
}

type geoapifyResult struct {
	Formatted    string  `json:"formatted"`
	AddressLine1 string  `json:"address_line1"`
	Name         string  `json:"name"`
	HouseNumber  string  `json:"housenumber"`
	Street       string  `json:"street"`
	City         string  `json:"city"`
	Town         string  `json:"town"`
	Village      string  `json:"village"`
	Suburb       string  `json:"suburb"`
	County       string  `json:"county"`
	State        string  `json:"state"`
	Postcode     string  `json:"postcode"`
	Country      string  `json:"country"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
}

type geoapifyResponse struct {
	Results []geoapifyResult `json:"results"`
}

func (g *GeoapifyGeocoder) fetch(baseURL, query string, limit int, maxWait time.Duration) ([]geoapifyResult, error) {
	if err := g.limiter.wait(maxWait); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("text", query)
	params.Set("format", "json")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("apiKey", g.apiKey)
	req, err := http.NewRequest(http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Geoapify request: %w", err)
	}
	req.Header.Set("User-Agent", "Bonds/1.0")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Geoapify request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Geoapify returned status %d", resp.StatusCode)
	}
	var decoded geoapifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to parse Geoapify response: %w", err)
	}
	return decoded.Results, nil
}
