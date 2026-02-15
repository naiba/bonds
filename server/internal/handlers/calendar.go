package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.CalendarResponse

type CalendarHandler struct {
	calendarService *services.CalendarService
}

func NewCalendarHandler(calendarService *services.CalendarService) *CalendarHandler {
	return &CalendarHandler{calendarService: calendarService}
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

	calendar, err := h.calendarService.GetCalendar(vaultID, month, year)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_calendar_data")
	}
	return response.OK(c, calendar)
}
