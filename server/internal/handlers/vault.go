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

func (h *VaultHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	vaults, err := h.vaultService.ListVaults(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vaults")
	}
	return response.OK(c, vaults)
}

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
