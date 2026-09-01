package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/secret"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GeocodingProviderConfigService persists one encrypted JSON configuration per
// provider. The unique provider index is the final guard against duplicates;
// Upsert makes edits replace that one row on both SQLite and PostgreSQL.
type GeocodingProviderConfigService struct {
	db       *gorm.DB
	cipher   *secret.Cipher
	registry *GeocodingProviderRegistry
}

func NewGeocodingProviderConfigService(db *gorm.DB, encKey string, registry *GeocodingProviderRegistry) *GeocodingProviderConfigService {
	return &GeocodingProviderConfigService{
		db:       db,
		cipher:   secret.New(encKey),
		registry: registry,
	}
}

func (s *GeocodingProviderConfigService) GetStored(provider string) (map[string]string, bool, error) {
	definition, err := s.registry.Definition(provider)
	if err != nil {
		return nil, false, err
	}
	provider = definition.ID
	var row models.GeocodingProviderConfig
	if err := s.db.Where("provider = ?", provider).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]string{}, false, nil
		}
		return nil, false, err
	}
	plain, err := s.cipher.Decrypt(row.Config)
	if err != nil {
		return nil, false, fmt.Errorf("decrypt geocoding config for %s: %w", provider, err)
	}
	config := map[string]string{}
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return nil, false, fmt.Errorf("decode geocoding config for %s: %w", provider, err)
	}
	return config, true, nil
}

// Effective returns stored values overlaid on code-owned defaults. It does not
// require the result to be configured: callers use this to render an empty API
// key field while Photon can be immediately usable with its public demo URL.
func (s *GeocodingProviderConfigService) Effective(provider string) (map[string]string, bool, error) {
	definition, err := s.registry.Definition(provider)
	if err != nil {
		return nil, false, err
	}
	config := make(map[string]string, len(definition.Fields))
	for key, value := range definition.Defaults {
		config[key] = value
	}
	stored, exists, err := s.GetStored(definition.ID)
	if err != nil {
		return nil, false, err
	}
	for key, value := range stored {
		config[key] = value
	}
	return config, exists, nil
}

func (s *GeocodingProviderConfigService) Save(provider string, incoming map[string]string) (map[string]string, error) {
	definition, err := s.registry.Definition(provider)
	if err != nil {
		return nil, err
	}
	provider = definition.ID
	existing, _, err := s.Effective(provider)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]string, len(incoming))
	for key, value := range incoming {
		merged[key] = value
	}
	for _, field := range definition.Fields {
		if field.Secret && merged[field.Key] == RedactedSecretValue {
			merged[field.Key] = existing[field.Key]
		}
	}
	normalized, err := s.registry.Normalize(provider, merged)
	if err != nil {
		return nil, err
	}
	// Build performs provider-specific validation such as Photon base URL
	// validation before any invalid configuration reaches the database.
	if _, err := s.registry.Build(provider, normalized); err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	stored, err := s.cipher.Encrypt(string(encoded))
	if err != nil {
		return nil, err
	}
	row := models.GeocodingProviderConfig{Provider: provider, Config: stored}
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"config", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	return normalized, nil
}

func (s *GeocodingProviderConfigService) Delete(provider string) error {
	definition, err := s.registry.Definition(provider)
	if err != nil {
		return err
	}
	return s.db.Where("provider = ?", definition.ID).Delete(&models.GeocodingProviderConfig{}).Error
}

func (s *GeocodingProviderConfigService) Redacted(provider string) (map[string]string, bool, error) {
	definition, err := s.registry.Definition(provider)
	if err != nil {
		return nil, false, err
	}
	config, exists, err := s.Effective(provider)
	if err != nil {
		return nil, false, err
	}
	for _, field := range definition.Fields {
		if field.Secret && config[field.Key] != "" {
			config[field.Key] = RedactedSecretValue
		}
	}
	return config, exists, nil
}

// MigratePlaintextSecrets encrypts legacy plaintext JSON rows after an
// operator enables SETTINGS_ENC_KEY. It is idempotent and leaves already
// encrypted rows untouched.
func (s *GeocodingProviderConfigService) MigratePlaintextSecrets() (int, error) {
	if !s.cipher.Enabled() {
		return 0, nil
	}
	var rows []models.GeocodingProviderConfig
	if err := s.db.Find(&rows).Error; err != nil {
		return 0, err
	}
	migrated := 0
	for _, row := range rows {
		if secret.IsCiphertext(row.Config) {
			continue
		}
		stored, err := s.cipher.Encrypt(row.Config)
		if err != nil {
			return migrated, err
		}
		if err := s.db.Model(&row).Update("config", stored).Error; err != nil {
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}
