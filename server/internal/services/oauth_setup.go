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
