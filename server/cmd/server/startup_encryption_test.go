package main

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
)

func TestStartupSystemSettingServiceUsesSettingsEncKeyForSeedSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := startupEncryptionTestConfig()

	seedWriter := services.NewSystemSettingServiceWithCipher(db, cfg.Security.SettingsEncKey)
	if err := seedWriter.Set("smtp.password", "stored-secret"); err != nil {
		t.Fatalf("seed encrypted smtp.password: %v", err)
	}

	startupSettings := startupSystemSettingService(db, cfg)
	if err := services.SeedSettingsFromEnv(startupSettings, cfg); err != nil {
		t.Fatalf("startup settings must read existing encrypted rows with SETTINGS_ENC_KEY: %v", err)
	}
}

func TestStartupOAuthProviderServiceUsesSettingsEncKeyForReloadProviders(t *testing.T) {
	defer goth.ClearProviders()
	db := testutil.SetupTestDB(t)
	cfg := startupEncryptionTestConfig()

	settings := services.NewSystemSettingServiceWithCipher(db, cfg.Security.SettingsEncKey)
	if err := settings.Set("app.url", cfg.App.URL); err != nil {
		t.Fatalf("seed app.url: %v", err)
	}
	providerWriter := services.NewOAuthProviderServiceWithCipher(db, cfg.Security.SettingsEncKey)
	providerWriter.SetSystemSettings(settings)
	if _, err := providerWriter.Create(dto.CreateOAuthProviderRequest{
		Type:         "github",
		Name:         "github",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}); err != nil {
		t.Fatalf("seed encrypted OAuth provider: %v", err)
	}
	goth.ClearProviders()

	startupSettings := startupSystemSettingService(db, cfg)
	startupOAuth := startupOAuthProviderService(db, cfg, startupSettings)
	startupOAuth.ReloadProviders()

	if _, err := goth.GetProvider("github"); err != nil {
		t.Fatalf("startup OAuth reload must decrypt provider secret with SETTINGS_ENC_KEY: %v", err)
	}
}

func startupEncryptionTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name: "Bonds",
			URL:  "http://localhost:8080",
		},
		SMTP: config.SMTPConfig{
			Port: "587",
		},
		Storage: config.StorageConfig{
			MaxSizeMB: 10,
		},
		Backup: config.BackupConfig{
			Retention: 30,
		},
		Security: config.SecurityConfig{
			SettingsEncKey: "boot-key",
		},
	}
}
