package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactJobHandler struct {
	contactJobService *services.ContactJobService
}

func NewContactJobHandler(contactJobService *services.ContactJobService) *ContactJobHandler {
	return &ContactJobHandler{contactJobService: contactJobService}
}

func (h *ContactJobHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.UpdateJobInfoRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	contact, err := h.contactJobService.Update(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_job_info")
	}
	return response.OK(c, contact)
}

func (h *ContactJobHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	contact, err := h.contactJobService.Delete(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_job_info")
	}
	return response.OK(c, contact)
}
