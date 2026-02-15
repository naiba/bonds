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

type PostTemplateSectionHandler struct {
	svc *services.PostTemplateSectionService
}

func NewPostTemplateSectionHandler(svc *services.PostTemplateSectionService) *PostTemplateSectionHandler {
	return &PostTemplateSectionHandler{svc: svc}
}

func (h *PostTemplateSectionHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	var req dto.CreatePostTemplateSectionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	section, err := h.svc.Create(accountID, uint(templateID), req)
	if err != nil {
		if errors.Is(err, services.ErrPostTemplateNotFound) {
			return response.NotFound(c, "err.post_template_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_section")
	}
	return response.Created(c, section)
}

func (h *PostTemplateSectionHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	sectionID, err := strconv.ParseUint(c.Param("sectionId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_section_id", nil)
	}
	var req dto.UpdatePostTemplateSectionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	section, err := h.svc.Update(accountID, uint(templateID), uint(sectionID), req)
	if err != nil {
		if errors.Is(err, services.ErrPostTemplateNotFound) {
			return response.NotFound(c, "err.post_template_not_found")
		}
		if errors.Is(err, services.ErrPostTemplateSectionNotFound) {
			return response.NotFound(c, "err.section_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_section")
	}
	return response.OK(c, section)
}

func (h *PostTemplateSectionHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	sectionID, err := strconv.ParseUint(c.Param("sectionId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_section_id", nil)
	}
	if err := h.svc.Delete(accountID, uint(templateID), uint(sectionID)); err != nil {
		if errors.Is(err, services.ErrPostTemplateNotFound) {
			return response.NotFound(c, "err.post_template_not_found")
		}
		if errors.Is(err, services.ErrPostTemplateSectionNotFound) {
			return response.NotFound(c, "err.section_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_section")
	}
	return response.NoContent(c)
}

func (h *PostTemplateSectionHandler) UpdatePosition(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	sectionID, err := strconv.ParseUint(c.Param("sectionId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_section_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.svc.UpdatePosition(accountID, uint(templateID), uint(sectionID), req.Position); err != nil {
		if errors.Is(err, services.ErrPostTemplateNotFound) {
			return response.NotFound(c, "err.post_template_not_found")
		}
		if errors.Is(err, services.ErrPostTemplateSectionNotFound) {
			return response.NotFound(c, "err.section_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_section_position")
	}
	return response.NoContent(c)
}

type GroupTypeRoleHandler struct {
	svc *services.GroupTypeRoleService
}

func NewGroupTypeRoleHandler(svc *services.GroupTypeRoleService) *GroupTypeRoleHandler {
	return &GroupTypeRoleHandler{svc: svc}
}

func (h *GroupTypeRoleHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_type_id", nil)
	}
	var req dto.CreateGroupTypeRoleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	role, err := h.svc.Create(accountID, uint(groupTypeID), req)
	if err != nil {
		if errors.Is(err, services.ErrGroupTypeNotFound) {
			return response.NotFound(c, "err.group_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_role")
	}
	return response.Created(c, role)
}

func (h *GroupTypeRoleHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_type_id", nil)
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_role_id", nil)
	}
	var req dto.UpdateGroupTypeRoleRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	role, err := h.svc.Update(accountID, uint(groupTypeID), uint(roleID), req)
	if err != nil {
		if errors.Is(err, services.ErrGroupTypeNotFound) {
			return response.NotFound(c, "err.group_type_not_found")
		}
		if errors.Is(err, services.ErrGroupTypeRoleNotFound) {
			return response.NotFound(c, "err.role_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_role")
	}
	return response.OK(c, role)
}

func (h *GroupTypeRoleHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_type_id", nil)
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_role_id", nil)
	}
	if err := h.svc.Delete(accountID, uint(groupTypeID), uint(roleID)); err != nil {
		if errors.Is(err, services.ErrGroupTypeNotFound) {
			return response.NotFound(c, "err.group_type_not_found")
		}
		if errors.Is(err, services.ErrGroupTypeRoleNotFound) {
			return response.NotFound(c, "err.role_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_role")
	}
	return response.NoContent(c)
}

func (h *GroupTypeRoleHandler) UpdatePosition(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_type_id", nil)
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_role_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.svc.UpdatePosition(accountID, uint(groupTypeID), uint(roleID), req.Position); err != nil {
		if errors.Is(err, services.ErrGroupTypeNotFound) {
			return response.NotFound(c, "err.group_type_not_found")
		}
		if errors.Is(err, services.ErrGroupTypeRoleNotFound) {
			return response.NotFound(c, "err.role_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_role_position")
	}
	return response.NoContent(c)
}

type RelationshipTypeHandler struct {
	svc *services.RelationshipTypeService
}

func NewRelationshipTypeHandler(svc *services.RelationshipTypeService) *RelationshipTypeHandler {
	return &RelationshipTypeHandler{svc: svc}
}

func (h *RelationshipTypeHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_group_type_id", nil)
	}
	var req dto.CreateRelationshipTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	rt, err := h.svc.Create(accountID, uint(groupTypeID), req)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipGroupTypeNotFound) {
			return response.NotFound(c, "err.relationship_group_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_relationship_type")
	}
	return response.Created(c, rt)
}

func (h *RelationshipTypeHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_group_type_id", nil)
	}
	typeID, err := strconv.ParseUint(c.Param("typeId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_type_id", nil)
	}
	var req dto.UpdateRelationshipTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	rt, err := h.svc.Update(accountID, uint(groupTypeID), uint(typeID), req)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipGroupTypeNotFound) {
			return response.NotFound(c, "err.relationship_group_type_not_found")
		}
		if errors.Is(err, services.ErrRelationshipTypeNotFound) {
			return response.NotFound(c, "err.relationship_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_relationship_type")
	}
	return response.OK(c, rt)
}

func (h *RelationshipTypeHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_group_type_id", nil)
	}
	typeID, err := strconv.ParseUint(c.Param("typeId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_type_id", nil)
	}
	if err := h.svc.Delete(accountID, uint(groupTypeID), uint(typeID)); err != nil {
		if errors.Is(err, services.ErrRelationshipGroupTypeNotFound) {
			return response.NotFound(c, "err.relationship_group_type_not_found")
		}
		if errors.Is(err, services.ErrRelationshipTypeNotFound) {
			return response.NotFound(c, "err.relationship_type_not_found")
		}
		if errors.Is(err, services.ErrRelationshipTypeCannotBeDeleted) {
			return response.BadRequest(c, "err.relationship_type_cannot_be_deleted", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_relationship_type")
	}
	return response.NoContent(c)
}

type CallReasonHandler struct {
	svc *services.CallReasonService
}

func NewCallReasonHandler(svc *services.CallReasonService) *CallReasonHandler {
	return &CallReasonHandler{svc: svc}
}

func (h *CallReasonHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	callReasonTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_type_id", nil)
	}
	var req dto.CreateCallReasonRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	cr, err := h.svc.Create(accountID, uint(callReasonTypeID), req)
	if err != nil {
		if errors.Is(err, services.ErrCallReasonTypeNotFound) {
			return response.NotFound(c, "err.call_reason_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_call_reason")
	}
	return response.Created(c, cr)
}

func (h *CallReasonHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	callReasonTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_type_id", nil)
	}
	reasonID, err := strconv.ParseUint(c.Param("reasonId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_id", nil)
	}
	var req dto.UpdateCallReasonRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	cr, err := h.svc.Update(accountID, uint(callReasonTypeID), uint(reasonID), req)
	if err != nil {
		if errors.Is(err, services.ErrCallReasonTypeNotFound) {
			return response.NotFound(c, "err.call_reason_type_not_found")
		}
		if errors.Is(err, services.ErrCallReasonNotFound) {
			return response.NotFound(c, "err.call_reason_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_call_reason")
	}
	return response.OK(c, cr)
}

func (h *CallReasonHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	callReasonTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_type_id", nil)
	}
	reasonID, err := strconv.ParseUint(c.Param("reasonId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_id", nil)
	}
	if err := h.svc.Delete(accountID, uint(callReasonTypeID), uint(reasonID)); err != nil {
		if errors.Is(err, services.ErrCallReasonTypeNotFound) {
			return response.NotFound(c, "err.call_reason_type_not_found")
		}
		if errors.Is(err, services.ErrCallReasonNotFound) {
			return response.NotFound(c, "err.call_reason_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_call_reason")
	}
	return response.NoContent(c)
}
