package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactInformationHandler struct {
	contactInformationService *services.ContactInformationService
}

func NewContactInformationHandler(contactInformationService *services.ContactInformationService) *ContactInformationHandler {
	return &ContactInformationHandler{contactInformationService: contactInformationService}
}

func (h *ContactInformationHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	items, err := h.contactInformationService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_contact_information")
	}
	return response.OK(c, items)
}

func (h *ContactInformationHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateContactInformationRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	item, err := h.contactInformationService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_contact_information")
	}
	return response.Created(c, item)
}

func (h *ContactInformationHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_contact_information_id", nil)
	}

	var req dto.UpdateContactInformationRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	item, err := h.contactInformationService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactInformationNotFound) {
			return response.NotFound(c, "err.contact_information_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_contact_information")
	}
	return response.OK(c, item)
}

func (h *ContactInformationHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_contact_information_id", nil)
	}

	if err := h.contactInformationService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactInformationNotFound) {
			return response.NotFound(c, "err.contact_information_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_contact_information")
	}
	return response.NoContent(c)
}
