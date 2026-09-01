package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PostHandler struct {
	postService *services.PostService
}

func NewPostHandler(postService *services.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

// List godoc
//
//	@Summary		List posts
//	@Description	Return all posts for a journal
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.PostResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts [get]
func (h *PostHandler) List(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	posts, err := h.postService.List(uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_posts")
	}
	return response.OK(c, posts)
}

// Create godoc
//
//	@Summary		Create a post
//	@Description	Create a new post in a journal
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			journal_id	path		integer					true	"Journal ID"
//	@Param			request		body		dto.CreatePostRequest	true	"Create post request"
//	@Success		201			{object}	response.APIResponse{data=dto.PostResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts [post]
func (h *PostHandler) Create(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	var req dto.CreatePostRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	post, err := h.postService.Create(uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactIDInvalid) || errors.Is(err, services.ErrContactIDsLimitExceeded) {
			return response.BadRequest(c, "err.invalid_request_body", nil)
		}
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.file_not_found")
		}
		if errors.Is(err, services.ErrInvalidContentFormat) {
			return response.BadRequest(c, "err.invalid_request_body", nil)
		}
		return response.InternalError(c, "err.failed_to_create_post")
	}
	return response.Created(c, post)
}

// Get godoc
//
//	@Summary		Get a post
//	@Description	Return a single post by ID
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Param			id			path		integer	true	"Post ID"
//	@Success		200			{object}	response.APIResponse{data=dto.PostResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id} [get]
func (h *PostHandler) Get(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	post, err := h.postService.Get(uint(id), uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_post")
	}
	return response.OK(c, post)
}

// Update godoc
//
//	@Summary		Update a post
//	@Description	Update a post by ID
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			journal_id	path		integer					true	"Journal ID"
//	@Param			id			path		integer					true	"Post ID"
//	@Param			request		body		dto.UpdatePostRequest	true	"Update post request"
//	@Success		200			{object}	response.APIResponse{data=dto.PostResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id} [put]
func (h *PostHandler) Update(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	var req dto.UpdatePostRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	post, err := h.postService.Update(uint(id), uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactIDInvalid) || errors.Is(err, services.ErrContactIDsLimitExceeded) {
			return response.BadRequest(c, "err.invalid_request_body", nil)
		}
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.file_not_found")
		}
		if errors.Is(err, services.ErrInvalidContentFormat) {
			return response.BadRequest(c, "err.invalid_request_body", nil)
		}
		return response.InternalError(c, "err.failed_to_update_post")
	}
	return response.OK(c, post)
}

// Delete godoc
//
//	@Summary		Delete a post
//	@Description	Delete a post by ID
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
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id} [delete]
func (h *PostHandler) Delete(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	if err := h.postService.Delete(uint(id), uint(journalID), vaultID); err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_post")
	}
	return response.NoContent(c)
}
