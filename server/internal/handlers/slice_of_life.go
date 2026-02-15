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

// List godoc
//
//	@Summary		List slices of life
//	@Description	Return all slices of life for a journal
//	@Tags			slices-of-life
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.SliceOfLifeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices [get]
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

// Create godoc
//
//	@Summary		Create a slice of life
//	@Description	Create a new slice of life for a journal
//	@Tags			slices-of-life
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			journal_id	path		integer							true	"Journal ID"
//	@Param			request		body		dto.CreateSliceOfLifeRequest		true	"Create slice of life request"
//	@Success		201			{object}	response.APIResponse{data=dto.SliceOfLifeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices [post]
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

// Get godoc
//
//	@Summary		Get a slice of life
//	@Description	Return a single slice of life by ID
//	@Tags			slices-of-life
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Param			id			path		integer	true	"Slice of Life ID"
//	@Success		200			{object}	response.APIResponse{data=dto.SliceOfLifeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices/{id} [get]
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

// Update godoc
//
//	@Summary		Update a slice of life
//	@Description	Update a slice of life by ID
//	@Tags			slices-of-life
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			journal_id	path		integer							true	"Journal ID"
//	@Param			id			path		integer							true	"Slice of Life ID"
//	@Param			request		body		dto.UpdateSliceOfLifeRequest		true	"Update slice of life request"
//	@Success		200			{object}	response.APIResponse{data=dto.SliceOfLifeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices/{id} [put]
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

// Delete godoc
//
//	@Summary		Delete a slice of life
//	@Description	Delete a slice of life by ID
//	@Tags			slices-of-life
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Slice of Life ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices/{id} [delete]
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

// UpdateCover godoc
//
//	@Summary		Update slice of life cover
//	@Description	Set the cover image for a slice of life
//	@Tags			slices-of-life
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			journal_id	path		integer						true	"Journal ID"
//	@Param			id			path		integer						true	"Slice of Life ID"
//	@Param			request		body		dto.UpdateSliceCoverRequest	true	"Update cover request"
//	@Success		200			{object}	response.APIResponse{data=dto.SliceOfLifeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices/{id}/cover [put]
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

// RemoveCover godoc
//
//	@Summary		Remove slice of life cover
//	@Description	Remove the cover image from a slice of life
//	@Tags			slices-of-life
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Slice of Life ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/slices/{id}/cover [delete]
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
