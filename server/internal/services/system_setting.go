package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrSystemSettingNotFound = errors.New("system setting not found")
)

type SystemSettingService struct {
	db *gorm.DB
}

func NewSystemSettingService(db *gorm.DB) *SystemSettingService {
	return &SystemSettingService{db: db}
}

func (s *SystemSettingService) Get(key string) (string, error) {
	var setting models.SystemSetting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSystemSettingNotFound
		}
		return "", err
	}
	return setting.Value, nil
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

func (s *SystemSettingService) Set(key, value string) error {
	var setting models.SystemSetting
	err := s.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting = models.SystemSetting{Key: key, Value: value}
			return s.db.Create(&setting).Error
		}
		return err
	}
	return s.db.Model(&setting).Update("value", value).Error
}

func (s *SystemSettingService) GetAll() ([]dto.SystemSettingItem, error) {
	var settings []models.SystemSetting
	if err := s.db.Order("key ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	result := make([]dto.SystemSettingItem, len(settings))
	for i, setting := range settings {
		result[i] = dto.SystemSettingItem{
			Key:   setting.Key,
			Value: setting.Value,
		}
	}
	return result, nil
}

func (s *SystemSettingService) BulkSet(items []dto.SystemSettingItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			var setting models.SystemSetting
			err := tx.Where("key = ?", item.Key).First(&setting).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					setting = models.SystemSetting{Key: item.Key, Value: item.Value}
					if err := tx.Create(&setting).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			if err := tx.Model(&setting).Update("value", item.Value).Error; err != nil {
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
