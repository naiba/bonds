package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const photonPublicRequestInterval = time.Second

type PhotonGeocoder struct {
	client  *http.Client
	baseURL string
	limiter *rateLimiter
}

func NewPhotonGeocoder(baseURL string) (*PhotonGeocoder, error) {
	normalized, err := normalizePhotonBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	var limiter *rateLimiter
	parsed, _ := url.Parse(normalized)
	if strings.EqualFold(parsed.Hostname(), "photon.komoot.io") {
		// The public endpoint is explicitly a demo/fair-use service. Photon does
		// not publish a numeric quota, so pace it conservatively and tell admins
		// to self-host for production or extensive use.
		limiter = &rateLimiter{interval: photonPublicRequestInterval}
	}
	return &PhotonGeocoder{
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: normalized,
		limiter: limiter,
	}, nil
}

func normalizePhotonBaseURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("Photon base_url must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("Photon base_url cannot contain credentials, a query, or a fragment")
	}
	return value, nil
}

func (g *PhotonGeocoder) Geocode(address string) (*GeocodingResult, error) {
	features, err := g.fetch(address, 1, time.Minute)
	if err != nil {
		return nil, err
	}
	if len(features) == 0 || len(features[0].Geometry.Coordinates) < 2 {
		return nil, nil
	}
	return &GeocodingResult{
		Latitude:  features[0].Geometry.Coordinates[1],
		Longitude: features[0].Geometry.Coordinates[0],
	}, nil
}

func (g *PhotonGeocoder) Suggest(query string, limit int) ([]GeocodingSuggestion, error) {
	query = truncateQuery(strings.TrimSpace(query), suggestQueryMaxLength)
	if query == "" {
		return []GeocodingSuggestion{}, nil
	}
	features, err := g.fetch(query, normalizeSuggestionLimit(limit), suggestWait)
	if err != nil {
		return nil, err
	}
	suggestions := make([]GeocodingSuggestion, 0, len(features))
	for _, feature := range features {
		if len(feature.Geometry.Coordinates) < 2 {
			continue
		}
		properties := feature.Properties
		city := firstNonEmpty(properties.City, properties.Town, properties.Village, properties.Locality, properties.District)
		street := properties.Street
		if street == "" && properties.HouseNumber != "" {
			street = properties.Name
		}
		line1 := strings.TrimSpace(strings.Join([]string{properties.HouseNumber, street}, " "))
		if line1 == "" && properties.Name != city && properties.Name != properties.Country {
			line1 = properties.Name
		}
		labelParts := []string{properties.Name, city, properties.State, properties.Country}
		label := joinUniqueNonEmpty(labelParts...)
		suggestions = append(suggestions, GeocodingSuggestion{
			Label:      label,
			Line1:      line1,
			City:       city,
			Province:   firstNonEmpty(properties.State, properties.County),
			PostalCode: properties.Postcode,
			Country:    properties.Country,
			Latitude:   feature.Geometry.Coordinates[1],
			Longitude:  feature.Geometry.Coordinates[0],
		})
	}
	return suggestions, nil
}

func (g *PhotonGeocoder) SupportsAutocomplete() bool { return true }

func (g *PhotonGeocoder) Attribution() []ProviderAttribution {
	return []ProviderAttribution{
		osmAttribution,
		{Label: "Powered by Photon", URL: "https://github.com/komoot/photon"},
	}
}

type photonProperties struct {
	Name        string `json:"name"`
	HouseNumber string `json:"housenumber"`
	Street      string `json:"street"`
	City        string `json:"city"`
	Town        string `json:"town"`
	Village     string `json:"village"`
	Locality    string `json:"locality"`
	District    string `json:"district"`
	County      string `json:"county"`
	State       string `json:"state"`
	Postcode    string `json:"postcode"`
	Country     string `json:"country"`
}

type photonFeature struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties photonProperties `json:"properties"`
}

type photonResponse struct {
	Features []photonFeature `json:"features"`
}

func (g *PhotonGeocoder) fetch(query string, limit int, maxWait time.Duration) ([]photonFeature, error) {
	if err := g.limiter.wait(maxWait); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	req, err := http.NewRequest(http.MethodGet, g.baseURL+"/api?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Photon request: %w", err)
	}
	req.Header.Set("User-Agent", "Bonds/1.0")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Photon request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Photon returned status %d", resp.StatusCode)
	}
	var decoded photonResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to parse Photon response: %w", err)
	}
	return decoded.Features, nil
}

func joinUniqueNonEmpty(values ...string) string {
	seen := map[string]struct{}{}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, value)
	}
	return strings.Join(parts, ", ")
}
