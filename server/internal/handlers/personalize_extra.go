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

// List godoc
//
//	@Summary		List post template sections
//	@Description	Return all sections for a post template
//	@Tags			post-template-sections
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Post Template ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.PostTemplateSectionResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/post-templates/{id}/sections [get]
func (h *PostTemplateSectionHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	sections, err := h.svc.List(accountID, uint(templateID))
	if err != nil {
		if errors.Is(err, services.ErrPostTemplateNotFound) {
			return response.NotFound(c, "err.post_template_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_sections")
	}
	return response.OK(c, sections)
}

// Create godoc
//
//	@Summary		Create a post template section
//	@Description	Create a new section for a post template
//	@Tags			post-template-sections
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer									true	"Post Template ID"
//	@Param			request	body		dto.CreatePostTemplateSectionRequest		true	"Section details"
//	@Success		201		{object}	response.APIResponse{data=dto.PostTemplateSectionResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/post-templates/{id}/sections [post]
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

// Update godoc
//
//	@Summary		Update a post template section
//	@Description	Update an existing section of a post template
//	@Tags			post-template-sections
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		integer									true	"Post Template ID"
//	@Param			sectionId	path		integer									true	"Section ID"
//	@Param			request		body		dto.UpdatePostTemplateSectionRequest		true	"Section details"
//	@Success		200			{object}	response.APIResponse{data=dto.PostTemplateSectionResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/post-templates/{id}/sections/{sectionId} [put]
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

// Delete godoc
//
//	@Summary		Delete a post template section
//	@Description	Delete a section from a post template
//	@Tags			post-template-sections
//	@Security		BearerAuth
//	@Param			id			path	integer	true	"Post Template ID"
//	@Param			sectionId	path	integer	true	"Section ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/post-templates/{id}/sections/{sectionId} [delete]
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

// UpdatePosition godoc
//
//	@Summary		Update post template section position
//	@Description	Update the position of a section within a post template
//	@Tags			post-template-sections
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id			path	integer					true	"Post Template ID"
//	@Param			sectionId	path	integer					true	"Section ID"
//	@Param			request		body	dto.UpdatePositionRequest	true	"Position"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/post-templates/{id}/sections/{sectionId}/position [post]
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

// List godoc
//
//	@Summary		List group type roles
//	@Description	Return all roles for a group type
//	@Tags			group-type-roles
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Group Type ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.GroupTypeRoleResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/group-types/{id}/roles [get]
func (h *GroupTypeRoleHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_group_type_id", nil)
	}
	roles, err := h.svc.List(accountID, uint(groupTypeID))
	if err != nil {
		if errors.Is(err, services.ErrGroupTypeNotFound) {
			return response.NotFound(c, "err.group_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_roles")
	}
	return response.OK(c, roles)
}

// Create godoc
//
//	@Summary		Create a group type role
//	@Description	Create a new role for a group type
//	@Tags			group-type-roles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer							true	"Group Type ID"
//	@Param			request	body		dto.CreateGroupTypeRoleRequest	true	"Role details"
//	@Success		201		{object}	response.APIResponse{data=dto.GroupTypeRoleResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/group-types/{id}/roles [post]
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

// Update godoc
//
//	@Summary		Update a group type role
//	@Description	Update an existing role of a group type
//	@Tags			group-type-roles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer							true	"Group Type ID"
//	@Param			roleId	path		integer							true	"Role ID"
//	@Param			request	body		dto.UpdateGroupTypeRoleRequest	true	"Role details"
//	@Success		200		{object}	response.APIResponse{data=dto.GroupTypeRoleResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/group-types/{id}/roles/{roleId} [put]
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

// Delete godoc
//
//	@Summary		Delete a group type role
//	@Description	Delete a role from a group type
//	@Tags			group-type-roles
//	@Security		BearerAuth
//	@Param			id		path	integer	true	"Group Type ID"
//	@Param			roleId	path	integer	true	"Role ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/group-types/{id}/roles/{roleId} [delete]
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

// UpdatePosition godoc
//
//	@Summary		Update group type role position
//	@Description	Update the position of a role within a group type
//	@Tags			group-type-roles
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id		path	integer					true	"Group Type ID"
//	@Param			roleId	path	integer					true	"Role ID"
//	@Param			request	body	dto.UpdatePositionRequest	true	"Position"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/group-types/{id}/roles/{roleId}/position [post]
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

// ListAll godoc
//
//	@Summary		List all relationship types across all groups
//	@Description	Return all relationship types with group names for the account, used for grouped select in relationship forms
//	@Tags			relationship-types
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.RelationshipTypeWithGroupResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/relationship-types/all [get]
func (h *RelationshipTypeHandler) ListAll(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	types, err := h.svc.ListAll(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_relationship_types")
	}
	return response.OK(c, types)
}

// List godoc
//
//	@Summary		List relationship types
//	@Description	Return all relationship types for a group type
//	@Tags			relationship-types
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Relationship Group Type ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.RelationshipTypeResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/relationship-types/{id}/types [get]
func (h *RelationshipTypeHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	groupTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_group_type_id", nil)
	}
	types, err := h.svc.List(accountID, uint(groupTypeID))
	if err != nil {
		if errors.Is(err, services.ErrRelationshipGroupTypeNotFound) {
			return response.NotFound(c, "err.relationship_group_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_relationship_types")
	}
	return response.OK(c, types)
}

// Create godoc
//
//	@Summary		Create a relationship type
//	@Description	Create a new relationship type under a group type
//	@Tags			relationship-types
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer								true	"Relationship Group Type ID"
//	@Param			request	body		dto.CreateRelationshipTypeRequest	true	"Relationship type details"
//	@Success		201		{object}	response.APIResponse{data=dto.RelationshipTypeResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/relationship-types/{id}/types [post]
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

// Update godoc
//
//	@Summary		Update a relationship type
//	@Description	Update an existing relationship type
//	@Tags			relationship-types
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer								true	"Relationship Group Type ID"
//	@Param			typeId	path		integer								true	"Relationship Type ID"
//	@Param			request	body		dto.UpdateRelationshipTypeRequest	true	"Relationship type details"
//	@Success		200		{object}	response.APIResponse{data=dto.RelationshipTypeResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/relationship-types/{id}/types/{typeId} [put]
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

// Delete godoc
//
//	@Summary		Delete a relationship type
//	@Description	Delete a relationship type from a group type
//	@Tags			relationship-types
//	@Security		BearerAuth
//	@Param			id		path	integer	true	"Relationship Group Type ID"
//	@Param			typeId	path	integer	true	"Relationship Type ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/relationship-types/{id}/types/{typeId} [delete]
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

// List godoc
//
//	@Summary		List call reasons
//	@Description	Return all call reasons for a call reason type
//	@Tags			call-reasons
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Call Reason Type ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.CallReasonResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/call-reasons/{id}/reasons [get]
func (h *CallReasonHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	callReasonTypeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_reason_type_id", nil)
	}
	reasons, err := h.svc.List(accountID, uint(callReasonTypeID))
	if err != nil {
		if errors.Is(err, services.ErrCallReasonTypeNotFound) {
			return response.NotFound(c, "err.call_reason_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_call_reasons")
	}
	return response.OK(c, reasons)
}

// Create godoc
//
//	@Summary		Create a call reason
//	@Description	Create a new call reason under a call reason type
//	@Tags			call-reasons
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer						true	"Call Reason Type ID"
//	@Param			request	body		dto.CreateCallReasonRequest	true	"Call reason details"
//	@Success		201		{object}	response.APIResponse{data=dto.CallReasonResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/call-reasons/{id}/reasons [post]
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

// Update godoc
//
//	@Summary		Update a call reason
//	@Description	Update an existing call reason
//	@Tags			call-reasons
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		integer						true	"Call Reason Type ID"
//	@Param			reasonId	path		integer						true	"Call Reason ID"
//	@Param			request		body		dto.UpdateCallReasonRequest	true	"Call reason details"
//	@Success		200			{object}	response.APIResponse{data=dto.CallReasonResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/call-reasons/{id}/reasons/{reasonId} [put]
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

// Delete godoc
//
//	@Summary		Delete a call reason
//	@Description	Delete a call reason from a call reason type
//	@Tags			call-reasons
//	@Security		BearerAuth
//	@Param			id			path	integer	true	"Call Reason Type ID"
//	@Param			reasonId	path	integer	true	"Call Reason ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/call-reasons/{id}/reasons/{reasonId} [delete]
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
