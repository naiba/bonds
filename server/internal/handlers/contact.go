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

type ContactHandler struct {
	contactService *services.ContactService
}

func NewContactHandler(contactService *services.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

func (h *ContactHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	search := c.QueryParam("search")

	contacts, meta, err := h.contactService.ListContacts(vaultID, userID, page, perPage, search)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contacts")
	}
	return response.Paginated(c, contacts, meta)
}

func (h *ContactHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateContactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	contact, err := h.contactService.CreateContact(vaultID, userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_contact")
	}
	return response.Created(c, contact)
}

func (h *ContactHandler) Get(c echo.Context) error {
	contactID := c.Param("id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	contact, err := h.contactService.GetContact(contactID, userID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_contact")
	}
	return response.OK(c, contact)
}

func (h *ContactHandler) Update(c echo.Context) error {
	contactID := c.Param("id")
	vaultID := c.Param("vault_id")

	var req dto.UpdateContactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	contact, err := h.contactService.UpdateContact(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_contact")
	}
	return response.OK(c, contact)
}

func (h *ContactHandler) Delete(c echo.Context) error {
	contactID := c.Param("id")
	vaultID := c.Param("vault_id")
	if err := h.contactService.DeleteContact(contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_contact")
	}
	return response.NoContent(c)
}

func (h *ContactHandler) ToggleArchive(c echo.Context) error {
	contactID := c.Param("id")
	vaultID := c.Param("vault_id")
	contact, err := h.contactService.ToggleArchive(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_archive")
	}
	return response.OK(c, contact)
}

func (h *ContactHandler) ToggleFavorite(c echo.Context) error {
	contactID := c.Param("id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	contact, err := h.contactService.ToggleFavorite(contactID, userID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_favorite")
	}
	return response.OK(c, contact)
}
