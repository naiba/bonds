package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactReligionHandler struct {
	contactReligionService *services.ContactReligionService
}

func NewContactReligionHandler(contactReligionService *services.ContactReligionService) *ContactReligionHandler {
	return &ContactReligionHandler{contactReligionService: contactReligionService}
}

func (h *ContactReligionHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.UpdateContactReligionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	contact, err := h.contactReligionService.Update(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_religion")
	}
	return response.OK(c, contact)
}
