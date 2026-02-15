package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type JournalHandler struct {
	journalService *services.JournalService
}

func NewJournalHandler(journalService *services.JournalService) *JournalHandler {
	return &JournalHandler{journalService: journalService}
}

// List godoc
//
//	@Summary		List journals
//	@Description	Return all journals for a vault
//	@Tags			journals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.JournalResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals [get]
func (h *JournalHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journals, err := h.journalService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_journals")
	}
	return response.OK(c, journals)
}

// Create godoc
//
//	@Summary		Create a journal
//	@Description	Create a new journal in a vault
//	@Tags			journals
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.CreateJournalRequest		true	"Create journal request"
//	@Success		201			{object}	response.APIResponse{data=dto.JournalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals [post]
func (h *JournalHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateJournalRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	journal, err := h.journalService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_journal")
	}
	return response.Created(c, journal)
}

// Get godoc
//
//	@Summary		Get a journal
//	@Description	Return a single journal by ID
//	@Tags			journals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Journal ID"
//	@Success		200			{object}	response.APIResponse{data=dto.JournalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{id} [get]
func (h *JournalHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	journal, err := h.journalService.Get(uint(id), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_journal")
	}
	return response.OK(c, journal)
}

// Update godoc
//
//	@Summary		Update a journal
//	@Description	Update a journal by ID
//	@Tags			journals
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Journal ID"
//	@Param			request		body		dto.UpdateJournalRequest		true	"Update journal request"
//	@Success		200			{object}	response.APIResponse{data=dto.JournalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{id} [put]
func (h *JournalHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	var req dto.UpdateJournalRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	journal, err := h.journalService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_journal")
	}
	return response.OK(c, journal)
}

// Delete godoc
//
//	@Summary		Delete a journal
//	@Description	Delete a journal by ID
//	@Tags			journals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Journal ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{id} [delete]
func (h *JournalHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	if err := h.journalService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_journal")
	}
	return response.NoContent(c)
}
