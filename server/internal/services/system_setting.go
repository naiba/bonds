package services

import (
	"errors"
	"strconv"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/secret"
	"gorm.io/gorm"
)

var (
	ErrSystemSettingNotFound = errors.New("system setting not found")
)

// RedactedSecretValue is the placeholder returned by GetAllRedacted for keys
// listed in SecretSettingKeys. Writes that submit this exact value are
// treated as "leave the existing secret untouched".
const RedactedSecretValue = "***"

// SecretSettingKeys is the canonical list of system_settings rows that hold
// credentials. Values for these keys are encrypted at rest when
// SETTINGS_ENC_KEY is configured and redacted from admin reads.
var SecretSettingKeys = map[string]bool{
	"smtp.password":     true,
	"geocoding.api_key": true,
}

func IsSecretKey(key string) bool {
	if SecretSettingKeys[key] {
		return true
	}
	return strings.HasPrefix(key, "secret.")
}

type SystemSettingService struct {
	db     *gorm.DB
	cipher *secret.Cipher
}

func NewSystemSettingService(db *gorm.DB) *SystemSettingService {
	return &SystemSettingService{db: db, cipher: secret.New("")}
}

// NewSystemSettingServiceWithCipher returns a service that transparently
// encrypts SecretSettingKeys at rest. Pass an empty key to keep legacy
// plaintext storage.
func NewSystemSettingServiceWithCipher(db *gorm.DB, encKey string) *SystemSettingService {
	return &SystemSettingService{db: db, cipher: secret.New(encKey)}
}

// EncryptionEnabled reports whether SETTINGS_ENC_KEY is configured.
func (s *SystemSettingService) EncryptionEnabled() bool {
	return s.cipher.Enabled()
}

func (s *SystemSettingService) Get(key string) (string, error) {
	var setting models.SystemSetting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSystemSettingNotFound
		}
		return "", err
	}
	return s.decode(setting.Value)
}

func (s *SystemSettingService) GetWithDefault(key, defaultVal string) string {
	val, err := s.Get(key)
	if err != nil {
		return defaultVal
	}
	return val
}

func (s *SystemSettingService) GetBool(key string, defaultVal bool) bool {
	val, err := s.Get(key)
	if err != nil {
		return defaultVal
	}
	return val == "true" || val == "1"
}

func (s *SystemSettingService) GetInt(key string, defaultVal int) int {
	val, err := s.Get(key)
	if err != nil {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func (s *SystemSettingService) GetInt64(key string, defaultVal int64) int64 {
	val, err := s.Get(key)
	if err != nil {
		return defaultVal
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return i
}

func (s *SystemSettingService) Set(key, value string) error {
	stored, err := s.encode(key, value)
	if err != nil {
		return err
	}
	var setting models.SystemSetting
	err = s.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting = models.SystemSetting{Key: key, Value: stored}
			return s.db.Create(&setting).Error
		}
		return err
	}
	return s.db.Model(&setting).Update("value", stored).Error
}

// GetAll returns every setting decrypted. This is the read path for internal
// callers that need actual values (e.g. mailer, geocoder bootstrap).
func (s *SystemSettingService) GetAll() ([]dto.SystemSettingItem, error) {
	var settings []models.SystemSetting
	if err := s.db.Order("key ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	result := make([]dto.SystemSettingItem, len(settings))
	for i, setting := range settings {
		val, err := s.decode(setting.Value)
		if err != nil {
			return nil, err
		}
		result[i] = dto.SystemSettingItem{Key: setting.Key, Value: val}
	}
	return result, nil
}

// GetAllRedacted is the read path for the admin API. Secret keys are masked
// with RedactedSecretValue so admin consoles never receive plaintext
// credentials over the wire.
func (s *SystemSettingService) GetAllRedacted() ([]dto.SystemSettingItem, error) {
	var settings []models.SystemSetting
	if err := s.db.Order("key ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	result := make([]dto.SystemSettingItem, len(settings))
	for i, setting := range settings {
		if IsSecretKey(setting.Key) {
			masked := ""
			if setting.Value != "" {
				masked = RedactedSecretValue
			}
			result[i] = dto.SystemSettingItem{Key: setting.Key, Value: masked}
			continue
		}
		val, err := s.decode(setting.Value)
		if err != nil {
			return nil, err
		}
		result[i] = dto.SystemSettingItem{Key: setting.Key, Value: val}
	}
	return result, nil
}

// BulkSet upserts items in a transaction. For secret keys, the sentinel
// RedactedSecretValue means "keep current value" so admin UIs can round-trip
// the redacted GET response without wiping credentials.
func (s *SystemSettingService) BulkSet(items []dto.SystemSettingItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if IsSecretKey(item.Key) && item.Value == RedactedSecretValue {
				continue
			}
			stored, err := s.encode(item.Key, item.Value)
			if err != nil {
				return err
			}
			var setting models.SystemSetting
			err = tx.Where("key = ?", item.Key).First(&setting).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					setting = models.SystemSetting{Key: item.Key, Value: stored}
					if err := tx.Create(&setting).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			if err := tx.Model(&setting).Update("value", stored).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SystemSettingService) Delete(key string) error {
	result := s.db.Where("key = ?", key).Delete(&models.SystemSetting{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSystemSettingNotFound
	}
	return nil
}

// MigratePlaintextSecrets re-encrypts plaintext rows for SecretSettingKeys.
// It is idempotent (rows already encrypted are skipped) and safe to run on
// every boot. No-op when encryption is disabled.
func (s *SystemSettingService) MigratePlaintextSecrets() (int, error) {
	if !s.cipher.Enabled() {
		return 0, nil
	}
	var settings []models.SystemSetting
	if err := s.db.Find(&settings).Error; err != nil {
		return 0, err
	}
	migrated := 0
	for _, setting := range settings {
		if !IsSecretKey(setting.Key) {
			continue
		}
		if setting.Value == "" || secret.IsCiphertext(setting.Value) {
			continue
		}
		ct, err := s.cipher.Encrypt(setting.Value)
		if err != nil {
			return migrated, err
		}
		if err := s.db.Model(&models.SystemSetting{}).
			Where("key = ?", setting.Key).
			Update("value", ct).Error; err != nil {
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}

func (s *SystemSettingService) encode(key, value string) (string, error) {
	if !IsSecretKey(key) || value == "" {
		return value, nil
	}
	return s.cipher.Encrypt(value)
}

func (s *SystemSettingService) decode(stored string) (string, error) {
	return s.cipher.Decrypt(stored)
}
