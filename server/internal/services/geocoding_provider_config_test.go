package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"github.com/naiba/bonds/pkg/secret"
)

func TestGeocodingProviderRegistryValidatesStructuredConfigs(t *testing.T) {
	registry := NewGeocodingProviderRegistry()

	photon, err := registry.Normalize(GeocodingProviderPhoton, map[string]string{})
	if err != nil {
		t.Fatalf("Photon defaults should be valid: %v", err)
	}
	if photon["base_url"] != PhotonPublicBaseURL {
		t.Fatalf("unexpected Photon default: %q", photon["base_url"])
	}
	if _, err := registry.Build(GeocodingProviderPhoton, map[string]string{"base_url": "file:///tmp/photon"}); !errors.Is(err, ErrInvalidGeocodingConfig) {
		t.Fatalf("invalid Photon URL should be rejected, got %v", err)
	}
	if _, err := registry.Build(GeocodingProviderLocationIQ, map[string]string{}); !errors.Is(err, ErrInvalidGeocodingConfig) {
		t.Fatalf("missing LocationIQ key should be rejected, got %v", err)
	}
	if _, err := registry.Normalize(GeocodingProviderGeoapify, map[string]string{"api_key": "key", "typo": "value"}); !errors.Is(err, ErrInvalidGeocodingConfig) {
		t.Fatalf("unknown config fields should be rejected, got %v", err)
	}
	if _, err := registry.Definition("made-up"); !errors.Is(err, ErrUnknownGeocodingProvider) {
		t.Fatalf("unknown provider should be rejected, got %v", err)
	}
}

func TestGeocodingProviderConfigOneEncryptedRowPerProvider(t *testing.T) {
	db := testutil.SetupTestDB(t)
	registry := NewGeocodingProviderRegistry()
	svc := NewGeocodingProviderConfigService(db, "settings-key", registry)

	if _, err := svc.Save(GeocodingProviderLocationIQ, map[string]string{"api_key": "first-secret"}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	var row models.GeocodingProviderConfig
	if err := db.Where("provider = ?", GeocodingProviderLocationIQ).First(&row).Error; err != nil {
		t.Fatalf("read raw row: %v", err)
	}
	if !secret.IsCiphertext(row.Config) || strings.Contains(row.Config, "first-secret") {
		t.Fatalf("provider config was not encrypted at rest: %q", row.Config)
	}

	redacted, exists, err := svc.Redacted(GeocodingProviderLocationIQ)
	if err != nil || !exists {
		t.Fatalf("redacted config: exists=%v err=%v", exists, err)
	}
	if redacted["api_key"] != RedactedSecretValue {
		t.Fatalf("API key leaked from redacted response: %q", redacted["api_key"])
	}
	if _, err := svc.Save(GeocodingProviderLocationIQ, map[string]string{"api_key": RedactedSecretValue}); err != nil {
		t.Fatalf("redaction sentinel save: %v", err)
	}
	stored, _, err := svc.GetStored(GeocodingProviderLocationIQ)
	if err != nil || stored["api_key"] != "first-secret" {
		t.Fatalf("redaction sentinel did not preserve secret: config=%v err=%v", stored, err)
	}

	if _, err := svc.Save("  "+GeocodingProviderLocationIQ+"  ", map[string]string{"api_key": "second-secret"}); err != nil {
		t.Fatalf("replacement save: %v", err)
	}
	var count int64
	if err := db.Model(&models.GeocodingProviderConfig{}).Where("provider = ?", GeocodingProviderLocationIQ).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one LocationIQ config row, got %d", count)
	}
	stored, _, _ = svc.GetStored(GeocodingProviderLocationIQ)
	if stored["api_key"] != "second-secret" {
		t.Fatalf("upsert did not replace the one provider row: %v", stored)
	}
}

func TestGeocodingProviderConfigMigratesPlaintextIdempotently(t *testing.T) {
	db := testutil.SetupTestDB(t)
	row := models.GeocodingProviderConfig{
		Provider: GeocodingProviderGeoapify,
		Config:   `{"api_key":"legacy-key"}`,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed plaintext row: %v", err)
	}
	svc := NewGeocodingProviderConfigService(db, "settings-key", NewGeocodingProviderRegistry())

	migrated, err := svc.MigratePlaintextSecrets()
	if err != nil || migrated != 1 {
		t.Fatalf("first migration: count=%d err=%v", migrated, err)
	}
	if err := db.First(&row, row.ID).Error; err != nil {
		t.Fatalf("reload migrated row: %v", err)
	}
	if !secret.IsCiphertext(row.Config) {
		t.Fatalf("plaintext row was not encrypted: %q", row.Config)
	}
	migrated, err = svc.MigratePlaintextSecrets()
	if err != nil || migrated != 0 {
		t.Fatalf("second migration should be a no-op: count=%d err=%v", migrated, err)
	}
	config, exists, err := svc.GetStored(GeocodingProviderGeoapify)
	if err != nil || !exists || config["api_key"] != "legacy-key" {
		t.Fatalf("migrated config unreadable: config=%v exists=%v err=%v", config, exists, err)
	}
}
