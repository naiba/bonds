package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

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
