package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func TestGetCalendarMonth(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "cal-month-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	calSvc := NewCalendarService(db)
	result, err := calSvc.GetCalendarMonth(vault.ID, 2025, 6)
	if err != nil {
		t.Fatalf("GetCalendarMonth failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetCalendarDay(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, _ := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "cal-day-test@example.com", Password: "password123",
	})
	vault, _ := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"})

	calSvc := NewCalendarService(db)
	result, err := calSvc.GetCalendarDay(vault.ID, 2025, 6, 15)
	if err != nil {
		t.Fatalf("GetCalendarDay failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}
