package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

// ListProviders godoc
//
//	@Summary		List linked OAuth providers
//	@Description	Return all linked OAuth providers for the current user
//	@Tags			oauth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/oauth [get]
func (h *OAuthHandler) ListProviders(c echo.Context) error {
	userID := middleware.GetUserID(c)
	providers, err := h.oauthService.ListProviders(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_oauth_providers")
	}
	return response.OK(c, providers)
}

// UnlinkProvider godoc
//
//	@Summary		Unlink an OAuth provider
//	@Description	Unlink an OAuth provider from the current user
//	@Tags			oauth
//	@Security		BearerAuth
//	@Param			driver	path	string	true	"OAuth driver name"
//	@Success		204		"No Content"
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/oauth/{driver} [delete]
func (h *OAuthHandler) UnlinkProvider(c echo.Context) error {
	userID := middleware.GetUserID(c)
	driver := c.Param("driver")
	if err := h.oauthService.UnlinkProvider(userID, driver); err != nil {
		if errors.Is(err, services.ErrOAuthTokenNotFound) {
			return response.NotFound(c, "err.oauth_provider_not_found")
		}
		return response.InternalError(c, "err.failed_to_unlink_oauth_provider")
	}
	return response.NoContent(c)
}

// LinkProvider godoc
//
//	@Summary		Link OAuth provider to current account
//	@Description	Use a link_token from OAuth callback to bind the provider to the logged-in user
//	@Tags			oauth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.OAuthLinkRequest	true	"Link token"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		409		{object}	response.APIResponse
//	@Router			/auth/oauth/link [post]
func (h *OAuthHandler) LinkProvider(c echo.Context) error {
	var req dto.OAuthLinkRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	userID := middleware.GetUserID(c)
	authResp, err := h.oauthService.LinkOAuthToUser(req.LinkToken, userID)
	if err != nil {
		if errors.Is(err, services.ErrOAuthLinkTokenInvalid) {
			return response.BadRequest(c, "err.oauth_link_token_expired", nil)
		}
		if errors.Is(err, services.ErrOAuthAlreadyLinked) {
			return response.Conflict(c, "err.oauth_already_linked")
		}
		return response.InternalError(c, "err.failed_to_link_oauth")
	}

	return response.OK(c, authResp)
}

// LinkRegister godoc
//
//	@Summary		Register new account and link OAuth provider
//	@Description	Use a link_token from OAuth callback to register a new account and bind the provider
//	@Tags			oauth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.OAuthLinkRegisterRequest	true	"Registration info with link token"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		409		{object}	response.APIResponse
//	@Router			/auth/oauth/link-register [post]
func (h *OAuthHandler) LinkRegister(c echo.Context) error {
	var req dto.OAuthLinkRegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	locale := middleware.GetLocale(c)
	authResp, err := h.oauthService.LinkOAuthAndRegister(req.LinkToken, req, locale)
	if err != nil {
		if errors.Is(err, services.ErrOAuthLinkTokenInvalid) {
			return response.BadRequest(c, "err.oauth_link_token_expired", nil)
		}
		if errors.Is(err, services.ErrOAuthAlreadyLinked) {
			return response.Conflict(c, "err.oauth_already_linked")
		}
		if errors.Is(err, services.ErrEmailExists) {
			return response.Conflict(c, "err.email_exists")
		}
		if errors.Is(err, services.ErrRegistrationDisabled) {
			return response.BadRequest(c, "err.registration_disabled", nil)
		}
		return response.InternalError(c, "err.failed_to_link_oauth")
	}

	return response.Created(c, authResp)
}

// BeginLinkProvider godoc
//
//	@Summary		Begin OAuth linking for logged-in user
//	@Description	Redirect to OAuth provider to begin linking process (for settings page)
//	@Tags			oauth
//	@Security		BearerAuth
//	@Param			provider	path	string	true	"OAuth provider name"
//	@Success		307			"Redirect to OAuth provider"
//	@Failure		404			{object}	response.APIResponse
//	@Router			/settings/oauth/link/{provider} [get]
func (h *OAuthHandler) BeginLinkProvider(c echo.Context) error {
	provider := c.Param("provider")

	if _, err := goth.GetProvider(provider); err != nil {
		return response.NotFound(c, "err.oauth_provider_not_found")
	}

	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Response(), c.Request())
	return nil
}

// LinkCallback godoc
//
//	@Summary		OAuth callback for account linking
//	@Description	Handle OAuth callback when user initiated linking from settings page
//	@Tags			oauth
//	@Security		BearerAuth
//	@Param			provider	path	string	true	"OAuth provider name"
//	@Success		307			"Redirect to settings with result"
//	@Router			/settings/oauth/link/{provider}/callback [get]
func (h *OAuthHandler) LinkCallback(c echo.Context) error {
	provider := c.Param("provider")

	q := c.Request().URL.Query()
	q.Set("provider", provider)
	c.Request().URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/settings/oauth?error=oauth_failed", h.getAppURL()))
	}

	userID := middleware.GetUserID(c)
	email := gothUser.Email
	newToken := services.OAuthLinkInfo{
		Provider:       provider,
		ProviderUserID: gothUser.UserID,
		Email:          email,
		Name:           gothUser.Name,
	}
	linkToken, err := h.oauthService.GenerateLinkToken(&newToken)
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
