package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ActivityHandler struct{ activityService *services.ActivityService }

func NewActivityHandler(service *services.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: service}
}

// List godoc
//
//	@Summary List activities
//	@Tags activities
//	@Produce json
//	@Security BearerAuth
//	@Param vault_id path string true "Vault ID"
//	@Param contact_id query string false "Contact ID"
//	@Param page query integer false "Page"
//	@Param per_page query integer false "Items per page"
//	@Success 200 {object} response.APIResponse{data=[]dto.ActivityResponse}
//	@Router /vaults/{vault_id}/activities [get]
func (h *ActivityHandler) List(c *echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	items, meta, err := h.activityService.ListForUser(c.Param("vault_id"), middleware.GetUserID(c), c.QueryParam("contact_id"), page, perPage)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_activities")
	}
	return response.Paginated(c, items, meta)
}

// Get godoc
//
//	@Summary Get activity details
//	@Tags activities
//	@Produce json
//	@Security BearerAuth
//	@Param vault_id path string true "Vault ID"
//	@Param id path integer true "Activity ID"
//	@Success 200 {object} response.APIResponse{data=dto.ActivityResponse}
//	@Router /vaults/{vault_id}/activities/{id} [get]
func (h *ActivityHandler) Get(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_activity_id", nil)
	}
	item, err := h.activityService.Get(c.Param("vault_id"), middleware.GetUserID(c), uint(id))
	if err != nil {
		return activityError(c, err, "err.failed_to_get_activity")
	}
	return response.OK(c, item)
}

// Create godoc
//
//	@Summary Create an activity
//	@Tags activities
//	@Accept json
//	@Produce json
//	@Security BearerAuth
//	@Param vault_id path string true "Vault ID"
//	@Param request body dto.ActivityUpsertRequest true "Activity"
//	@Success 201 {object} response.APIResponse{data=dto.ActivityResponse}
//	@Router /vaults/{vault_id}/activities [post]
func (h *ActivityHandler) Create(c *echo.Context) error {
	var req dto.ActivityUpsertRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	item, err := h.activityService.CreateForUser(c.Param("vault_id"), middleware.GetUserID(c), req)
	if err != nil {
		return activityError(c, err, "err.failed_to_create_activity")
	}
	return response.Created(c, item)
}

// Update godoc
//
//	@Summary Update an activity
//	@Tags activities
//	@Accept json
//	@Produce json
//	@Security BearerAuth
//	@Param vault_id path string true "Vault ID"
//	@Param id path integer true "Activity ID"
//	@Param request body dto.ActivityUpsertRequest true "Activity"
//	@Success 200 {object} response.APIResponse{data=dto.ActivityResponse}
//	@Router /vaults/{vault_id}/activities/{id} [put]
func (h *ActivityHandler) Update(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_activity_id", nil)
	}
	var req dto.ActivityUpsertRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	item, err := h.activityService.UpdateForUser(c.Param("vault_id"), middleware.GetUserID(c), uint(id), req)
	if err != nil {
		return activityError(c, err, "err.failed_to_update_activity")
	}
	return response.OK(c, item)
}

// Delete godoc
//
//	@Summary Delete an activity
//	@Tags activities
//	@Produce json
//	@Security BearerAuth
//	@Param vault_id path string true "Vault ID"
//	@Param id path integer true "Activity ID"
//	@Success 204
//	@Router /vaults/{vault_id}/activities/{id} [delete]
func (h *ActivityHandler) Delete(c *echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_activity_id", nil)
	}
	if err := h.activityService.Delete(c.Param("vault_id"), uint(id)); err != nil {
		return activityError(c, err, "err.failed_to_delete_activity")
	}
	return response.NoContent(c)
}

func activityError(c *echo.Context, err error, fallback string) error {
	switch {
	case errors.Is(err, services.ErrActivityNotFound):
		return response.NotFound(c, "err.activity_not_found")
	case errors.Is(err, services.ErrContactNotFound):
		return response.NotFound(c, "err.contact_not_found")
	case errors.Is(err, services.ErrInvalidActivityTime):
		return response.BadRequest(c, "err.invalid_activity_time", nil)
	case errors.Is(err, services.ErrInvalidActivityInput):
		return response.BadRequest(c, "err.invalid_request_body", nil)
	case errors.Is(err, services.ErrInvalidContentFormat):
		return response.BadRequest(c, "err.invalid_request_body", nil)
	case errors.Is(err, services.ErrFileNotFound):
		return response.NotFound(c, "err.file_not_found")
	default:
		return response.InternalError(c, fallback)
	}
}
