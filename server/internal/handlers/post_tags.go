package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PostTagHandler struct {
	svc *services.PostTagService
}

func NewPostTagHandler(svc *services.PostTagService) *PostTagHandler {
	return &PostTagHandler{svc: svc}
}

func (h *PostTagHandler) Add(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	var req dto.AddPostTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	tag, err := h.svc.Add(uint(postID), uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrTagNotFound) {
			return response.NotFound(c, "err.tag_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_post_tag")
	}
	return response.Created(c, tag)
}

func (h *PostTagHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	tagID, err := strconv.ParseUint(c.Param("tagId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_tag_id", nil)
	}
	var req dto.UpdatePostTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	tag, err := h.svc.Update(uint(tagID), uint(postID), uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrTagNotFound) {
			return response.NotFound(c, "err.tag_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_post_tag")
	}
	return response.OK(c, tag)
}

func (h *PostTagHandler) Remove(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	tagID, err := strconv.ParseUint(c.Param("tagId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_tag_id", nil)
	}
	if err := h.svc.Remove(uint(tagID), uint(postID), uint(journalID)); err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrPostTagNotFound) {
			return response.NotFound(c, "err.post_tag_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_post_tag")
	}
	return response.NoContent(c)
}
