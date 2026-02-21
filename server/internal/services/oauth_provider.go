package services

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/discord"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrOAuthProviderNotFoundByID = errors.New("oauth provider not found")
	ErrOAuthProviderNameExists   = errors.New("oauth provider name already exists")
)

type OAuthProviderService struct {
	db       *gorm.DB
	settings *SystemSettingService
}

func NewOAuthProviderService(db *gorm.DB) *OAuthProviderService {
	return &OAuthProviderService{db: db}
}

func (s *OAuthProviderService) SetSystemSettings(ss *SystemSettingService) {
	s.settings = ss
}

func (s *OAuthProviderService) List() ([]dto.OAuthProviderResponse, error) {
	var providers []models.OAuthProvider
	if err := s.db.Order("id ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	result := make([]dto.OAuthProviderResponse, len(providers))
	for i, p := range providers {
		result[i] = toOAuthProviderResponse(p)
	}
	return result, nil
}

func (s *OAuthProviderService) Create(req dto.CreateOAuthProviderRequest) (*dto.OAuthProviderResponse, error) {
	var count int64
	s.db.Model(&models.OAuthProvider{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, ErrOAuthProviderNameExists
	}

	provider := models.OAuthProvider{
		Type:         req.Type,
		Name:         req.Name,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		DisplayName:  req.DisplayName,
		DiscoveryURL: req.DiscoveryURL,
		Scopes:       req.Scopes,
	}

	if err := s.db.Create(&provider).Error; err != nil {
		return nil, err
	}

	// Handle Enabled field: default true, but if req.Enabled is explicitly false, do two-step (SQLite bool zero-value trap)
	if req.Enabled != nil && !*req.Enabled {
		s.db.Model(&provider).Update("enabled", false)
		provider.Enabled = false
	} else {
		provider.Enabled = true
	}

	s.ReloadProviders()

	resp := toOAuthProviderResponse(provider)
	return &resp, nil
}

func (s *OAuthProviderService) Update(id uint, req dto.UpdateOAuthProviderRequest) (*dto.OAuthProviderResponse, error) {
	var provider models.OAuthProvider
	if err := s.db.First(&provider, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOAuthProviderNotFoundByID
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.ClientID != nil {
		updates["client_id"] = *req.ClientID
	}
	if req.ClientSecret != nil {
		updates["client_secret"] = *req.ClientSecret
	}
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.DiscoveryURL != nil {
		updates["discovery_url"] = *req.DiscoveryURL
	}
	if req.Scopes != nil {
		updates["scopes"] = *req.Scopes
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) > 0 {
		if err := s.db.Model(&provider).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := s.db.First(&provider, id).Error; err != nil {
		return nil, err
	}

	s.ReloadProviders()

	resp := toOAuthProviderResponse(provider)
	return &resp, nil
}

func (s *OAuthProviderService) Delete(id uint) error {
	var provider models.OAuthProvider
	if err := s.db.First(&provider, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOAuthProviderNotFoundByID
		}
		return err
	}

	if err := s.db.Delete(&provider).Error; err != nil {
		return err
	}

	s.ReloadProviders()
	return nil
}

func (s *OAuthProviderService) ReloadProviders() {
	appURL := "http://localhost:8080"
	if s.settings != nil {
		appURL = s.settings.GetWithDefault("app.url", appURL)
	}

	goth.ClearProviders()

	var providers []models.OAuthProvider
	if err := s.db.Where("enabled = ?", true).Find(&providers).Error; err != nil {
		log.Printf("WARNING: Failed to load OAuth providers from DB: %v", err)
		return
	}

	var gothProviders []goth.Provider
	for _, p := range providers {
		gp, err := createGothProvider(p, appURL)
		if err != nil {
			log.Printf("WARNING: Failed to create goth provider %q (%s): %v", p.Name, p.Type, err)
			continue
		}
		gothProviders = append(gothProviders, gp)
	}

	if len(gothProviders) > 0 {
		goth.UseProviders(gothProviders...)
	}
}

func createGothProvider(p models.OAuthProvider, appURL string) (goth.Provider, error) {
	callback := appURL + "/api/auth/" + p.Name + "/callback"
	switch p.Type {
	case "github":
		return github.New(p.ClientID, p.ClientSecret, callback), nil
	case "google":
		scopes := splitScopes(p.Scopes, "email", "profile")
		return google.New(p.ClientID, p.ClientSecret, callback, scopes...), nil
	case "gitlab":
		scopes := splitScopes(p.Scopes, "read_user", "openid", "email")
		provider := gitlab.New(p.ClientID, p.ClientSecret, callback, scopes...)
		return provider, nil
	case "discord":
		scopes := splitScopes(p.Scopes, "identify", "email")
		return discord.New(p.ClientID, p.ClientSecret, callback, scopes...), nil
	case "oidc":
		if p.DiscoveryURL == "" {
			return nil, errors.New("oidc: discovery_url required")
		}
		oidcProvider, err := openidConnect.New(p.ClientID, p.ClientSecret, callback, p.DiscoveryURL)
		if err != nil {
			return nil, err
		}
		oidcProvider.SetName(p.Name)
		return oidcProvider, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", p.Type)
	}
}

func splitScopes(csv string, defaults ...string) []string {
	if csv == "" {
		return defaults
	}
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return defaults
	}
	return result
}

func toOAuthProviderResponse(p models.OAuthProvider) dto.OAuthProviderResponse {
	return dto.OAuthProviderResponse{
		ID:           p.ID,
		Type:         p.Type,
		Name:         p.Name,
		ClientID:     p.ClientID,
		HasSecret:    p.ClientSecret != "",
		Enabled:      p.Enabled,
		DisplayName:  p.DisplayName,
		DiscoveryURL: p.DiscoveryURL,
		Scopes:       p.Scopes,
	}
}
