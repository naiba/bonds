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

type PersonalizeHandler struct {
	personalizeService *services.PersonalizeService
}

func NewPersonalizeHandler(personalizeService *services.PersonalizeService) *PersonalizeHandler {
	return &PersonalizeHandler{personalizeService: personalizeService}
}

// List godoc
//
//	@Summary		List personalize entities
//	@Description	Return all entities of the given type for the account
//	@Tags			personalize
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity	path		string	true	"Entity type"
//	@Success		200		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/{entity} [get]
func (h *PersonalizeHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	entity := c.Param("entity")

	if entity == "templates" {
		data, err := h.personalizeService.ListTemplates(accountID)
		if err != nil {
			return response.InternalError(c, "err.failed_to_list_templates")
		}
		return response.OK(c, data)
	}
	items, err := h.personalizeService.List(accountID, entity)
	if err != nil {
		if errors.Is(err, services.ErrUnknownEntityType) {
			return response.NotFound(c, "err.unknown_entity_type")
		}
		return response.InternalError(c, "err.failed_to_list_entities")
	}
	return response.OK(c, items)
}

// Create godoc
//
//	@Summary		Create a personalize entity
//	@Description	Create a new entity of the given type
//	@Tags			personalize
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity	path		string							true	"Entity type"
//	@Param			request	body		dto.PersonalizeEntityRequest		true	"Entity details"
//	@Success		201		{object}	response.APIResponse{data=dto.PersonalizeEntityResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/{entity} [post]
func (h *PersonalizeHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	entity := c.Param("entity")
	var req dto.PersonalizeEntityRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	item, err := h.personalizeService.Create(accountID, entity, req)
	if err != nil {
		if errors.Is(err, services.ErrUnknownEntityType) {
			return response.NotFound(c, "err.unknown_entity_type")
		}
		return response.InternalError(c, "err.failed_to_create_entity")
	}
	return response.Created(c, item)
}

// Update godoc
//
//	@Summary		Update a personalize entity
//	@Description	Update an existing entity of the given type
//	@Tags			personalize
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity	path		string							true	"Entity type"
//	@Param			id		path		integer							true	"Entity ID"
//	@Param			request	body		dto.PersonalizeEntityRequest		true	"Entity details"
//	@Success		200		{object}	response.APIResponse{data=dto.PersonalizeEntityResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/{entity}/{id} [put]
func (h *PersonalizeHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	entity := c.Param("entity")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_entity_id", nil)
	}
	var req dto.PersonalizeEntityRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	item, err := h.personalizeService.Update(accountID, entity, uint(id), req)
	if err != nil {
		if errors.Is(err, services.ErrUnknownEntityType) {
			return response.NotFound(c, "err.unknown_entity_type")
		}
		if errors.Is(err, services.ErrPersonalizeEntityNotFound) {
			return response.NotFound(c, "err.entity_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_entity")
	}
	return response.OK(c, item)
}

// Delete godoc
//
//	@Summary		Delete a personalize entity
//	@Description	Delete an entity of the given type
//	@Tags			personalize
//	@Security		BearerAuth
//	@Param			entity	path	string	true	"Entity type"
//	@Param			id		path	integer	true	"Entity ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/personalize/{entity}/{id} [delete]
func (h *PersonalizeHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	entity := c.Param("entity")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_entity_id", nil)
	}
	if err := h.personalizeService.Delete(accountID, entity, uint(id)); err != nil {
		if errors.Is(err, services.ErrUnknownEntityType) {
			return response.NotFound(c, "err.unknown_entity_type")
		}
		if errors.Is(err, services.ErrPersonalizeEntityNotFound) {
			return response.NotFound(c, "err.entity_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_entity")
	}
	return response.NoContent(c)
}
