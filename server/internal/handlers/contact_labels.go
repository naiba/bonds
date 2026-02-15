package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactLabelHandler struct {
	contactLabelService *services.ContactLabelService
}

func NewContactLabelHandler(contactLabelService *services.ContactLabelService) *ContactLabelHandler {
	return &ContactLabelHandler{contactLabelService: contactLabelService}
}

func (h *ContactLabelHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	labels, err := h.contactLabelService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_contact_labels")
	}
	return response.OK(c, labels)
}

func (h *ContactLabelHandler) Add(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.AddContactLabelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.LabelID == 0 {
		return response.ValidationError(c, map[string]string{"label_id": "label_id is required"})
	}

	label, err := h.contactLabelService.Add(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrLabelNotFound) {
			return response.NotFound(c, "err.label_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_contact_label")
	}
	return response.Created(c, label)
}

func (h *ContactLabelHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_label_id", nil)
	}

	var req dto.UpdateContactLabelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.LabelID == 0 {
		return response.ValidationError(c, map[string]string{"label_id": "label_id is required"})
	}

	label, err := h.contactLabelService.Update(contactID, vaultID, uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactLabelNotFound) {
			return response.NotFound(c, "err.contact_label_not_found")
		}
		if errors.Is(err, services.ErrLabelNotFound) {
			return response.NotFound(c, "err.label_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_contact_label")
	}
	return response.OK(c, label)
}

func (h *ContactLabelHandler) Remove(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_label_id", nil)
	}

	if err := h.contactLabelService.Remove(contactID, vaultID, uint(id)); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactLabelNotFound) {
			return response.NotFound(c, "err.contact_label_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_contact_label")
	}
	return response.NoContent(c)
}
