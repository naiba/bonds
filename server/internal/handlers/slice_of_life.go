package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type SliceOfLifeHandler struct {
	svc *services.SliceOfLifeService
}

func NewSliceOfLifeHandler(svc *services.SliceOfLifeService) *SliceOfLifeHandler {
	return &SliceOfLifeHandler{svc: svc}
}

func (h *SliceOfLifeHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	slices, err := h.svc.List(uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_slices")
	}
	return response.OK(c, slices)
}

func (h *SliceOfLifeHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	var req dto.CreateSliceOfLifeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	slice, err := h.svc.Create(uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_slice")
	}
	return response.Created(c, slice)
}

func (h *SliceOfLifeHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_slice_id", nil)
	}
	slice, err := h.svc.Get(uint(id), uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrSliceOfLifeNotFound) {
			return response.NotFound(c, "err.slice_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_slice")
	}
	return response.OK(c, slice)
}

func (h *SliceOfLifeHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_slice_id", nil)
	}
	var req dto.UpdateSliceOfLifeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	slice, err := h.svc.Update(uint(id), uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrSliceOfLifeNotFound) {
			return response.NotFound(c, "err.slice_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_slice")
	}
	return response.OK(c, slice)
}

func (h *SliceOfLifeHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_slice_id", nil)
	}
	if err := h.svc.Delete(uint(id), uint(journalID), vaultID); err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrSliceOfLifeNotFound) {
			return response.NotFound(c, "err.slice_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_slice")
	}
	return response.NoContent(c)
}

func (h *SliceOfLifeHandler) UpdateCover(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_slice_id", nil)
	}
	var req dto.UpdateSliceCoverRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	slice, err := h.svc.UpdateCover(uint(id), uint(journalID), vaultID, req.FileID)
	if err != nil {
		if errors.Is(err, services.ErrSliceOfLifeNotFound) {
			return response.NotFound(c, "err.slice_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_slice_cover")
	}
	return response.OK(c, slice)
}

func (h *SliceOfLifeHandler) RemoveCover(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_slice_id", nil)
	}
	if err := h.svc.RemoveCover(uint(id), uint(journalID), vaultID); err != nil {
		if errors.Is(err, services.ErrSliceOfLifeNotFound) {
			return response.NotFound(c, "err.slice_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_slice_cover")
	}
	return response.NoContent(c)
}
