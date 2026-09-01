package services

import (
	"errors"
	"fmt"
	"strings"
)

const (
	GeocodingProviderNominatim  = "nominatim"
	GeocodingProviderLocationIQ = "locationiq"
	GeocodingProviderGeoapify   = "geoapify"
	GeocodingProviderPhoton     = "photon"

	PhotonPublicBaseURL = "https://photon.komoot.io"
)

var (
	ErrUnknownGeocodingProvider = errors.New("unknown geocoding provider")
	ErrInvalidGeocodingConfig   = errors.New("invalid geocoding provider configuration")
	ErrGeocodingNotConfigured   = errors.New("geocoding provider is not configured")
)

type GeocodingProviderConfigField struct {
	Key      string
	Type     string
	Required bool
	Secret   bool
}

// GeocodingProviderDefinition is code-owned provider metadata plus the one
// constructor that understands its structured configuration. Adding another
// provider is deliberately local to this registry and its Geocoder adapter.
type GeocodingProviderDefinition struct {
	ID                   string
	Name                 string
	Notice               string
	SupportsAutocomplete bool
	Fields               []GeocodingProviderConfigField
	Defaults             map[string]string
	Attribution          []ProviderAttribution
	New                  func(config map[string]string) (Geocoder, error)
}

type GeocodingProviderRegistry struct {
	ordered []GeocodingProviderDefinition
	byID    map[string]GeocodingProviderDefinition
}

func NewGeocodingProviderRegistry() *GeocodingProviderRegistry {
	definitions := []GeocodingProviderDefinition{
		{
			ID:                   GeocodingProviderNominatim,
			Name:                 "Nominatim",
			Notice:               "public_nominatim",
			SupportsAutocomplete: false,
			Attribution:          []ProviderAttribution{osmAttribution},
			New: func(_ map[string]string) (Geocoder, error) {
				return NewNominatimGeocoder(), nil
			},
		},
		{
			ID:                   GeocodingProviderLocationIQ,
			Name:                 "LocationIQ",
			SupportsAutocomplete: true,
			Fields: []GeocodingProviderConfigField{
				{Key: "api_key", Type: "password", Required: true, Secret: true},
			},
			Attribution: []ProviderAttribution{
				osmAttribution,
				{Label: "Search by LocationIQ.com", URL: "https://locationiq.com/attribution"},
			},
			New: func(config map[string]string) (Geocoder, error) {
				return NewLocationIQGeocoder(config["api_key"]), nil
			},
		},
		{
			ID:                   GeocodingProviderGeoapify,
			Name:                 "Geoapify",
			SupportsAutocomplete: true,
			Fields: []GeocodingProviderConfigField{
				{Key: "api_key", Type: "password", Required: true, Secret: true},
			},
			Attribution: []ProviderAttribution{
				osmAttribution,
				{Label: "Powered by Geoapify", URL: "https://www.geoapify.com/"},
			},
			New: func(config map[string]string) (Geocoder, error) {
				return NewGeoapifyGeocoder(config["api_key"]), nil
			},
		},
		{
			ID:                   GeocodingProviderPhoton,
			Name:                 "Photon",
			Notice:               "public_demo",
			SupportsAutocomplete: true,
			Fields: []GeocodingProviderConfigField{
				{Key: "base_url", Type: "url", Required: true},
			},
			Defaults: map[string]string{"base_url": PhotonPublicBaseURL},
			Attribution: []ProviderAttribution{
				osmAttribution,
				{Label: "Powered by Photon", URL: "https://github.com/komoot/photon"},
			},
			New: func(config map[string]string) (Geocoder, error) {
				return NewPhotonGeocoder(config["base_url"])
			},
		},
	}

	byID := make(map[string]GeocodingProviderDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	return &GeocodingProviderRegistry{ordered: definitions, byID: byID}
}

func (r *GeocodingProviderRegistry) Definitions() []GeocodingProviderDefinition {
	result := make([]GeocodingProviderDefinition, len(r.ordered))
	copy(result, r.ordered)
	return result
}

func (r *GeocodingProviderRegistry) Definition(provider string) (GeocodingProviderDefinition, error) {
	definition, ok := r.byID[strings.TrimSpace(provider)]
	if !ok {
		return GeocodingProviderDefinition{}, fmt.Errorf("%w: %s", ErrUnknownGeocodingProvider, provider)
	}
	return definition, nil
}

// Normalize validates a provider configuration, applies code-owned defaults,
// and returns only declared fields. Unknown keys are rejected so a typo cannot
// become silently persisted pseudo-configuration.
func (r *GeocodingProviderRegistry) Normalize(provider string, config map[string]string) (map[string]string, error) {
	definition, err := r.Definition(provider)
	if err != nil {
		return nil, err
	}

	known := make(map[string]GeocodingProviderConfigField, len(definition.Fields))
	for _, field := range definition.Fields {
		known[field.Key] = field
	}
	for key := range config {
		if _, ok := known[key]; !ok {
			return nil, fmt.Errorf("%w: %s does not define %q", ErrInvalidGeocodingConfig, provider, key)
		}
	}

	normalized := make(map[string]string, len(definition.Fields))
	for _, field := range definition.Fields {
		value := strings.TrimSpace(config[field.Key])
		if value == "" {
			value = definition.Defaults[field.Key]
		}
		if field.Required && value == "" {
			return nil, fmt.Errorf("%w: %s.%s is required", ErrInvalidGeocodingConfig, provider, field.Key)
		}
		normalized[field.Key] = value
	}
	return normalized, nil
}

func (r *GeocodingProviderRegistry) Build(provider string, config map[string]string) (Geocoder, error) {
	normalized, err := r.Normalize(provider, config)
	if err != nil {
		return nil, err
	}
	definition, _ := r.Definition(provider)
	geocoder, err := definition.New(normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGeocodingConfig, err)
	}
	return geocoder, nil
}
