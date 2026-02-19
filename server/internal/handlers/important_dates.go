package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ImportantDateHandler struct {
	importantDateService *services.ImportantDateService
}

func NewImportantDateHandler(importantDateService *services.ImportantDateService) *ImportantDateHandler {
	return &ImportantDateHandler{importantDateService: importantDateService}
}

// List godoc
//
//	@Summary		List important dates
//	@Description	Return all important dates for a contact
//	@Tags			important-dates
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ImportantDateResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/dates [get]
func (h *ImportantDateHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	dates, err := h.importantDateService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_important_dates")
	}
	return response.OK(c, dates)
}

// Create godoc
//
//	@Summary		Create an important date
//	@Description	Create a new important date for a contact
//	@Tags			important-dates
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			contact_id	path		string								true	"Contact ID"
//	@Param			request		body		dto.CreateImportantDateRequest		true	"Important date details"
//	@Success		201			{object}	response.APIResponse{data=dto.ImportantDateResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/dates [post]
func (h *ImportantDateHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateImportantDateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	date, err := h.importantDateService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrImportantDateLabelRequired) {
			return response.BadRequest(c, "err.label_required_without_type", nil)
		}
		return response.InternalError(c, "err.failed_to_create_important_date")
	}
	return response.Created(c, date)
}

// Update godoc
//
//	@Summary		Update an important date
//	@Description	Update an existing important date
//	@Tags			important-dates
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			contact_id	path		string								true	"Contact ID"
//	@Param			id			path		integer								true	"Important date ID"
//	@Param			request		body		dto.UpdateImportantDateRequest		true	"Important date details"
//	@Success		200			{object}	response.APIResponse{data=dto.ImportantDateResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/dates/{id} [put]
func (h *ImportantDateHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_important_date_id", nil)
	}

	var req dto.UpdateImportantDateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	date, err := h.importantDateService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrImportantDateNotFound) {
			return response.NotFound(c, "err.important_date_not_found")
		}
		if errors.Is(err, services.ErrImportantDateLabelRequired) {
			return response.BadRequest(c, "err.label_required_without_type", nil)
		}
		return response.InternalError(c, "err.failed_to_update_important_date")
	}
	return response.OK(c, date)
}

// Delete godoc
//
//	@Summary		Delete an important date
//	@Description	Permanently delete an important date
//	@Tags			important-dates
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Important date ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/dates/{id} [delete]
func (h *ImportantDateHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_important_date_id", nil)
	}

	if err := h.importantDateService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrImportantDateNotFound) {
			return response.NotFound(c, "err.important_date_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_important_date")
	}
	return response.NoContent(c)
}
