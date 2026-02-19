package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type GroupHandler struct {
	groupService *services.GroupService
}

func NewGroupHandler(groupService *services.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// Create godoc
//
//	@Summary		Create a group
//	@Description	Create a new group in the vault
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			request		body		dto.CreateGroupRequest	true	"Create group request"
//	@Success		201			{object}	response.APIResponse{data=dto.GroupResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/groups [post]
func (h *GroupHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	group, err := h.groupService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_group")
	}
	return response.Created(c, group)
}

// List godoc
//
//	@Summary		List groups
//	@Description	Return all groups for a vault
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.GroupResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/groups [get]
func (h *GroupHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	groups, err := h.groupService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_groups")
	}
	return response.OK(c, groups)
}

// Get godoc
//
//	@Summary		Get a group
//	@Description	Return a single group by ID
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Group ID"
//	@Success		200			{object}	response.APIResponse{data=dto.GroupResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/groups/{id} [get]
func (h *GroupHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_id", nil)
	}
	group, err := h.groupService.Get(uint(id), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrGroupNotFound) {
			return response.NotFound(c, "err.group_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_group")
	}
	return response.OK(c, group)
}

// ListContactGroups godoc
//
//	@Summary		List contact groups
//	@Description	Return all groups a contact belongs to
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.GroupResponse}
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/groups [get]
func (h *GroupHandler) ListContactGroups(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	groups, err := h.groupService.ListByContact(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_contact_groups")
	}
	return response.OK(c, groups)
}

// AddContactToGroup godoc
//
//	@Summary		Add contact to group
//	@Description	Add a contact to a group
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			request		body		dto.AddContactToGroupRequest	true	"Add contact to group request"
//	@Success		201			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/groups [post]
func (h *GroupHandler) AddContactToGroup(c echo.Context) error {
	contactID := c.Param("contact_id")
	var req dto.AddContactToGroupRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.groupService.AddContactToGroup(contactID, req); err != nil {
		return response.InternalError(c, "err.failed_to_add_contact_to_group")
	}
	return response.Created(c, map[string]string{"status": "ok"})
}

// RemoveContactFromGroup godoc
//
//	@Summary		Remove contact from group
//	@Description	Remove a contact from a group
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Group ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/groups/{id} [delete]
func (h *GroupHandler) RemoveContactFromGroup(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_id", nil)
	}
	if err := h.groupService.RemoveContactFromGroup(contactID, uint(id)); err != nil {
		return response.InternalError(c, "err.failed_to_remove_contact_from_group")
	}
	return response.NoContent(c)
}

// Update godoc
//
//	@Summary		Update a group
//	@Description	Update a group by ID
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			id			path		integer					true	"Group ID"
//	@Param			request		body		dto.UpdateGroupRequest	true	"Update group request"
//	@Success		200			{object}	response.APIResponse{data=dto.GroupResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/groups/{id} [put]
func (h *GroupHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_id", nil)
	}
	var req dto.UpdateGroupRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	group, err := h.groupService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrGroupNotFound) {
			return response.NotFound(c, "err.group_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_group")
	}
	return response.OK(c, group)
}

// Delete godoc
//
//	@Summary		Delete a group
//	@Description	Delete a group by ID
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Group ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/groups/{id} [delete]
func (h *GroupHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_id", nil)
	}
	if err := h.groupService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrGroupNotFound) {
			return response.NotFound(c, "err.group_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_group")
	}
	return response.NoContent(c)
}
