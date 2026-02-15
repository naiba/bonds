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

// List godoc
//
//	@Summary		List contacts
//	@Description	Return paginated contacts in a vault
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Param			search		query		string	false	"Search term"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ContactResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts [get]
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

// ListByLabel godoc
//
//	@Summary		List contacts by label
//	@Description	Return paginated contacts that have a specific label
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			labelId		path		integer	true	"Label ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ContactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/labels/{labelId} [get]
func (h *ContactHandler) ListByLabel(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	labelID, err := strconv.ParseUint(c.Param("labelId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_label_id", nil)
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	contacts, meta, err := h.contactService.ListContactsByLabel(vaultID, userID, uint(labelID), page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contacts")
	}
	return response.Paginated(c, contacts, meta)
}

// Create godoc
//
//	@Summary		Create a contact
//	@Description	Create a new contact in the vault
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.CreateContactRequest		true	"Contact details"
//	@Success		201			{object}	response.APIResponse{data=dto.ContactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts [post]
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

// Get godoc
//
//	@Summary		Get a contact
//	@Description	Return a single contact by ID
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{id} [get]
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

// Update godoc
//
//	@Summary		Update a contact
//	@Description	Update an existing contact's details
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		string						true	"Contact ID"
//	@Param			request		body		dto.UpdateContactRequest		true	"Contact details"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{id} [put]
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

// Delete godoc
//
//	@Summary		Delete a contact
//	@Description	Permanently delete a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	string	true	"Contact ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{id} [delete]
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

// ToggleArchive godoc
//
//	@Summary		Toggle contact archive status
//	@Description	Archive or unarchive a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{id}/archive [put]
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

// ToggleFavorite godoc
//
//	@Summary		Toggle contact favorite status
//	@Description	Mark or unmark a contact as favorite
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{id}/favorite [put]
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

// QuickSearch godoc
//
//	@Summary		Quick search contacts
//	@Description	Search contacts by name within a vault
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.ContactSearchRequest	true	"Search criteria"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ContactSearchItem}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/search/contacts [post]
func (h *ContactHandler) QuickSearch(c echo.Context) error {
	vaultID := c.Param("vault_id")

	var req dto.ContactSearchRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	results, err := h.contactService.QuickSearch(vaultID, req.SearchTerm)
	if err != nil {
		return response.InternalError(c, "err.failed_to_search_contacts")
	}
	return response.OK(c, results)
}
