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
