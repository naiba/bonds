package services

import (
	"log"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/naiba/bonds/internal/config"
)

func SetupOAuthProviders(cfg *config.OAuthConfig, appURL string) {
	var providers []goth.Provider
	if cfg.GitHubKey != "" && cfg.GitHubSecret != "" {
		providers = append(providers, github.New(cfg.GitHubKey, cfg.GitHubSecret, appURL+"/api/auth/github/callback"))
	}
	if cfg.GoogleKey != "" && cfg.GoogleSecret != "" {
		providers = append(providers, google.New(cfg.GoogleKey, cfg.GoogleSecret, appURL+"/api/auth/google/callback", "email", "profile"))
	}
	if cfg.OIDCKey != "" && cfg.OIDCSecret != "" && cfg.OIDCDiscoveryURL != "" {
		oidcProvider, err := openidConnect.New(cfg.OIDCKey, cfg.OIDCSecret, appURL+"/api/auth/openid-connect/callback", cfg.OIDCDiscoveryURL)
		if err == nil {
			oidcProvider.SetName("openid-connect")
			providers = append(providers, oidcProvider)
		} else {
			log.Printf("WARNING: Failed to initialize OIDC provider: %v", err)
		}
	}
	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}

func SetupOAuthProvidersFromDB(settings *SystemSettingService) {
	appURL := settings.GetWithDefault("app.url", "http://localhost:8080")

	ghKey := settings.GetWithDefault("oauth_github_key", "")
	ghSecret := settings.GetWithDefault("oauth_github_secret", "")
	goKey := settings.GetWithDefault("oauth_google_key", "")
	goSecret := settings.GetWithDefault("oauth_google_secret", "")
	oidcKey := settings.GetWithDefault("oidc_client_id", "")
	oidcSecret := settings.GetWithDefault("oidc_client_secret", "")
	oidcDiscovery := settings.GetWithDefault("oidc_discovery_url", "")

	goth.ClearProviders()

	var providers []goth.Provider
	if ghKey != "" && ghSecret != "" {
		providers = append(providers, github.New(ghKey, ghSecret, appURL+"/api/auth/github/callback"))
	}
	if goKey != "" && goSecret != "" {
		providers = append(providers, google.New(goKey, goSecret, appURL+"/api/auth/google/callback", "email", "profile"))
	}
	if oidcKey != "" && oidcSecret != "" && oidcDiscovery != "" {
		oidcProvider, err := openidConnect.New(oidcKey, oidcSecret, appURL+"/api/auth/openid-connect/callback", oidcDiscovery)
		if err == nil {
			oidcProvider.SetName("openid-connect")
			providers = append(providers, oidcProvider)
		} else {
			log.Printf("WARNING: Failed to initialize OIDC provider from DB: %v", err)
		}
	}
	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}
