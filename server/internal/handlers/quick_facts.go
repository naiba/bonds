package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type QuickFactHandler struct {
	quickFactService *services.QuickFactService
}

func NewQuickFactHandler(quickFactService *services.QuickFactService) *QuickFactHandler {
	return &QuickFactHandler{quickFactService: quickFactService}
}

func (h *QuickFactHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	templateID, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	facts, err := h.quickFactService.List(contactID, uint(templateID))
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_quick_facts")
	}
	return response.OK(c, facts)
}

func (h *QuickFactHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	templateID, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	var req dto.CreateQuickFactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	fact, err := h.quickFactService.Create(contactID, uint(templateID), req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_quick_fact")
	}
	return response.Created(c, fact)
}

func (h *QuickFactHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_quick_fact_id", nil)
	}
	var req dto.UpdateQuickFactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	fact, err := h.quickFactService.Update(uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrQuickFactNotFound) {
			return response.NotFound(c, "err.quick_fact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_quick_fact")
	}
	return response.OK(c, fact)
}

func (h *QuickFactHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_quick_fact_id", nil)
	}
	if err := h.quickFactService.Delete(uint(id)); err != nil {
		if errors.Is(err, services.ErrQuickFactNotFound) {
			return response.NotFound(c, "err.quick_fact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_quick_fact")
	}
	return response.NoContent(c)
}
