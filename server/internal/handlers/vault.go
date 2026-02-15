package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type VaultHandler struct {
	vaultService *services.VaultService
}

func NewVaultHandler(vaultService *services.VaultService) *VaultHandler {
	return &VaultHandler{vaultService: vaultService}
}

// List godoc
//
//	@Summary		List vaults
//	@Description	Return all vaults accessible to the current user
//	@Tags			vaults
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.VaultResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/vaults [get]
func (h *VaultHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	vaults, err := h.vaultService.ListVaults(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vaults")
	}
	return response.OK(c, vaults)
}

// Create godoc
//
//	@Summary		Create a vault
//	@Description	Create a new vault for the current account
//	@Tags			vaults
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateVaultRequest	true	"Vault details"
//	@Success		201		{object}	response.APIResponse{data=dto.VaultResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/vaults [post]
func (h *VaultHandler) Create(c echo.Context) error {
	var req dto.CreateVaultRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	accountID := middleware.GetAccountID(c)
	userID := middleware.GetUserID(c)
	vault, err := h.vaultService.CreateVault(accountID, userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_vault")
	}
	return response.Created(c, vault)
}

// Get godoc
//
//	@Summary		Get a vault
//	@Description	Return a single vault by ID
//	@Tags			vaults
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Vault ID"
//	@Success		200	{object}	response.APIResponse{data=dto.VaultResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/vaults/{id} [get]
func (h *VaultHandler) Get(c echo.Context) error {
	vaultID := c.Param("id")
	vault, err := h.vaultService.GetVault(vaultID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_vault")
	}
	return response.OK(c, vault)
}

// Update godoc
//
//	@Summary		Update a vault
//	@Description	Update an existing vault's name and description
//	@Tags			vaults
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Vault ID"
//	@Param			request	body		dto.UpdateVaultRequest	true	"Vault details"
//	@Success		200		{object}	response.APIResponse{data=dto.VaultResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/vaults/{id} [put]
func (h *VaultHandler) Update(c echo.Context) error {
	vaultID := c.Param("id")
	var req dto.UpdateVaultRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	vault, err := h.vaultService.UpdateVault(vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_vault")
	}
	return response.OK(c, vault)
}

// Delete godoc
//
//	@Summary		Delete a vault
//	@Description	Permanently delete a vault and all its data
//	@Tags			vaults
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Vault ID"
//	@Success		204	"No Content"
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/vaults/{id} [delete]
func (h *VaultHandler) Delete(c echo.Context) error {
	vaultID := c.Param("id")
	if err := h.vaultService.DeleteVault(vaultID); err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_vault")
	}
	return response.NoContent(c)
}

func VaultPermissionMiddleware(vaultService *services.VaultService, requiredPerm int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := middleware.GetUserID(c)
			vaultID := c.Param("vault_id")
			if vaultID == "" {
				vaultID = c.Param("id")
			}

			if err := vaultService.CheckUserVaultAccess(userID, vaultID, requiredPerm); err != nil {
				if errors.Is(err, services.ErrVaultForbidden) {
					return response.Forbidden(c, "err.no_vault_access_short")
				}
				if errors.Is(err, services.ErrInsufficientPerm) {
					return response.Forbidden(c, "err.insufficient_permissions_short")
				}
				return response.InternalError(c, "err.failed_to_check_permissions")
			}

			return next(c)
		}
	}
}
