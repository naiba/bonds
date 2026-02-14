package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type CalendarHandler struct {
	calendarService *services.CalendarService
}

func NewCalendarHandler(calendarService *services.CalendarService) *CalendarHandler {
	return &CalendarHandler{calendarService: calendarService}
}

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
