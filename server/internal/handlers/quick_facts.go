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

// List godoc
//
//	@Summary		List quick facts
//	@Description	Return all quick facts for a contact template
//	@Tags			quick-facts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			templateId	path		integer	true	"Template ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.QuickFactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/quickFacts/{templateId} [get]
func (h *QuickFactHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	templateID, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	facts, err := h.quickFactService.List(contactID, vaultID, uint(templateID))
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_quick_facts")
	}
	return response.OK(c, facts)
}

// Create godoc
//
//	@Summary		Create a quick fact
//	@Description	Create a new quick fact for a contact template
//	@Tags			quick-facts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			templateId	path		integer						true	"Template ID"
//	@Param			request		body		dto.CreateQuickFactRequest	true	"Create quick fact request"
//	@Success		201			{object}	response.APIResponse{data=dto.QuickFactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/quickFacts/{templateId} [post]
func (h *QuickFactHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
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
	fact, err := h.quickFactService.Create(contactID, vaultID, uint(templateID), req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_quick_fact")
	}
	return response.Created(c, fact)
}

// Update godoc
//
//	@Summary		Update a quick fact
//	@Description	Update a quick fact by ID
//	@Tags			quick-facts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			templateId	path		integer						true	"Template ID"
//	@Param			id			path		integer						true	"Quick Fact ID"
//	@Param			request		body		dto.UpdateQuickFactRequest	true	"Update quick fact request"
//	@Success		200			{object}	response.APIResponse{data=dto.QuickFactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/quickFacts/{templateId}/{id} [put]
func (h *QuickFactHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
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
	fact, err := h.quickFactService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrQuickFactNotFound) {
			return response.NotFound(c, "err.quick_fact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_quick_fact")
	}
	return response.OK(c, fact)
}

// Delete godoc
//
//	@Summary		Delete a quick fact
//	@Description	Delete a quick fact by ID
//	@Tags			quick-facts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			templateId	path	integer	true	"Template ID"
//	@Param			id			path	integer	true	"Quick Fact ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/quickFacts/{templateId}/{id} [delete]
func (h *QuickFactHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_quick_fact_id", nil)
	}
	if err := h.quickFactService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrQuickFactNotFound) {
			return response.NotFound(c, "err.quick_fact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_quick_fact")
	}
	return response.NoContent(c)
}
