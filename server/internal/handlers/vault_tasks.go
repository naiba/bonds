package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultTaskResponse

type VaultTaskHandler struct {
	vaultTaskService *services.VaultTaskService
}

func NewVaultTaskHandler(vaultTaskService *services.VaultTaskService) *VaultTaskHandler {
	return &VaultTaskHandler{vaultTaskService: vaultTaskService}
}

// List godoc
//
//	@Summary		List vault tasks
//	@Description	Return all tasks for a vault
//	@Tags			vault-tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultTaskResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks [get]
func (h *VaultTaskHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tasks, err := h.vaultTaskService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_tasks")
	}
	return response.OK(c, tasks)
}
