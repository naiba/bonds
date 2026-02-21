package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type OAuthProviderHandler struct {
	svc *services.OAuthProviderService
}

func NewOAuthProviderHandler(svc *services.OAuthProviderService) *OAuthProviderHandler {
	return &OAuthProviderHandler{svc: svc}
}

// List godoc
//
//	@Summary		List OAuth providers
//	@Description	List all configured OAuth providers (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.OAuthProviderResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/oauth-providers [get]
func (h *OAuthProviderHandler) List(c echo.Context) error {
	providers, err := h.svc.List()
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_providers")
	}
	return response.OK(c, providers)
}

// Create godoc
//
//	@Summary		Create OAuth provider
//	@Description	Create a new OAuth provider configuration (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateOAuthProviderRequest	true	"Provider config"
//	@Success		201		{object}	response.APIResponse{data=dto.OAuthProviderResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/oauth-providers [post]
func (h *OAuthProviderHandler) Create(c echo.Context) error {
	var req dto.CreateOAuthProviderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	provider, err := h.svc.Create(req)
	if err != nil {
		if errors.Is(err, services.ErrOAuthProviderNameExists) {
			return response.BadRequest(c, "err.oauth_provider_name_exists", nil)
		}
		return response.InternalError(c, "err.failed_to_create_provider")
	}
	return response.Created(c, provider)
}

// Update godoc
//
//	@Summary		Update OAuth provider
//	@Description	Update an existing OAuth provider configuration (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int								true	"Provider ID"
//	@Param			request	body		dto.UpdateOAuthProviderRequest	true	"Fields to update"
//	@Success		200		{object}	response.APIResponse{data=dto.OAuthProviderResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/oauth-providers/{id} [put]
func (h *OAuthProviderHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	var req dto.UpdateOAuthProviderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	provider, err := h.svc.Update(uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrOAuthProviderNotFoundByID) {
			return response.NotFound(c, "err.oauth_provider_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_provider")
	}
	return response.OK(c, provider)
}

// Delete godoc
//
//	@Summary		Delete OAuth provider
//	@Description	Delete an OAuth provider configuration (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	int	true	"Provider ID"
//	@Success		204
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/oauth-providers/{id} [delete]
func (h *OAuthProviderHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		if errors.Is(err, services.ErrOAuthProviderNotFoundByID) {
			return response.NotFound(c, "err.oauth_provider_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_provider")
	}
	return response.NoContent(c)
}
