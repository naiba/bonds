package handlers

import (
	"errors"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type GeocodingAdminHandler struct {
	manager *services.GeocodingManager
}

func NewGeocodingAdminHandler(manager *services.GeocodingManager) *GeocodingAdminHandler {
	return &GeocodingAdminHandler{manager: manager}
}

// Get godoc
//
//	@Summary		Get geocoding configuration
//	@Description	Get the active geocoding provider and the structured configuration metadata for every supported provider (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.GeocodingAdminResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/geocoding [get]
func (h *GeocodingAdminHandler) Get(c *echo.Context) error {
	result, err := h.manager.Get()
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_geocoding_settings")
	}
	return response.OK(c, result)
}

// UpdateSettings godoc
//
//	@Summary		Update active geocoding settings
//	@Description	Select the active geocoding provider and address precision (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateGeocodingSettingsRequest	true	"Active geocoding settings"
//	@Success		200		{object}	response.APIResponse{data=dto.GeocodingAdminResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/geocoding [put]
func (h *GeocodingAdminHandler) UpdateSettings(c *echo.Context) error {
	var req dto.UpdateGeocodingSettingsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.BadRequest(c, "err.invalid_geocoding_settings", map[string]string{"validation": err.Error()})
	}
	result, err := h.manager.UpdateSettings(req.ActiveProvider, req.Precision)
	if err != nil {
		return geocodingAdminError(c, err, "err.failed_to_update_geocoding_settings")
	}
	return response.OK(c, result)
}

// UpdateProvider godoc
//
//	@Summary		Update a geocoding provider configuration
//	@Description	Create or replace the one structured configuration for a geocoding provider (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider	path		string									true	"Provider ID"
//	@Param			request		body		dto.UpdateGeocodingProviderConfigRequest	true	"Provider configuration"
//	@Success		200			{object}	response.APIResponse{data=dto.GeocodingAdminResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		403			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/admin/geocoding/providers/{provider} [put]
func (h *GeocodingAdminHandler) UpdateProvider(c *echo.Context) error {
	var req dto.UpdateGeocodingProviderConfigRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.Config == nil {
		return response.BadRequest(c, "err.invalid_geocoding_settings", map[string]string{"config": "config is required"})
	}
	result, err := h.manager.SaveProvider(c.Param("provider"), req.Config)
	if err != nil {
		return geocodingAdminError(c, err, "err.failed_to_update_geocoding_provider")
	}
	return response.OK(c, result)
}

// DeleteProvider godoc
//
//	@Summary		Delete a geocoding provider configuration
//	@Description	Delete the one stored configuration for a provider; code-owned defaults remain available (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider	path		string	true	"Provider ID"
//	@Success		200			{object}	response.APIResponse{data=dto.GeocodingAdminResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		403			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/admin/geocoding/providers/{provider} [delete]
func (h *GeocodingAdminHandler) DeleteProvider(c *echo.Context) error {
	result, err := h.manager.DeleteProvider(c.Param("provider"))
	if err != nil {
		return geocodingAdminError(c, err, "err.failed_to_delete_geocoding_provider")
	}
	return response.OK(c, result)
}

func geocodingAdminError(c *echo.Context, err error, fallback string) error {
	if errors.Is(err, services.ErrUnknownGeocodingProvider) ||
		errors.Is(err, services.ErrInvalidGeocodingConfig) ||
		errors.Is(err, services.ErrGeocodingNotConfigured) {
		return response.BadRequest(c, "err.invalid_geocoding_settings", map[string]string{"validation": err.Error()})
	}
	return response.InternalError(c, fallback)
}
