package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

// GetPhotos godoc
//
//	@Summary		Get journal photos
//	@Description	Return all photos for a journal
//	@Tags			journals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Journal ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{id}/photos [get]
func (h *JournalHandler) GetPhotos(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	photos, err := h.journalService.GetPhotos(uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_journal_photos")
	}
	return response.OK(c, photos)
}

// GetByYear godoc
//
//	@Summary		Get journal posts by year
//	@Description	Return all posts for a journal filtered by year
//	@Tags			journals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Journal ID"
//	@Param			year		path		integer	true	"Year"
//	@Success		200			{object}	response.APIResponse{data=[]dto.PostResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{id}/years/{year} [get]
func (h *JournalHandler) GetByYear(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		return response.BadRequest(c, "err.invalid_year", nil)
	}
	posts, err := h.journalService.GetByYear(uint(journalID), vaultID, year)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_journal_posts_by_year")
	}
	return response.OK(c, posts)
}

// SetSlice godoc
//
//	@Summary		Set post slice of life
//	@Description	Assign a post to a slice of life
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			journal_id	path		integer						true	"Journal ID"
//	@Param			id			path		integer						true	"Post ID"
//	@Param			request		body		dto.UpdatePostSliceRequest	true	"Set slice request"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/slices [put]
func (h *PostHandler) SetSlice(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	var req dto.UpdatePostSliceRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.postService.SetSliceOfLife(uint(postID), uint(journalID), req.SliceOfLifeID); err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		return response.InternalError(c, "err.failed_to_set_post_slice")
	}
	return response.NoContent(c)
}

// ClearSlice godoc
//
//	@Summary		Clear post slice of life
//	@Description	Remove a post from its slice of life
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Post ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/slices [delete]
func (h *PostHandler) ClearSlice(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	if err := h.postService.ClearSliceOfLife(uint(postID), uint(journalID)); err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		return response.InternalError(c, "err.failed_to_clear_post_slice")
	}
	return response.NoContent(c)
}
