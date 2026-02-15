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

// Get godoc
//
//	@Summary		Get user preferences
//	@Description	Return the current user's preferences
//	@Tags			preferences
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.PreferencesResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/preferences [get]
func (h *PreferenceHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	prefs, err := h.preferenceService.Get(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_preferences")
	}
	return response.OK(c, prefs)
}

// UpdateAll godoc
//
//	@Summary		Update all preferences
//	@Description	Update all user preferences at once
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdatePreferencesRequest	true	"Preferences"
//	@Success		200		{object}	response.APIResponse{data=dto.PreferencesResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences [put]
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

// UpdateNameOrder godoc
//
//	@Summary		Update name order preference
//	@Description	Update the display name order preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateNameOrderRequest	true	"Name order"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/name [post]
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

// UpdateDateFormat godoc
//
//	@Summary		Update date format preference
//	@Description	Update the date format preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateDateFormatRequest	true	"Date format"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/date [post]
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

// UpdateTimezone godoc
//
//	@Summary		Update timezone preference
//	@Description	Update the timezone preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateTimezoneRequest	true	"Timezone"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/timezone [post]
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

// UpdateLocale godoc
//
//	@Summary		Update locale preference
//	@Description	Update the locale preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateLocaleRequest	true	"Locale"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/locale [post]
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

// UpdateNumberFormat godoc
//
//	@Summary		Update number format preference
//	@Description	Update the number format preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateNumberFormatRequest	true	"Number format"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/number [post]
func (h *PreferenceHandler) UpdateNumberFormat(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateNumberFormatRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateNumberFormat(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_number_format")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

// UpdateDistanceFormat godoc
//
//	@Summary		Update distance format preference
//	@Description	Update the distance format preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateDistanceFormatRequest	true	"Distance format"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/distance [post]
func (h *PreferenceHandler) UpdateDistanceFormat(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateDistanceFormatRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateDistanceFormat(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_distance_format")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

// UpdateMapsPreference godoc
//
//	@Summary		Update maps preference
//	@Description	Update the default map site preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateMapsPreferenceRequest	true	"Maps preference"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/maps [post]
func (h *PreferenceHandler) UpdateMapsPreference(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateMapsPreferenceRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.preferenceService.UpdateMapsPreference(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_maps_preference")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}

// UpdateHelpShown godoc
//
//	@Summary		Update help shown preference
//	@Description	Update the help shown preference
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateHelpShownRequest	true	"Help shown"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/preferences/help [post]
func (h *PreferenceHandler) UpdateHelpShown(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.UpdateHelpShownRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.preferenceService.UpdateHelpShown(userID, req); err != nil {
		return response.InternalError(c, "err.failed_to_update_help_shown")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}
