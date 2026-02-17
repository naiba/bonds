package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupCurrencyTest(t *testing.T) *CurrencyService {
	t.Helper()
	db := testutil.SetupTestDB(t)

	models.SeedCurrencies(db)

	return NewCurrencyService(db)
}

func TestCurrencyList(t *testing.T) {
	svc := setupCurrencyTest(t)

	currencies, err := svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(currencies) == 0 {
		t.Error("Expected at least some currencies after seeding")
	}
}

func TestCurrencyListEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewCurrencyService(db)

	currencies, err := svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	_, err = authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "currency-test@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_ = currencies
}

func setupCurrencyTestWithAccount(t *testing.T) (*CurrencyService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	models.SeedCurrencies(db)

	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test", LastName: "User",
		Email: "currency-toggle-test@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	return NewCurrencyService(db), resp.User.AccountID
}

func TestCurrencyToggle(t *testing.T) {
	svc, accountID := setupCurrencyTestWithAccount(t)

	var count int64
	svc.db.Model(&models.AccountCurrency{}).Where("account_id = ?", accountID).Count(&count)
	initialCount := count

	var firstCurrency models.Currency
	if err := svc.db.First(&firstCurrency).Error; err != nil {
		t.Fatalf("Failed to get first currency: %v", err)
	}

	if err := svc.Toggle(accountID, firstCurrency.ID); err != nil {
		t.Fatalf("Toggle (disable) failed: %v", err)
	}
	svc.db.Model(&models.AccountCurrency{}).Where("account_id = ?", accountID).Count(&count)
	if count != initialCount-1 {
		t.Errorf("Expected %d after toggle off, got %d", initialCount-1, count)
	}

	if err := svc.Toggle(accountID, firstCurrency.ID); err != nil {
		t.Fatalf("Toggle (enable) failed: %v", err)
	}
	svc.db.Model(&models.AccountCurrency{}).Where("account_id = ?", accountID).Count(&count)
	if count != initialCount {
		t.Errorf("Expected %d after toggle on, got %d", initialCount, count)
	}
}

func TestCurrencyToggleNotFound(t *testing.T) {
	svc, accountID := setupCurrencyTestWithAccount(t)

	if err := svc.Toggle(accountID, 999999); err != ErrCurrencyNotFound {
		t.Errorf("Expected ErrCurrencyNotFound, got %v", err)
	}
}

func TestCurrencyDisableAll(t *testing.T) {
	svc, accountID := setupCurrencyTestWithAccount(t)

	if err := svc.DisableAll(accountID); err != nil {
		t.Fatalf("DisableAll failed: %v", err)
	}

	var count int64
	svc.db.Model(&models.AccountCurrency{}).Where("account_id = ?", accountID).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 after DisableAll, got %d", count)
	}
}

func TestCurrencyEnableAll(t *testing.T) {
	svc, accountID := setupCurrencyTestWithAccount(t)

	if err := svc.DisableAll(accountID); err != nil {
		t.Fatalf("DisableAll failed: %v", err)
	}

	if err := svc.EnableAll(accountID); err != nil {
		t.Fatalf("EnableAll failed: %v", err)
	}

	var totalCurrencies int64
	svc.db.Model(&models.Currency{}).Count(&totalCurrencies)

	var linkedCount int64
	svc.db.Model(&models.AccountCurrency{}).Where("account_id = ?", accountID).Count(&linkedCount)
	if linkedCount != totalCurrencies {
		t.Errorf("Expected %d linked after EnableAll, got %d", totalCurrencies, linkedCount)
	}
}
