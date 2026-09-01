package services

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupGeocodingManagerTest(t *testing.T, encKey string) (*GeocodingManager, *SystemSettingService, *GeocodingProviderConfigService, *AddressService) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	settings := NewSystemSettingServiceWithCipher(db, encKey)
	registry := NewGeocodingProviderRegistry()
	configs := NewGeocodingProviderConfigService(db, encKey, registry)
	address := NewAddressService(db)
	manager := NewGeocodingManager(settings, configs, registry, address, "", GeocodingPrecisionExact)
	return manager, settings, configs, address
}

func TestGeocodingManagerAtomicallyReloadsAndDeletesActiveConfig(t *testing.T) {
	manager, settings, _, address := setupGeocodingManagerTest(t, "")
	if err := settings.Set("geocoding.provider", GeocodingProviderLocationIQ); err != nil {
		t.Fatal(err)
	}
	if err := settings.Set("geocoding.precision", GeocodingPrecisionExact); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SaveProvider(GeocodingProviderLocationIQ, map[string]string{"api_key": "first"}); err != nil {
		t.Fatalf("save active provider: %v", err)
	}
	first, ok := address.geocodingSnapshot().geocoder.(*LocationIQGeocoder)
	if !ok || first.apiKey != "first" {
		t.Fatalf("first runtime mismatch: %T", address.geocodingSnapshot().geocoder)
	}
	if _, err := manager.SaveProvider(GeocodingProviderLocationIQ, map[string]string{"api_key": "second"}); err != nil {
		t.Fatalf("replace active provider: %v", err)
	}
	second, ok := address.geocodingSnapshot().geocoder.(*LocationIQGeocoder)
	if !ok || second.apiKey != "second" || first == second {
		t.Fatalf("runtime was not replaced atomically: first=%p second=%p", first, second)
	}

	result, err := manager.DeleteProvider(GeocodingProviderLocationIQ)
	if err != nil {
		t.Fatalf("delete active provider: %v", err)
	}
	if address.geocodingSnapshot().geocoder != nil {
		t.Fatalf("deleted credential remained active: %T", address.geocodingSnapshot().geocoder)
	}
	for _, provider := range result.Providers {
		if provider.ID == GeocodingProviderLocationIQ && provider.Configured {
			t.Fatal("deleted LocationIQ provider still reports configured")
		}
	}
}

func TestGeocodingManagerPhotonDefaultsAndOneRowConstraint(t *testing.T) {
	manager, settings, _, address := setupGeocodingManagerTest(t, "")
	manager.defaultProvider = GeocodingProviderPhoton
	if err := settings.Set("geocoding.provider", GeocodingProviderPhoton); err != nil {
		t.Fatal(err)
	}
	if err := settings.Set("geocoding.precision", GeocodingPrecisionExact); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize Photon: %v", err)
	}
	photon, ok := address.geocodingSnapshot().geocoder.(*PhotonGeocoder)
	if !ok || photon.baseURL != PhotonPublicBaseURL {
		t.Fatalf("public Photon default not active: %T %+v", address.geocodingSnapshot().geocoder, photon)
	}

	if _, err := manager.SaveProvider(GeocodingProviderPhoton, map[string]string{"base_url": "https://photon.internal.example"}); err != nil {
		t.Fatalf("save self-hosted Photon: %v", err)
	}
	if _, err := manager.SaveProvider(GeocodingProviderPhoton, map[string]string{"base_url": "https://photon-2.internal.example"}); err != nil {
		t.Fatalf("replace self-hosted Photon: %v", err)
	}
	var count int64
	if err := settings.db.Model(&models.GeocodingProviderConfig{}).Where("provider = ?", GeocodingProviderPhoton).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one Photon row, got %d", count)
	}
	if current := address.geocodingSnapshot().geocoder.(*PhotonGeocoder); current.baseURL != "https://photon-2.internal.example" || current.limiter != nil {
		t.Fatalf("self-hosted Photon runtime mismatch: %+v", current)
	}
}
