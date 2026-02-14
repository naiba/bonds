package services

import (
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
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
	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}
