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

type NoteHandler struct {
	noteService *services.NoteService
}

func NewNoteHandler(noteService *services.NoteService) *NoteHandler {
	return &NoteHandler{noteService: noteService}
}

// List godoc
//
//	@Summary		List notes
//	@Description	Return all notes for a contact
//	@Tags			notes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.NoteResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/notes [get]
func (h *NoteHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	notes, meta, err := h.noteService.List(contactID, vaultID, page, perPage)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_notes")
	}
	return response.Paginated(c, notes, meta)
}

// Create godoc
//
//	@Summary		Create a note
//	@Description	Create a new note for a contact
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreateNoteRequest	true	"Note details"
//	@Success		201			{object}	response.APIResponse{data=dto.NoteResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/notes [post]
func (h *NoteHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateNoteRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	note, err := h.noteService.Create(contactID, vaultID, userID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_note")
	}
	return response.Created(c, note)
}

// Update godoc
//
//	@Summary		Update a note
//	@Description	Update an existing note
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Note ID"
//	@Param			request		body		dto.UpdateNoteRequest	true	"Note details"
//	@Success		200			{object}	response.APIResponse{data=dto.NoteResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/notes/{id} [put]
func (h *NoteHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_note_id", nil)
	}

	var req dto.UpdateNoteRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	note, err := h.noteService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrNoteNotFound) {
			return response.NotFound(c, "err.note_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_note")
	}
	return response.OK(c, note)
}

// Delete godoc
//
//	@Summary		Delete a note
//	@Description	Permanently delete a note
//	@Tags			notes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Note ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/notes/{id} [delete]
func (h *NoteHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_note_id", nil)
	}

	if err := h.noteService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrNoteNotFound) {
			return response.NotFound(c, "err.note_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_note")
	}
	return response.NoContent(c)
}
