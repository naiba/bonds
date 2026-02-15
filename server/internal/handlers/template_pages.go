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

// List godoc
//
//	@Summary		List template pages
//	@Description	Return all pages for a template
//	@Tags			template-pages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Template ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.TemplatePageResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages [get]
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

// Create godoc
//
//	@Summary		Create a template page
//	@Description	Create a new page for a template
//	@Tags			template-pages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer						true	"Template ID"
//	@Param			request	body		dto.CreateTemplatePageRequest	true	"Page details"
//	@Success		201		{object}	response.APIResponse{data=dto.TemplatePageResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages [post]
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

// Get godoc
//
//	@Summary		Get a template page
//	@Description	Return a template page by ID
//	@Tags			template-pages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer	true	"Template ID"
//	@Param			pageId	path		integer	true	"Page ID"
//	@Success		200		{object}	response.APIResponse{data=dto.TemplatePageResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId} [get]
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

// Update godoc
//
//	@Summary		Update a template page
//	@Description	Update an existing template page
//	@Tags			template-pages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer						true	"Template ID"
//	@Param			pageId	path		integer						true	"Page ID"
//	@Param			request	body		dto.UpdateTemplatePageRequest	true	"Page details"
//	@Success		200		{object}	response.APIResponse{data=dto.TemplatePageResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId} [put]
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

// Delete godoc
//
//	@Summary		Delete a template page
//	@Description	Delete a template page by ID
//	@Tags			template-pages
//	@Security		BearerAuth
//	@Param			id		path	integer	true	"Template ID"
//	@Param			pageId	path	integer	true	"Page ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId} [delete]
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

// UpdatePosition godoc
//
//	@Summary		Update template page position
//	@Description	Update the position of a template page
//	@Tags			template-pages
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id		path	integer						true	"Template ID"
//	@Param			pageId	path	integer						true	"Page ID"
//	@Param			request	body	dto.UpdatePositionRequest	true	"Position"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId}/position [post]
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

// ListModules godoc
//
//	@Summary		List modules for a template page
//	@Description	Return all modules assigned to a template page
//	@Tags			template-pages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer	true	"Template ID"
//	@Param			pageId	path		integer	true	"Page ID"
//	@Success		200		{object}	response.APIResponse{data=[]dto.TemplatePageModuleResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId}/modules [get]
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

// AddModule godoc
//
//	@Summary		Add a module to a template page
//	@Description	Add a module to a template page
//	@Tags			template-pages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer						true	"Template ID"
//	@Param			pageId	path		integer						true	"Page ID"
//	@Param			request	body		dto.AddModuleToPageRequest	true	"Module details"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId}/modules [post]
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

// RemoveModule godoc
//
//	@Summary		Remove a module from a template page
//	@Description	Remove a module from a template page
//	@Tags			template-pages
//	@Security		BearerAuth
//	@Param			id			path	integer	true	"Template ID"
//	@Param			pageId		path	integer	true	"Page ID"
//	@Param			moduleId	path	integer	true	"Module ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId}/modules/{moduleId} [delete]
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

// UpdateModulePosition godoc
//
//	@Summary		Update module position in a template page
//	@Description	Update the position of a module within a template page
//	@Tags			template-pages
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id			path	integer						true	"Template ID"
//	@Param			pageId		path	integer						true	"Page ID"
//	@Param			moduleId	path	integer						true	"Module ID"
//	@Param			request		body	dto.UpdatePositionRequest	true	"Position"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/templates/{id}/pages/{pageId}/modules/{moduleId}/position [post]
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
