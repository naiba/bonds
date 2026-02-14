package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PreferenceHandler struct {
	preferenceService *services.PreferenceService
}

func NewPreferenceHandler(preferenceService *services.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{preferenceService: preferenceService}
}

func (h *PreferenceHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	prefs, err := h.preferenceService.Get(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_preferences")
	}
	return response.OK(c, prefs)
}

func (h *PreferenceHandler) UpdateAll(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdatePreferencesRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	prefs, err := h.preferenceService.UpdateAll(userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_update_preferences")
	}
	return response.OK(c, prefs)
}

func (h *PreferenceHandler) UpdateNameOrder(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateNameOrderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateNameOrder(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_name_order")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

func (h *PreferenceHandler) UpdateDateFormat(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateDateFormatRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateDateFormat(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_date_format")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

func (h *PreferenceHandler) UpdateTimezone(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateTimezoneRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateTimezone(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_timezone")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

func (h *PreferenceHandler) UpdateLocale(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateLocaleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateLocale(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_locale")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}
