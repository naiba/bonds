package dto

type CreateOAuthProviderRequest struct {
	Type         string `json:"type" validate:"required,oneof=github google gitlab discord oidc" example:"github"`
	Name         string `json:"name" validate:"required" example:"github"`
	ClientID     string `json:"client_id" validate:"required" example:"abc123"`
	ClientSecret string `json:"client_secret" validate:"required" example:"secret"`
	DisplayName  string `json:"display_name" example:"GitHub"`
	DiscoveryURL string `json:"discovery_url" example:"https://accounts.google.com/.well-known/openid-configuration"`
	Scopes       string `json:"scopes" example:"email,profile"`
	Enabled      *bool  `json:"enabled" example:"true"`
}

type UpdateOAuthProviderRequest struct {
	ClientID     *string `json:"client_id" example:"abc123"`
	ClientSecret *string `json:"client_secret" example:"secret"`
	DisplayName  *string `json:"display_name" example:"GitHub"`
	DiscoveryURL *string `json:"discovery_url" example:""`
	Scopes       *string `json:"scopes" example:"email,profile"`
	Enabled      *bool   `json:"enabled" example:"true"`
}

type OAuthProviderResponse struct {
	ID           uint   `json:"id" example:"1"`
	Type         string `json:"type" example:"github"`
	Name         string `json:"name" example:"github"`
	ClientID     string `json:"client_id" example:"abc123"`
	HasSecret    bool   `json:"has_secret" example:"true"`
	Enabled      bool   `json:"enabled" example:"true"`
	DisplayName  string `json:"display_name" example:"GitHub"`
	DiscoveryURL string `json:"discovery_url" example:""`
	Scopes       string `json:"scopes" example:""`
}
