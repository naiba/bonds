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

type CallHandler struct {
	callService *services.CallService
}

func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{callService: callService}
}

func (h *CallHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	calls, meta, err := h.callService.List(contactID, page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_calls")
	}
	return response.Paginated(c, calls, meta)
}

func (h *CallHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateCallRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	call, err := h.callService.Create(contactID, userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_call")
	}
	return response.Created(c, call)
}

func (h *CallHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_id", nil)
	}

	var req dto.UpdateCallRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	call, err := h.callService.Update(uint(id), contactID, req)
	if err != nil {
		if errors.Is(err, services.ErrCallNotFound) {
			return response.NotFound(c, "err.call_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_call")
	}
	return response.OK(c, call)
}

func (h *CallHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_id", nil)
	}

	if err := h.callService.Delete(uint(id), contactID); err != nil {
		if errors.Is(err, services.ErrCallNotFound) {
			return response.NotFound(c, "err.call_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_call")
	}
	return response.NoContent(c)
}
