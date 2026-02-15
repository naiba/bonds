package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupOAuthExtraTest(t *testing.T) (*OAuthService, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "oauth-extra-test@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	svc := NewOAuthService(db, cfg, "http://localhost:8080")

	tokens := []models.UserToken{
		{UserID: resp.User.ID, Driver: "github", DriverID: "gh-123", Token: "token1"},
		{UserID: resp.User.ID, Driver: "google", DriverID: "goo-456", Token: "token2"},
	}
	for _, tok := range tokens {
		if err := db.Create(&tok).Error; err != nil {
			t.Fatalf("Create token failed: %v", err)
		}
	}

	return svc, resp.User.ID
}

func TestListProviders(t *testing.T) {
	svc, userID := setupOAuthExtraTest(t)

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	driverSet := map[string]bool{}
	for _, p := range providers {
		driverSet[p["driver"]] = true
		if p["driver_id"] == "" {
			t.Error("Expected non-empty driver_id")
		}
	}
	if !driverSet["github"] {
		t.Error("Expected 'github' in providers")
	}
	if !driverSet["google"] {
		t.Error("Expected 'google' in providers")
	}
}

func TestListProvidersEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "No",
		LastName:  "OAuth",
		Email:     "no-oauth@example.com",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	svc := NewOAuthService(db, cfg, "http://localhost:8080")
	providers, err := svc.ListProviders(resp.User.ID)
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestUnlinkProvider(t *testing.T) {
	svc, userID := setupOAuthExtraTest(t)

	err := svc.UnlinkProvider(userID, "github")
	if err != nil {
		t.Fatalf("UnlinkProvider failed: %v", err)
	}

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider after unlink, got %d", len(providers))
	}
	if providers[0]["driver"] != "google" {
		t.Errorf("Expected remaining provider to be 'google', got '%s'", providers[0]["driver"])
	}
}

func TestUnlinkProviderNotFound(t *testing.T) {
	svc, userID := setupOAuthExtraTest(t)

	err := svc.UnlinkProvider(userID, "twitter")
	if err != ErrOAuthTokenNotFound {
		t.Errorf("Expected ErrOAuthTokenNotFound, got %v", err)
	}
}

func TestUnlinkProviderWrongUser(t *testing.T) {
	svc, _ := setupOAuthExtraTest(t)

	err := svc.UnlinkProvider("non-existent-user-id", "github")
	if err != ErrOAuthTokenNotFound {
		t.Errorf("Expected ErrOAuthTokenNotFound, got %v", err)
	}
}
