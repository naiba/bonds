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

type TaskHandler struct {
	taskService *services.TaskService
}

func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

func (h *TaskHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	tasks, err := h.taskService.List(contactID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_tasks")
	}
	return response.OK(c, tasks)
}

func (h *TaskHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	task, err := h.taskService.Create(contactID, userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_task")
	}
	return response.Created(c, task)
}

func (h *TaskHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}

	var req dto.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	task, err := h.taskService.Update(uint(id), contactID, req)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_task")
	}
	return response.OK(c, task)
}

func (h *TaskHandler) ToggleCompleted(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}

	task, err := h.taskService.ToggleCompleted(uint(id), contactID)
	if err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_task")
	}
	return response.OK(c, task)
}

func (h *TaskHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_task_id", nil)
	}

	if err := h.taskService.Delete(uint(id), contactID); err != nil {
		if errors.Is(err, services.ErrTaskNotFound) {
			return response.NotFound(c, "err.task_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_task")
	}
	return response.NoContent(c)
}
