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

type PersonalAccessTokenHandler struct {
	service *services.PersonalAccessTokenService
}

func NewPersonalAccessTokenHandler(service *services.PersonalAccessTokenService) *PersonalAccessTokenHandler {
	return &PersonalAccessTokenHandler{service: service}
}

// List godoc
//
//	@Summary		List personal access tokens
//	@Description	Return all personal access tokens for the authenticated user
//	@Tags			personal-access-tokens
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.PersonalAccessTokenResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/tokens [get]
func (h *PersonalAccessTokenHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	tokens, err := h.service.List(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_tokens")
	}
	return response.OK(c, tokens)
}

// Create godoc
//
//	@Summary		Create a personal access token
//	@Description	Create a new personal access token. The token value is only returned once.
//	@Tags			personal-access-tokens
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreatePersonalAccessTokenRequest	true	"Token details"
//	@Success		201	{object}	response.APIResponse{data=dto.PersonalAccessTokenCreatedResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		409	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/tokens [post]
func (h *PersonalAccessTokenHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountID := middleware.GetAccountID(c)

	var req dto.CreatePersonalAccessTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	result, err := h.service.Create(userID, accountID, req)
	if err != nil {
		if errors.Is(err, services.ErrTokenNameDuplicate) {
			return response.Conflict(c, "err.token_name_duplicate")
		}
		return response.InternalError(c, "err.failed_to_create_token")
	}
	return response.Created(c, result)
}

// Delete godoc
//
//	@Summary		Delete a personal access token
//	@Description	Delete a personal access token by ID
//	@Tags			personal-access-tokens
//	@Security		BearerAuth
//	@Param			id	path	integer	true	"Token ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/tokens/{id} [delete]
func (h *PersonalAccessTokenHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_token_id", nil)
	}

	if err := h.service.Delete(uint(id), userID); err != nil {
		if errors.Is(err, services.ErrTokenNotFound) {
			return response.NotFound(c, "err.token_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_token")
	}
	return response.NoContent(c)
}