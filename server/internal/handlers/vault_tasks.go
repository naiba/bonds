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
//	@Description	Return all tasks for a vault, optionally filtered by contact and/or status. Includes both contact-attached tasks and standalone vault-level tasks.
//	@Tags			vault-tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	query		string	false	"Filter by contact ID. Use the literal value 'none' to return only standalone tasks."
//	@Param			status		query		string	false	"Filter by status (todo|in_progress|done|blocked|cancelled)"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultTaskResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks [get]
func (h *VaultTaskHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")

	filters := services.VaultTaskFilters{}
	if cq := c.QueryParam("contact_id"); cq != "" {
		if cq == "none" {
			empty := ""
			filters.ContactID = &empty
		} else {
			filters.ContactID = &cq
		}
	}
	if sq := c.QueryParam("status"); sq != "" {
		filters.Status = &sq
	}

	tasks, err := h.vaultTaskService.List(vaultID, filters)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_tasks")
	}
	return response.OK(c, tasks)
}

// Create godoc
//
//	@Summary		Create a vault-level task
//	@Description	Create a task in the vault. ContactID is optional — when omitted or empty, the task is a standalone vault-level task.
//	@Tags			vault-tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.CreateVaultTaskRequest	true	"Task details"
//	@Success		201			{object}	response.APIResponse{data=dto.VaultTaskResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks [post]
func (h *VaultTaskHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateVaultTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	task, err := h.vaultTaskService.Create(vaultID, userID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrInvalidTaskStatus) {
			return response.BadRequest(c, "err.invalid_task_status", nil)
		}
		return response.InternalError(c, "err.failed_to_create_task")
	}
	return response.Created(c, task)
}

// Update godoc
//
//	@Summary		Update a vault task
//	@Description	Update label, description, due_at, status, and/or contact link of a task. Used by the click-to-edit modal.
//	@Tags			vault-tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Task ID"
//	@Param			request		body		dto.UpdateVaultTaskRequest	true	"Updated fields"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultTaskResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks/{id} [patch]
func (h *VaultTaskHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}
	var req dto.UpdateVaultTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	task, err := h.vaultTaskService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrInvalidTaskStatus) {
			return response.BadRequest(c, "err.invalid_task_status", nil)
		}
		return response.InternalError(c, "err.failed_to_update_task")
	}
	return response.OK(c, task)
}

// UpdateStatus godoc
//
//	@Summary		Update task status (kanban column)
//	@Description	Move a task to a different status column. Used by drag-drop across kanban columns.
//	@Tags			vault-tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			id			path		integer							true	"Task ID"
//	@Param			request		body		dto.UpdateTaskStatusRequest		true	"New status"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultTaskResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks/{id}/status [patch]
func (h *VaultTaskHandler) UpdateStatus(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}
	var req dto.UpdateTaskStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	task, err := h.vaultTaskService.UpdateStatus(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		if errors.Is(err, services.ErrInvalidTaskStatus) {
			return response.BadRequest(c, "err.invalid_task_status", nil)
		}
		return response.InternalError(c, "err.failed_to_update_task")
	}
	return response.OK(c, task)
}

// UpdatePosition godoc
//
//	@Summary		Update task position (kanban reorder)
//	@Description	Set a task's 0-based position within its kanban column. Optionally also moves to a different column when status is provided.
//	@Tags			vault-tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			id			path		integer							true	"Task ID"
//	@Param			request		body		dto.UpdateTaskPositionRequest	true	"New position (and optional new status)"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultTaskResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks/{id}/position [patch]
func (h *VaultTaskHandler) UpdatePosition(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}
	var req dto.UpdateTaskPositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	task, err := h.vaultTaskService.UpdatePosition(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		if errors.Is(err, services.ErrInvalidTaskStatus) {
			return response.BadRequest(c, "err.invalid_task_status", nil)
		}
		return response.InternalError(c, "err.failed_to_update_task")
	}
	return response.OK(c, task)
}

// Delete godoc
//
//	@Summary		Delete a vault task
//	@Description	Permanently delete a task. Both contact-attached and standalone tasks are removable through this endpoint.
//	@Tags			vault-tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Task ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/tasks/{id} [delete]
func (h *VaultTaskHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}
	if err := h.vaultTaskService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_task")
	}
	return response.NoContent(c)
}
