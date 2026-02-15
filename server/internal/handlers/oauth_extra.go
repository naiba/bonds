package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

func (h *OAuthHandler) ListProviders(c echo.Context) error {
	userID := middleware.GetUserID(c)
	providers, err := h.oauthService.ListProviders(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_oauth_providers")
	}
	return response.OK(c, providers)
}

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
