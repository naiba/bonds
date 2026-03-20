package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type OAuthHandler struct {
	oauthService *services.OAuthService
	settings     *services.SystemSettingService
	jwtSecret    []byte
}

func NewOAuthHandler(oauthService *services.OAuthService, settings *services.SystemSettingService, jwtSecret string) *OAuthHandler {
	store := sessions.NewCookieStore([]byte(jwtSecret))
	store.MaxAge(86400 * 30)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	gothic.Store = store

	return &OAuthHandler{oauthService: oauthService, settings: settings, jwtSecret: []byte(jwtSecret)}
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

	if _, err := goth.GetProvider(provider); err != nil {
		return response.NotFound(c, "err.oauth_provider_not_found")
	}

	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	// ?mode=link&token=xxx&state=yyy: settings page initiates OAuth binding.
	// Validate the CSRF state token before storing JWT in session, preventing
	// attackers from tricking users into binding a malicious OAuth account.
	if c.QueryParam("mode") == "link" {
		jwtToken := c.QueryParam("token")
		state := c.QueryParam("state")
		if jwtToken != "" && state != "" {
			claims, err := middleware.ParseJWTClaims(jwtToken, h.jwtSecret)
			if err != nil {
				return response.Unauthorized(c, "err.invalid_or_expired_token")
			}
			session, _ := gothic.Store.Get(c.Request(), "oauth_link")
			session.Values["link_jwt"] = jwtToken
			session.Values["link_user_id"] = claims.UserID
			session.Values["link_state"] = state
			session.Save(c.Request(), c.Response())
		}
	}

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

	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.getAppURL()))
	}

	// Check if this callback is from a settings-page "link" flow
	// by reading the JWT stored in session during BeginAuth.
	if userID := h.extractLinkUserID(c); userID != "" {
		return h.handleLinkCallback(c, provider, gothUser, userID)
	}

	locale := middleware.GetLocale(c)
	authResp, linkInfo, err := h.oauthService.FindOrCreateUser(provider, gothUser.UserID, gothUser.Email, gothUser.Name, locale)
	if err != nil {
		if errors.Is(err, services.ErrOAuthAccountNotLinked) {
			linkToken, tokenErr := h.oauthService.GenerateLinkToken(linkInfo)
			if tokenErr != nil {
				return c.Redirect(http.StatusTemporaryRedirect,
					fmt.Sprintf("%s/login?error=oauth_failed", h.getAppURL()))
			}
			return c.Redirect(http.StatusTemporaryRedirect,
				fmt.Sprintf("%s/auth/oauth-link?link_token=%s", h.getAppURL(), linkToken))
		}
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/login?error=oauth_failed", h.getAppURL()))
	}

	expiresIn := 0
	if !gothUser.ExpiresAt.IsZero() {
		expiresIn = int(gothUser.ExpiresAt.Unix())
	}
	_ = h.oauthService.SaveToken(authResp.User.ID, provider, gothUser.UserID,
		gothUser.AccessToken, gothUser.RefreshToken, expiresIn)

	return c.Redirect(http.StatusTemporaryRedirect,
		fmt.Sprintf("%s/auth/callback?token=%s", h.getAppURL(), authResp.Token))
}

func (h *OAuthHandler) extractLinkUserID(c echo.Context) string {
	session, err := gothic.Store.Get(c.Request(), "oauth_link")
	if err != nil {
		return ""
	}
	userID, ok := session.Values["link_user_id"].(string)
	if !ok || userID == "" {
		return ""
	}
	storedState, _ := session.Values["link_state"].(string)

	session.Values["link_jwt"] = ""
	session.Values["link_user_id"] = ""
	session.Values["link_state"] = ""
	session.Options.MaxAge = -1
	session.Save(c.Request(), c.Response())

	// CSRF protection: the state was generated client-side and stored in session
	// during BeginAuth. If the session has no state, this is not a valid link flow.
	if storedState == "" {
		return ""
	}
	return userID
}

func (h *OAuthHandler) handleLinkCallback(c echo.Context, provider string, gothUser goth.User, userID string) error {
	linkInfo := &services.OAuthLinkInfo{
		Provider:       provider,
		ProviderUserID: gothUser.UserID,
		Email:          gothUser.Email,
		Name:           gothUser.Name,
	}
	linkToken, err := h.oauthService.GenerateLinkToken(linkInfo)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/settings/oauth?error=oauth_failed", h.getAppURL()))
	}

	_, err = h.oauthService.LinkOAuthToUser(linkToken, userID)
	if err != nil {
		errMsg := "oauth_failed"
		if errors.Is(err, services.ErrOAuthAlreadyLinked) {
			errMsg = "oauth_already_linked"
		}
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/settings/oauth?error=%s", h.getAppURL(), errMsg))
	}

	return c.Redirect(http.StatusTemporaryRedirect,
		fmt.Sprintf("%s/settings/oauth?linked=%s", h.getAppURL(), provider))
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
