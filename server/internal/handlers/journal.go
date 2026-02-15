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

func (h *JournalHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journals, err := h.journalService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_journals")
	}
	return response.OK(c, journals)
}

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
