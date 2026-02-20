package config

import (
	"os"
	"strconv"
	"strings"
)

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

type Config struct {
	Debug        bool
	Server       ServerConfig
	Database     DatabaseConfig
	JWT          JWTConfig
	App          AppConfig
	SMTP         SMTPConfig
	Storage      StorageConfig
	OAuth        OAuthConfig
	WebAuthn     WebAuthnConfig
	Telegram     TelegramConfig
	Geocoding    GeocodingConfig
	Bleve        BleveConfig
	Backup       BackupConfig
	Announcement string
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type JWTConfig struct {
	Secret     string
	ExpiryHrs  int
	RefreshHrs int
}

type AppConfig struct {
	Name string
	Env  string
	URL  string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type StorageConfig struct {
	UploadDir string
	MaxSize   int64
}

type OAuthConfig struct {
	GitHubKey        string
	GitHubSecret     string
	GoogleKey        string
	GoogleSecret     string
	OIDCKey          string
	OIDCSecret       string
	OIDCDiscoveryURL string
	OIDCName         string // Display name, e.g. "Authentik", "Keycloak"
}

type WebAuthnConfig struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

type TelegramConfig struct {
	BotToken string
}

type GeocodingConfig struct {
	Provider string
	APIKey   string
}

type BleveConfig struct {
	IndexPath string
}

type BackupConfig struct {
	Dir       string
	Cron      string
	Retention int
}

func Load() *Config {
	webAuthnOrigins := getEnv("WEBAUTHN_RP_ORIGINS", "")
	var rpOrigins []string
	if webAuthnOrigins != "" {
		for _, origin := range splitAndTrim(webAuthnOrigins, ",") {
			if origin != "" {
				rpOrigins = append(rpOrigins, origin)
			}
		}
	}

	return &Config{
		Debug: getEnvBool("DEBUG", false),
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Driver: getEnv("DB_DRIVER", "sqlite"),
			DSN:    getEnv("DB_DSN", "bonds.db"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			ExpiryHrs:  getEnvInt("JWT_EXPIRY_HRS", 24),
			RefreshHrs: getEnvInt("JWT_REFRESH_HRS", 168),
		},
		App: AppConfig{
			Name: getEnv("APP_NAME", "Bonds"),
			Env:  getEnv("APP_ENV", "development"),
			URL:  getEnv("APP_URL", "http://localhost:8080"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", ""),
		},
		Storage: StorageConfig{
			UploadDir: getEnv("STORAGE_UPLOAD_DIR", "uploads"),
			MaxSize:   getEnvInt64("STORAGE_MAX_SIZE", 10485760),
		},
		OAuth: OAuthConfig{
			GitHubKey:        getEnv("OAUTH_GITHUB_KEY", ""),
			GitHubSecret:     getEnv("OAUTH_GITHUB_SECRET", ""),
			GoogleKey:        getEnv("OAUTH_GOOGLE_KEY", ""),
			GoogleSecret:     getEnv("OAUTH_GOOGLE_SECRET", ""),
			OIDCKey:          getEnv("OIDC_CLIENT_ID", ""),
			OIDCSecret:       getEnv("OIDC_CLIENT_SECRET", ""),
			OIDCDiscoveryURL: getEnv("OIDC_DISCOVERY_URL", ""),
			OIDCName:         getEnv("OIDC_NAME", "SSO"),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		},
		Geocoding: GeocodingConfig{
			Provider: getEnv("GEOCODING_PROVIDER", "nominatim"),
			APIKey:   getEnv("GEOCODING_API_KEY", ""),
		},
		WebAuthn: WebAuthnConfig{
			RPID:          getEnv("WEBAUTHN_RP_ID", ""),
			RPDisplayName: getEnv("WEBAUTHN_RP_DISPLAY_NAME", "Bonds"),
			RPOrigins:     rpOrigins,
		},
		Bleve: BleveConfig{
			IndexPath: getEnv("BLEVE_INDEX_PATH", "data/bonds.bleve"),
		},
		Backup: BackupConfig{
			Dir:       getEnv("BACKUP_DIR", "data/backups"),
			Cron:      getEnv("BACKUP_CRON", ""),
			Retention: getEnvInt("BACKUP_RETENTION", 30),
		},
		Announcement: getEnv("ANNOUNCEMENT", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}
