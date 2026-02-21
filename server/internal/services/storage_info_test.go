package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func setupStorageInfoTest(t *testing.T) (*StorageInfoService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "storage-test@example.com", Password: "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewStorageInfoService(db), resp.User.AccountID
}

func TestStorageInfoGet(t *testing.T) {
	svc, accountID := setupStorageInfoTest(t)

	info, err := svc.Get(accountID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info.UsedBytes != 0 {
		t.Errorf("Expected 0 used bytes, got %d", info.UsedBytes)
	}
}
