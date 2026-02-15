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

func (h *GroupHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	groups, err := h.groupService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_groups")
	}
	return response.OK(c, groups)
}

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
