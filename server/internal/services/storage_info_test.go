package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupStorageInfoTest(t *testing.T) (*StorageInfoService, *SystemSettingService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	settingSvc := NewSystemSettingService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "storage-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewStorageInfoService(db, settingSvc), settingSvc, resp.User.AccountID
}

func TestStorageInfoGet(t *testing.T) {
	svc, _, accountID := setupStorageInfoTest(t)

	info, err := svc.Get(accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info.UsedBytes != 0 {
		t.Errorf("Expected 0 used bytes, got %d", info.UsedBytes)
	}
}

func TestStorageInfoGet_InstanceDefaultLimit(t *testing.T) {
	svc, settingSvc, accountID := setupStorageInfoTest(t)

	// 设置实例级默认限制为 500MB
	if err := settingSvc.Set("storage.default_limit_mb", "500"); err != nil {
		t.Fatalf("Set default limit failed: %v", err)
	}

	info, err := svc.Get(accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	expectedLimit := int64(500) * 1024 * 1024
	if info.LimitBytes != expectedLimit {
		t.Errorf("Expected limit %d, got %d", expectedLimit, info.LimitBytes)
	}
}

func TestStorageInfoGet_UserLimitOverridesInstanceDefault(t *testing.T) {
	svc, settingSvc, accountID := setupStorageInfoTest(t)

	// 设置实例级默认限制为 500MB
	if err := settingSvc.Set("storage.default_limit_mb", "500"); err != nil {
		t.Fatalf("Set default limit failed: %v", err)
	}

	// 设置用户单独限制为 1024MB，应覆盖实例级限制
	if err := svc.db.Model(&models.Account{}).Where("id = ?", accountID).Update("storage_limit_in_mb", 1024).Error; err != nil {
		t.Fatalf("Update account limit failed: %v", err)
	}

	info, err := svc.Get(accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	expectedLimit := int64(1024) * 1024 * 1024
	if info.LimitBytes != expectedLimit {
		t.Errorf("Expected limit %d (user override), got %d", expectedLimit, info.LimitBytes)
	}
}
