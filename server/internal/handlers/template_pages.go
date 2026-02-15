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

type TemplatePageHandler struct {
	svc *services.TemplatePageService
}

func NewTemplatePageHandler(svc *services.TemplatePageService) *TemplatePageHandler {
	return &TemplatePageHandler{svc: svc}
}

func (h *TemplatePageHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	pages, err := h.svc.List(uint(templateID), accountID)
	if err != nil {
		if errors.Is(err, services.ErrTemplateNotFound) {
			return response.NotFound(c, "err.template_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_template_pages")
	}
	return response.OK(c, pages)
}

func (h *TemplatePageHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	var req dto.CreateTemplatePageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	page, err := h.svc.Create(uint(templateID), accountID, req)
	if err != nil {
		if errors.Is(err, services.ErrTemplateNotFound) {
			return response.NotFound(c, "err.template_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_template_page")
	}
	return response.Created(c, page)
}

func (h *TemplatePageHandler) Get(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	page, err := h.svc.Get(uint(pageID), accountID)
	if err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_template_page")
	}
	return response.OK(c, page)
}

func (h *TemplatePageHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	var req dto.UpdateTemplatePageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	page, err := h.svc.Update(uint(pageID), accountID, req)
	if err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_template_page")
	}
	return response.OK(c, page)
}

func (h *TemplatePageHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	if err := h.svc.Delete(uint(pageID), accountID); err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		if errors.Is(err, services.ErrTemplatePageCannotBeDeleted) {
			return response.BadRequest(c, "err.template_page_cannot_be_deleted", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_template_page")
	}
	return response.NoContent(c)
}

func (h *TemplatePageHandler) UpdatePosition(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.svc.UpdatePosition(uint(pageID), accountID, req.Position); err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_template_page_position")
	}
	return response.NoContent(c)
}

func (h *TemplatePageHandler) ListModules(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	modules, err := h.svc.ListModules(uint(pageID), accountID)
	if err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_template_page_modules")
	}
	return response.OK(c, modules)
}

func (h *TemplatePageHandler) AddModule(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	var req dto.AddModuleToPageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.svc.AddModule(uint(pageID), accountID, req); err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_module_to_page")
	}
	return response.Created(c, nil)
}

func (h *TemplatePageHandler) RemoveModule(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	moduleID, err := strconv.ParseUint(c.Param("moduleId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_module_id", nil)
	}
	if err := h.svc.RemoveModule(uint(pageID), uint(moduleID), accountID); err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_module_from_page")
	}
	return response.NoContent(c)
}

func (h *TemplatePageHandler) UpdateModulePosition(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	pageID, err := strconv.ParseUint(c.Param("pageId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_page_id", nil)
	}
	moduleID, err := strconv.ParseUint(c.Param("moduleId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_module_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.svc.UpdateModulePosition(uint(pageID), uint(moduleID), accountID, req.Position); err != nil {
		if errors.Is(err, services.ErrTemplatePageNotFound) {
			return response.NotFound(c, "err.template_page_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_module_position")
	}
	return response.NoContent(c)
}
