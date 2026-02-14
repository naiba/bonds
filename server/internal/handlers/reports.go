package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ReportHandler struct {
	reportService *services.ReportService
}

func NewReportHandler(reportService *services.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) Addresses(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.AddressReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

func (h *ReportHandler) ImportantDates(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.ImportantDatesReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_important_dates_report")
	}
	return response.OK(c, data)
}

func (h *ReportHandler) MoodTrackingEvents(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.MoodReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_mood_report")
	}
	return response.OK(c, data)
}
