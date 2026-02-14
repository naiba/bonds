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
	appURL       string
}

func NewOAuthHandler(oauthService *services.OAuthService, appURL string, jwtSecret string) *OAuthHandler {
	// Set up Gothic session store
	store := sessions.NewCookieStore([]byte(jwtSecret))
	store.MaxAge(86400 * 30)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	gothic.Store = store

	return &OAuthHandler{oauthService: oauthService, appURL: appURL}
}

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

func (h *OAuthHandler) Callback(c echo.Context) error {
	provider := c.Param("provider")

	// Set provider for gothic
	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.appURL))
	}

	authResp, err := h.oauthService.FindOrCreateUser(provider, gothUser.UserID, gothUser.Email, gothUser.Name)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.appURL))
	}

	// Save OAuth token for future API calls
	expiresIn := 0
	if !gothUser.ExpiresAt.IsZero() {
		expiresIn = int(gothUser.ExpiresAt.Unix())
	}
	_ = h.oauthService.SaveToken(authResp.User.ID, provider, gothUser.UserID,
		gothUser.AccessToken, gothUser.RefreshToken, expiresIn)

	return c.Redirect(http.StatusTemporaryRedirect,
		fmt.Sprintf("%s/auth/callback?token=%s", h.appURL, authResp.Token))
}
