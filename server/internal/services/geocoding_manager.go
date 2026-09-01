package services

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/naiba/bonds/internal/dto"
)

// GeocodingManager owns persisted provider selection/configuration and swaps a
// fully-built runtime into AddressService. Its mutex serializes admin changes;
// AddressService then serves requests from an immutable snapshot.
type GeocodingManager struct {
	mu sync.Mutex

	settings       *SystemSettingService
	configs        *GeocodingProviderConfigService
	registry       *GeocodingProviderRegistry
	addressService *AddressService

	defaultProvider  string
	defaultPrecision string
}

func NewGeocodingManager(
	settings *SystemSettingService,
	configs *GeocodingProviderConfigService,
	registry *GeocodingProviderRegistry,
	addressService *AddressService,
	defaultProvider, defaultPrecision string,
) *GeocodingManager {
	return &GeocodingManager{
		settings: settings, configs: configs, registry: registry, addressService: addressService,
		defaultProvider: defaultProvider, defaultPrecision: defaultPrecision,
	}
}

func (m *GeocodingManager) Initialize() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	migrated, err := m.configs.MigratePlaintextSecrets()
	if err != nil {
		return migrated, err
	}
	return migrated, m.reloadLocked()
}

func (m *GeocodingManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadLocked()
}

func (m *GeocodingManager) reloadLocked() error {
	provider := m.settings.GetWithDefault("geocoding.provider", m.defaultProvider)
	precision := m.settings.GetWithDefault("geocoding.precision", m.defaultPrecision)
	if provider == "" {
		m.addressService.ConfigureGeocoding(nil, precision)
		return nil
	}
	config, _, err := m.configs.Effective(provider)
	if err != nil {
		m.addressService.ConfigureGeocoding(nil, precision)
		return err
	}
	geocoder, err := m.registry.Build(provider, config)
	if err != nil {
		m.addressService.ConfigureGeocoding(nil, precision)
		return err
	}
	m.addressService.ConfigureGeocoding(geocoder, precision)
	return nil
}

func (m *GeocodingManager) Get() (*dto.GeocodingAdminResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getLocked()
}

func (m *GeocodingManager) getLocked() (*dto.GeocodingAdminResponse, error) {
	response := &dto.GeocodingAdminResponse{
		ActiveProvider: m.settings.GetWithDefault("geocoding.provider", m.defaultProvider),
		Precision:      normalizeGeocodingPrecision(m.settings.GetWithDefault("geocoding.precision", m.defaultPrecision)),
		Providers:      []dto.GeocodingProviderResponse{},
	}
	for _, definition := range m.registry.Definitions() {
		plain, hasStoredConfig, err := m.configs.Effective(definition.ID)
		if err != nil {
			return nil, err
		}
		_, buildErr := m.registry.Build(definition.ID, plain)
		redacted, _, err := m.configs.Redacted(definition.ID)
		if err != nil {
			return nil, err
		}
		notice := definition.Notice
		if definition.ID == GeocodingProviderPhoton && !isPublicPhotonURL(plain["base_url"]) {
			notice = ""
		}
		provider := dto.GeocodingProviderResponse{
			ID:                   definition.ID,
			Name:                 definition.Name,
			Configured:           buildErr == nil,
			HasStoredConfig:      hasStoredConfig,
			SupportsAutocomplete: definition.SupportsAutocomplete,
			Notice:               notice,
			Fields:               []dto.GeocodingProviderFieldResponse{},
			Config:               redacted,
			Attribution:          []dto.AddressAttribution{},
		}
		for _, field := range definition.Fields {
			provider.Fields = append(provider.Fields, dto.GeocodingProviderFieldResponse{
				Key: field.Key, Type: field.Type, Required: field.Required, Secret: field.Secret,
			})
		}
		for _, credit := range definition.Attribution {
			provider.Attribution = append(provider.Attribution, dto.AddressAttribution{Label: credit.Label, URL: credit.URL})
		}
		response.Providers = append(response.Providers, provider)
	}
	return response, nil
}

func (m *GeocodingManager) UpdateSettings(provider, precision string) (*dto.GeocodingAdminResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	provider = strings.TrimSpace(provider)
	if precision != GeocodingPrecisionExact && precision != GeocodingPrecisionLocality {
		return nil, fmt.Errorf("%w: unsupported precision %q", ErrInvalidGeocodingConfig, precision)
	}
	if provider != "" {
		config, _, err := m.configs.Effective(provider)
		if err != nil {
			return nil, err
		}
		if _, err := m.registry.Build(provider, config); err != nil {
			if errors.Is(err, ErrInvalidGeocodingConfig) {
				return nil, fmt.Errorf("%w: %s", ErrGeocodingNotConfigured, provider)
			}
			return nil, err
		}
	}
	if err := m.settings.BulkSet([]dto.SystemSettingItem{
		{Key: "geocoding.provider", Value: provider},
		{Key: "geocoding.precision", Value: precision},
	}); err != nil {
		return nil, err
	}
	if err := m.reloadLocked(); err != nil {
		return nil, err
	}
	return m.getLocked()
}

func (m *GeocodingManager) SaveProvider(provider string, config map[string]string) (*dto.GeocodingAdminResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	definition, err := m.registry.Definition(provider)
	if err != nil {
		return nil, err
	}
	provider = definition.ID
	if _, err := m.configs.Save(provider, config); err != nil {
		return nil, err
	}
	if m.settings.GetWithDefault("geocoding.provider", m.defaultProvider) == provider {
		if err := m.reloadLocked(); err != nil {
			return nil, err
		}
	}
	return m.getLocked()
}

func (m *GeocodingManager) DeleteProvider(provider string) (*dto.GeocodingAdminResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	definition, err := m.registry.Definition(provider)
	if err != nil {
		return nil, err
	}
	provider = definition.ID
	if err := m.configs.Delete(provider); err != nil {
		return nil, err
	}
	if m.settings.GetWithDefault("geocoding.provider", m.defaultProvider) == provider {
		if err := m.reloadLocked(); err != nil {
			// A keyed active provider becomes deliberately unavailable after its
			// only config is deleted. The deletion succeeded; expose that state
			// instead of resurrecting stale credentials.
			m.addressService.ConfigureGeocoding(nil, m.settings.GetWithDefault("geocoding.precision", m.defaultPrecision))
		}
	}
	return m.getLocked()
}

func isPublicPhotonURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && strings.EqualFold(parsed.Hostname(), "photon.komoot.io")
}
