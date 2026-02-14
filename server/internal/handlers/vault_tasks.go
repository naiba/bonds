package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type VaultTaskHandler struct {
	vaultTaskService *services.VaultTaskService
}

func NewVaultTaskHandler(vaultTaskService *services.VaultTaskService) *VaultTaskHandler {
	return &VaultTaskHandler{vaultTaskService: vaultTaskService}
}

func (h *VaultTaskHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tasks, err := h.vaultTaskService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_tasks")
	}
	return response.OK(c, tasks)
}
