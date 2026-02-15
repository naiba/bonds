package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.AddressReportItem

type ReportHandler struct {
	reportService *services.ReportService
}

func NewReportHandler(reportService *services.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// Addresses godoc
//
//	@Summary		Get address report
//	@Description	Return address statistics for a vault
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.AddressReportItem}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reports/addresses [get]
func (h *ReportHandler) Addresses(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.AddressReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

// ImportantDates godoc
//
//	@Summary		Get important dates report
//	@Description	Return important dates for all contacts in a vault
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ImportantDateReportItem}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reports/importantDates [get]
func (h *ReportHandler) ImportantDates(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.ImportantDatesReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_important_dates_report")
	}
	return response.OK(c, data)
}

// MoodTrackingEvents godoc
//
//	@Summary		Get mood tracking report
//	@Description	Return mood tracking statistics for a vault
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.MoodReportItem}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reports/moodTrackingEvents [get]
func (h *ReportHandler) MoodTrackingEvents(c echo.Context) error {
	vaultID := c.Param("vault_id")
	data, err := h.reportService.MoodReport(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_mood_report")
	}
	return response.OK(c, data)
}
