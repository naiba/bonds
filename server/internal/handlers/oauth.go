package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type OAuthHandler struct {
	oauthService *services.OAuthService
	settings     *services.SystemSettingService
}

func NewOAuthHandler(oauthService *services.OAuthService, settings *services.SystemSettingService, jwtSecret string) *OAuthHandler {
	store := sessions.NewCookieStore([]byte(jwtSecret))
	store.MaxAge(86400 * 30)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	gothic.Store = store

	return &OAuthHandler{oauthService: oauthService, settings: settings}
}

func (h *OAuthHandler) getAppURL() string {
	return h.settings.GetWithDefault("app.url", "http://localhost:8080")
}

// BeginAuth godoc
//
//	@Summary		Begin OAuth authentication
//	@Description	Redirect to OAuth provider for authentication
//	@Tags			oauth
//	@Param			provider	path	string	true	"OAuth provider name"
//	@Success		307			"Redirect to OAuth provider"
//	@Failure		404			{object}	response.APIResponse
//	@Router			/auth/{provider} [get]
func (h *OAuthHandler) BeginAuth(c echo.Context) error {
	provider := c.Param("provider")

	// Verify the provider is registered with goth
	if _, err := goth.GetProvider(provider); err != nil {
		return response.NotFound(c, "err.oauth_provider_not_found")
	}

	// Set provider for gothic
	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Response(), c.Request())
	return nil
}

// Callback godoc
//
//	@Summary		OAuth callback
//	@Description	Handle OAuth callback and redirect with JWT token
//	@Tags			oauth
//	@Param			provider	path	string	true	"OAuth provider name"
//	@Success		307			"Redirect to frontend with token"
//	@Router			/auth/{provider}/callback [get]
func (h *OAuthHandler) Callback(c echo.Context) error {
	provider := c.Param("provider")

	// Set provider for gothic
	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.getAppURL()))
	}

	authResp, err := h.oauthService.FindOrCreateUser(provider, gothUser.UserID, gothUser.Email, gothUser.Name)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.getAppURL()))
	}

	// Save OAuth token for future API calls
	expiresIn := 0
	if !gothUser.ExpiresAt.IsZero() {
		expiresIn = int(gothUser.ExpiresAt.Unix())
	}
	_ = h.oauthService.SaveToken(authResp.User.ID, provider, gothUser.UserID,
		gothUser.AccessToken, gothUser.RefreshToken, expiresIn)

	return c.Redirect(http.StatusTemporaryRedirect,
		fmt.Sprintf("%s/auth/callback?token=%s", h.getAppURL(), authResp.Token))
}

// AvailableProviders godoc
//
//	@Summary		List available OAuth providers
//	@Description	Return all configured OAuth providers (public, no auth required)
//	@Tags			oauth
//	@Produce		json
//	@Success		200	{object}	response.APIResponse
//	@Router			/auth/providers [get]
func (h *OAuthHandler) AvailableProviders(c echo.Context) error {
	providers := h.oauthService.ListAvailableProviders()
	return response.OK(c, providers)
}
