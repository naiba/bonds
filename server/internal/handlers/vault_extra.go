package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type VaultReminderHandler struct {
	svc *services.VaultReminderService
}

func NewVaultReminderHandler(svc *services.VaultReminderService) *VaultReminderHandler {
	return &VaultReminderHandler{svc: svc}
}

func (h *VaultReminderHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	reminders, err := h.svc.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_reminders")
	}
	return response.OK(c, reminders)
}

func (h *CalendarHandler) GetMonth(c echo.Context) error {
	vaultID := c.Param("vault_id")
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_year", nil)
	}
	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_month", nil)
	}
	calendar, err := h.calendarService.GetCalendarMonth(vaultID, year, month)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_calendar_data")
	}
	return response.OK(c, calendar)
}

func (h *CalendarHandler) GetDay(c echo.Context) error {
	vaultID := c.Param("vault_id")
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_year", nil)
	}
	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_month", nil)
	}
	day, err := strconv.Atoi(c.Param("day"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_day", nil)
	}
	calendar, err := h.calendarService.GetCalendarDay(vaultID, year, month, day)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_calendar_data")
	}
	return response.OK(c, calendar)
}

func (h *ReportHandler) Index(c echo.Context) error {
	reports := []dto.ReportIndexItem{
		{Key: "addresses", Name: "Addresses"},
		{Key: "importantDates", Name: "Important Dates"},
		{Key: "moodTrackingEvents", Name: "Mood Tracking Events"},
	}
	return response.OK(c, reports)
}

func (h *ReportHandler) AddressesByCity(c echo.Context) error {
	vaultID := c.Param("vault_id")
	city := c.Param("city")
	data, err := h.reportService.AddressesByCity(vaultID, city)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

func (h *ReportHandler) AddressesByCountry(c echo.Context) error {
	vaultID := c.Param("vault_id")
	country := c.Param("country")
	data, err := h.reportService.AddressesByCountry(vaultID, country)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

func (h *VaultFileHandler) ListPhotos(c echo.Context) error {
	vaultID := c.Param("vault_id")
	files, err := h.vaultFileService.ListByType(vaultID, "photo")
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.OK(c, files)
}

func (h *VaultFileHandler) ListDocuments(c echo.Context) error {
	vaultID := c.Param("vault_id")
	files, err := h.vaultFileService.ListByType(vaultID, "document")
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.OK(c, files)
}

func (h *VaultFileHandler) ListAvatars(c echo.Context) error {
	vaultID := c.Param("vault_id")
	files, err := h.vaultFileService.ListByType(vaultID, "avatar")
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.OK(c, files)
}

func (h *VaultHandler) UpdateDefaultTab(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.UpdateDefaultTabRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.vaultService.UpdateDefaultTab(vaultID, req.DefaultActivityTab); err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_default_tab")
	}
	return response.NoContent(c)
}

type MostConsultedHandler struct {
	svc *services.MostConsultedService
}

func NewMostConsultedHandler(svc *services.MostConsultedService) *MostConsultedHandler {
	return &MostConsultedHandler{svc: svc}
}

func (h *MostConsultedHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	data, err := h.svc.List(vaultID, userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_most_consulted")
	}
	return response.OK(c, data)
}

func (h *PersonalizeHandler) UpdatePosition(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	entity := c.Param("entity")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_entity_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.personalizeService.UpdatePosition(accountID, entity, uint(id), req.Position); err != nil {
		if errors.Is(err, services.ErrUnknownEntityType) {
			return response.NotFound(c, "err.unknown_entity_type")
		}
		if errors.Is(err, services.ErrPersonalizeEntityNotFound) {
			return response.NotFound(c, "err.entity_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_position")
	}
	return response.NoContent(c)
}
