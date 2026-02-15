package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactTemplateHandler struct {
	contactTemplateService *services.ContactTemplateService
}

func NewContactTemplateHandler(contactTemplateService *services.ContactTemplateService) *ContactTemplateHandler {
	return &ContactTemplateHandler{contactTemplateService: contactTemplateService}
}

// Update godoc
//
//	@Summary		Update contact template
//	@Description	Update the template assigned to a contact
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			contact_id	path		string								true	"Contact ID"
//	@Param			request		body		dto.UpdateContactTemplateRequest	true	"Template details"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/template [put]
func (h *ContactTemplateHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.UpdateContactTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	contact, err := h.contactTemplateService.UpdateTemplate(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_template")
	}
	return response.OK(c, contact)
}
