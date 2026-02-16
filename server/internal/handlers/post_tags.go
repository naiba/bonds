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

// Add godoc
//
//	@Summary		Add a tag to a post
//	@Description	Add or create a tag and attach it to a post
//	@Tags			post-tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			journal_id	path		integer					true	"Journal ID"
//	@Param			id			path		integer					true	"Post ID"
//	@Param			request		body		dto.AddPostTagRequest	true	"Add post tag request"
//	@Success		201			{object}	response.APIResponse{data=dto.PostTagResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/tags [post]
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

// List godoc
//
//	@Summary		List post tags
//	@Description	Return all tags for a post
//	@Tags			post-tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Post ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.PostTagResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/tags [get]
func (h *PostTagHandler) List(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	tags, err := h.svc.List(uint(postID), uint(journalID))
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_post_tags")
	}
	return response.OK(c, tags)
}

// Update godoc
//
//	@Summary		Update a post tag
//	@Description	Update a tag on a post
//	@Tags			post-tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			journal_id	path		integer						true	"Journal ID"
//	@Param			id			path		integer						true	"Post ID"
//	@Param			tagId		path		integer						true	"Tag ID"
//	@Param			request		body		dto.UpdatePostTagRequest		true	"Update post tag request"
//	@Success		200			{object}	response.APIResponse{data=dto.PostTagResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/tags/{tagId} [put]
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

// Remove godoc
//
//	@Summary		Remove a tag from a post
//	@Description	Remove a tag from a post
//	@Tags			post-tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Post ID"
//	@Param			tagId		path	integer	true	"Tag ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/tags/{tagId} [delete]
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
