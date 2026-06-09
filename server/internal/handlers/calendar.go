package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.CalendarResponse

type CalendarHandler struct {
	calendarService *services.CalendarService
	icsService      *services.CalendarICSService
}

func NewCalendarHandler(calendarService *services.CalendarService, icsService *services.CalendarICSService) *CalendarHandler {
	return &CalendarHandler{calendarService: calendarService, icsService: icsService}
}

// Get godoc
//
//	@Summary		Get calendar
//	@Description	Return calendar data for a vault
//	@Tags			calendar
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			month		query		integer	false	"Month"
//	@Param			year		query		integer	false	"Year"
//	@Success		200			{object}	response.APIResponse{data=dto.CalendarResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/calendar [get]
func (h *CalendarHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	month := services.ParseIntParam(c.QueryParam("month"), 0)
	year := services.ParseIntParam(c.QueryParam("year"), 0)

	calendar, err := h.calendarService.GetCalendar(vaultID, middleware.GetUserID(c), month, year, middleware.GetLocale(c))
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_calendar_data")
	}
	return response.OK(c, calendar)
}

// GetICS godoc
//
//	@Summary		Export vault calendar as iCalendar feed
//	@Description	Return all dated items in a vault as a read-only .ics feed. Authenticate with a calendar:read scoped personal access token via the token query parameter.
//	@Tags			calendar
//	@Produce		text/calendar
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{string}	string	"iCalendar feed"
//	@Failure		403			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/calendar.ics [get]
func (h *CalendarHandler) GetICS(c echo.Context) error {
	vaultID := c.Param("vault_id")

	data, err := h.icsService.ExportVault(vaultID, middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_calendar_data")
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="bonds.ics"`)
	return c.Blob(http.StatusOK, "text/calendar; charset=utf-8", data)
}
