package services

import (
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type StorageInfoService struct {
	db             *gorm.DB
	settingService *SystemSettingService
}

func NewStorageInfoService(db *gorm.DB, settingService *SystemSettingService) *StorageInfoService {
	return &StorageInfoService{db: db, settingService: settingService}
}

func (s *StorageInfoService) Get(accountID string) (*dto.StorageResponse, error) {
	var vaultIDs []string
	if err := s.db.Model(&models.Vault{}).Where("account_id = ?", accountID).Pluck("id", &vaultIDs).Error; err != nil {
		return nil, err
	}

	var totalSize int64
	if len(vaultIDs) > 0 {
		var sum *int64
		if err := s.db.Model(&models.File{}).Where("vault_id IN ?", vaultIDs).Select("COALESCE(SUM(size), 0)").Scan(&sum).Error; err != nil {
			return nil, err
		}
		if sum != nil {
			totalSize = *sum
		}
	}

	var account models.Account
	if err := s.db.First(&account, "id = ?", accountID).Error; err != nil {
		return nil, err
	}

	// 用户单独限制优先；为 0 时回退到实例级默认限制（storage.default_limit_mb）。
	limitMB := int64(account.StorageLimitInMB)
	if limitMB == 0 && s.settingService != nil {
		limitMB = s.settingService.GetInt64("storage.default_limit_mb", 0)
	}
	limitBytes := limitMB * 1024 * 1024

	return &dto.StorageResponse{
		UsedBytes:  totalSize,
		LimitBytes: limitBytes,
	}, nil
}
