package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.ContactTabsResponse

type ContactTabHandler struct {
	contactTabService *services.ContactTabService
}

func NewContactTabHandler(contactTabService *services.ContactTabService) *ContactTabHandler {
	return &ContactTabHandler{contactTabService: contactTabService}
}

// GetTabs godoc
//
//	@Summary		Get contact tabs
//	@Description	Get the template pages and their associated modules for a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactTabsResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/tabs [get]
func (h *ContactTabHandler) GetTabs(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	tabs, err := h.contactTabService.GetTabs(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_tabs")
	}
	return response.OK(c, tabs)
}
