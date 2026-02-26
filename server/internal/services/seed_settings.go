package services

import (
	"fmt"
	"log"
	"strconv"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type settingSeed struct {
	Key   string
	Value string
}

// SeedSettingsFromEnv seeds system settings from the config (env vars) into the database.
// Only sets values that don't already exist in the DB (env vars are initial defaults).
func SeedSettingsFromEnv(svc *SystemSettingService, cfg *config.Config) error {
	seeds := []settingSeed{
		{"app.name", cfg.App.Name},
		{"app.url", cfg.App.URL},

		{"smtp.host", cfg.SMTP.Host},
		{"smtp.port", cfg.SMTP.Port},
		{"smtp.username", cfg.SMTP.Username},
		{"smtp.password", cfg.SMTP.Password},
		{"smtp.from", cfg.SMTP.From},

		{"webauthn.rp_id", cfg.WebAuthn.RPID},
		{"webauthn.rp_display_name", cfg.WebAuthn.RPDisplayName},
		{"webauthn.rp_origins", joinStrings(cfg.WebAuthn.RPOrigins)},

		{"geocoding.provider", cfg.Geocoding.Provider},
		{"geocoding.api_key", cfg.Geocoding.APIKey},

		{"backup.cron", cfg.Backup.Cron},
		{"backup.retention", strconv.Itoa(cfg.Backup.Retention)},

		{"jwt.expiry_hrs", strconv.Itoa(cfg.JWT.ExpiryHrs)},
		{"jwt.refresh_hrs", strconv.Itoa(cfg.JWT.RefreshHrs)},

		{"storage.max_size", strconv.FormatInt(cfg.Storage.MaxSize, 10)},
		{"storage.default_limit_mb", "0"},
		{"auth.require_email_verification", "false"},
		{"registration.enabled", "true"},
		{"auth.password.enabled", "true"},
		{"announcement", cfg.Announcement},
	}

	seeded := 0
	for _, s := range seeds {
		_, err := svc.Get(s.Key)
		if err == nil {
			continue
		}
		if err != ErrSystemSettingNotFound {
			return fmt.Errorf("check setting %q: %w", s.Key, err)
		}
		if err := svc.Set(s.Key, s.Value); err != nil {
			return fmt.Errorf("seed setting %q: %w", s.Key, err)
		}
		seeded++
	}

	if seeded > 0 {
		log.Printf("Seeded %d system settings from environment variables", seeded)
	}

	return nil
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += "," + s
	}
	return result
}

// MigrateOAuthSettingsToProviders migrates legacy oauth_* system settings
// to the OAuthProvider table. Runs once: skips if any providers already exist.
// After migration, the legacy settings are deleted from system_settings.
func MigrateOAuthSettingsToProviders(db *gorm.DB, settings *SystemSettingService) {
	var count int64
	db.Model(&models.OAuthProvider{}).Count(&count)
	if count > 0 {
		return
	}

	legacyKeys := []string{
		"oauth_github_key", "oauth_github_secret",
		"oauth_google_key", "oauth_google_secret",
		"oidc_client_id", "oidc_client_secret", "oidc_discovery_url", "oidc_name",
	}

	ghKey := settings.GetWithDefault("oauth_github_key", "")
	ghSecret := settings.GetWithDefault("oauth_github_secret", "")
	if ghKey != "" && ghSecret != "" {
		db.Create(&models.OAuthProvider{
			Type: "github", Name: "github", ClientID: ghKey, ClientSecret: ghSecret,
			DisplayName: "GitHub", Enabled: true,
		})
	}

	goKey := settings.GetWithDefault("oauth_google_key", "")
	goSecret := settings.GetWithDefault("oauth_google_secret", "")
	if goKey != "" && goSecret != "" {
		db.Create(&models.OAuthProvider{
			Type: "google", Name: "google", ClientID: goKey, ClientSecret: goSecret,
			DisplayName: "Google", Enabled: true,
		})
	}

	oidcKey := settings.GetWithDefault("oidc_client_id", "")
	oidcSecret := settings.GetWithDefault("oidc_client_secret", "")
	oidcDiscovery := settings.GetWithDefault("oidc_discovery_url", "")
	oidcName := settings.GetWithDefault("oidc_name", "SSO")
	if oidcKey != "" && oidcSecret != "" && oidcDiscovery != "" {
		db.Create(&models.OAuthProvider{
			Type: "oidc", Name: "openid-connect", ClientID: oidcKey, ClientSecret: oidcSecret,
			DisplayName: oidcName, DiscoveryURL: oidcDiscovery, Enabled: true,
		})
	}

	migrated := false
	for _, key := range legacyKeys {
		if err := settings.Delete(key); err == nil {
			migrated = true
		}
	}
	if migrated {
		log.Println("Migrated legacy OAuth settings to OAuthProvider table")
	}
}
