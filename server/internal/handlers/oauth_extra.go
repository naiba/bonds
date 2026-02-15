package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
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
