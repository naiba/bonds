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

// List godoc
//
//	@Summary		List goals for a contact
//	@Description	Return all goals belonging to a contact
//	@Tags			goals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.GoalResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals [get]
func (h *GoalHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	goals, err := h.goalService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_goals")
	}
	return response.OK(c, goals)
}

// Create godoc
//
//	@Summary		Create a goal
//	@Description	Create a new goal for a contact
//	@Tags			goals
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreateGoalRequest	true	"Goal details"
//	@Success		201			{object}	response.APIResponse{data=dto.GoalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals [post]
func (h *GoalHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	var req dto.CreateGoalRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	goal, err := h.goalService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_goal")
	}
	return response.Created(c, goal)
}

// Get godoc
//
//	@Summary		Get a goal
//	@Description	Get a specific goal by ID
//	@Tags			goals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Goal ID"
//	@Success		200			{object}	response.APIResponse{data=dto.GoalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals/{id} [get]
func (h *GoalHandler) Get(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	goal, err := h.goalService.Get(uint(id), contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_goal")
	}
	return response.OK(c, goal)
}

// Update godoc
//
//	@Summary		Update a goal
//	@Description	Update an existing goal
//	@Tags			goals
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Goal ID"
//	@Param			request		body		dto.UpdateGoalRequest	true	"Goal details"
//	@Success		200			{object}	response.APIResponse{data=dto.GoalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals/{id} [put]
func (h *GoalHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
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
	goal, err := h.goalService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_goal")
	}
	return response.OK(c, goal)
}

// AddStreak godoc
//
//	@Summary		Add a streak to a goal
//	@Description	Record a new streak entry for a goal
//	@Tags			goals
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Goal ID"
//	@Param			request		body		dto.AddStreakRequest		true	"Streak details"
//	@Success		200			{object}	response.APIResponse{data=dto.GoalResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals/{id}/streaks [put]
func (h *GoalHandler) AddStreak(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	var req dto.AddStreakRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	goal, err := h.goalService.AddStreak(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_streak")
	}
	return response.OK(c, goal)
}

// Delete godoc
//
//	@Summary		Delete a goal
//	@Description	Delete a goal from a contact
//	@Tags			goals
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Goal ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/goals/{id} [delete]
func (h *GoalHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_goal_id", nil)
	}
	if err := h.goalService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrGoalNotFound) {
			return response.NotFound(c, "err.goal_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_goal")
	}
	return response.NoContent(c)
}
