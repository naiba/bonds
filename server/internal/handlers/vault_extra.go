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

// List godoc
//
//	@Summary		List vault reminders
//	@Description	Return all reminders for a vault
//	@Tags			reminders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reminders [get]
func (h *VaultReminderHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	reminders, err := h.svc.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_reminders")
	}
	return response.OK(c, reminders)
}

// GetMonth godoc
//
//	@Summary		Get calendar month
//	@Description	Return calendar data for a specific month
//	@Tags			calendar
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			year		path		integer	true	"Year"
//	@Param			month		path		integer	true	"Month"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/calendar/years/{year}/months/{month} [get]
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

// GetDay godoc
//
//	@Summary		Get calendar day
//	@Description	Return calendar data for a specific day
//	@Tags			calendar
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			year		path		integer	true	"Year"
//	@Param			month		path		integer	true	"Month"
//	@Param			day			path		integer	true	"Day"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/calendar/years/{year}/months/{month}/days/{day} [get]
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

// Index godoc
//
//	@Summary		List available reports
//	@Description	Return the list of available report types
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ReportIndexItem}
//	@Router			/vaults/{vault_id}/reports [get]
func (h *ReportHandler) Index(c echo.Context) error {
	reports := []dto.ReportIndexItem{
		{Key: "addresses", Name: "Addresses"},
		{Key: "importantDates", Name: "Important Dates"},
		{Key: "moodTrackingEvents", Name: "Mood Tracking Events"},
	}
	return response.OK(c, reports)
}

// AddressesByCity godoc
//
//	@Summary		Get addresses by city
//	@Description	Return contacts with addresses in a specific city
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			city		path		string	true	"City name"
//	@Success		200			{object}	response.APIResponse{data=[]dto.AddressContactItem}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reports/addresses/city/{city} [get]
func (h *ReportHandler) AddressesByCity(c echo.Context) error {
	vaultID := c.Param("vault_id")
	city := c.Param("city")
	data, err := h.reportService.AddressesByCity(vaultID, city)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

// AddressesByCountry godoc
//
//	@Summary		Get addresses by country
//	@Description	Return contacts with addresses in a specific country
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			country		path		string	true	"Country code"
//	@Success		200			{object}	response.APIResponse{data=[]dto.AddressContactItem}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/reports/addresses/country/{country} [get]
func (h *ReportHandler) AddressesByCountry(c echo.Context) error {
	vaultID := c.Param("vault_id")
	country := c.Param("country")
	data, err := h.reportService.AddressesByCountry(vaultID, country)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_address_report")
	}
	return response.OK(c, data)
}

// ListPhotos godoc
//
//	@Summary		List vault photos
//	@Description	Return all photo files for a vault
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files/photos [get]
func (h *VaultFileHandler) ListPhotos(c echo.Context) error {
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	files, meta, err := h.vaultFileService.ListByType(vaultID, "photo", page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.Paginated(c, files, meta)
}

// ListDocuments godoc
//
//	@Summary		List vault documents
//	@Description	Return all document files for a vault
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files/documents [get]
func (h *VaultFileHandler) ListDocuments(c echo.Context) error {
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	files, meta, err := h.vaultFileService.ListByType(vaultID, "document", page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.Paginated(c, files, meta)
}

// ListAvatars godoc
//
//	@Summary		List vault avatars
//	@Description	Return all avatar files for a vault
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files/avatars [get]
func (h *VaultFileHandler) ListAvatars(c echo.Context) error {
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	files, meta, err := h.vaultFileService.ListByType(vaultID, "avatar", page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.Paginated(c, files, meta)
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

// List godoc
//
//	@Summary		List most consulted contacts
//	@Description	Return most consulted contacts for the current user in a vault
//	@Tags			search
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/search/mostConsulted [get]
func (h *MostConsultedHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	data, err := h.svc.List(vaultID, userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_most_consulted")
	}
	return response.OK(c, data)
}

// UpdatePosition godoc
//
//	@Summary		Update personalize entity position
//	@Description	Update the position of a personalize entity
//	@Tags			personalize
//	@Accept			json
//	@Security		BearerAuth
//	@Param			entity	path	string					true	"Entity type"
//	@Param			id		path	integer					true	"Entity ID"
//	@Param			request	body	dto.UpdatePositionRequest	true	"Position"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/{entity}/{id}/position [post]
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
