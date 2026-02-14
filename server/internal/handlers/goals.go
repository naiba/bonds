package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type GoalHandler struct {
	goalService *services.GoalService
}

func NewGoalHandler(goalService *services.GoalService) *GoalHandler {
	return &GoalHandler{goalService: goalService}
}

func (h *GoalHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	goals, err := h.goalService.List(contactID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_goals")
	}
	return response.OK(c, goals)
}

func (h *GoalHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	var req dto.CreateGoalRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	goal, err := h.goalService.Create(contactID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_goal")
	}
	return response.Created(c, goal)
}

func (h *GoalHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	goal, err := h.goalService.Get(uint(id))
	if err != nil {
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_goal")
	}
	return response.OK(c, goal)
}

func (h *GoalHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	var req dto.UpdateGoalRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	goal, err := h.goalService.Update(uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_goal")
	}
	return response.OK(c, goal)
}

func (h *GoalHandler) AddStreak(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	var req dto.AddStreakRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	goal, err := h.goalService.AddStreak(uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_streak")
	}
	return response.OK(c, goal)
}

func (h *GoalHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	if err := h.goalService.Delete(uint(id)); err != nil {
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_goal")
	}
	return response.NoContent(c)
}
