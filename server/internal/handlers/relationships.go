package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type RelationshipHandler struct {
	relationshipService *services.RelationshipService
}

func NewRelationshipHandler(relationshipService *services.RelationshipService) *RelationshipHandler {
	return &RelationshipHandler{relationshipService: relationshipService}
}

// List godoc
//
//	@Summary		List relationships for a contact
//	@Description	Return all relationships belonging to a contact
//	@Tags			relationships
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.RelationshipResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships [get]
func (h *RelationshipHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	relationships, err := h.relationshipService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_relationships")
	}
	return response.OK(c, relationships)
}

// Create godoc
//
//	@Summary		Create a relationship
//	@Description	Create a new relationship for a contact
//	@Tags			relationships
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			contact_id	path		string							true	"Contact ID"
//	@Param			request		body		dto.CreateRelationshipRequest	true	"Relationship details"
//	@Success		201			{object}	response.APIResponse{data=dto.RelationshipResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships [post]
func (h *RelationshipHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateRelationshipRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	relationship, err := h.relationshipService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_relationship")
	}
	return response.Created(c, relationship)
}

// Update godoc
//
//	@Summary		Update a relationship
//	@Description	Update an existing relationship
//	@Tags			relationships
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			contact_id	path		string							true	"Contact ID"
//	@Param			id			path		integer							true	"Relationship ID"
//	@Param			request		body		dto.UpdateRelationshipRequest	true	"Relationship details"
//	@Success		200			{object}	response.APIResponse{data=dto.RelationshipResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships/{id} [put]
func (h *RelationshipHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_id", nil)
	}

	var req dto.UpdateRelationshipRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	relationship, err := h.relationshipService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrRelationshipNotFound) {
			return response.NotFound(c, "err.relationship_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_relationship")
	}
	return response.OK(c, relationship)
}

// Delete godoc
//
//	@Summary		Delete a relationship
//	@Description	Delete a relationship from a contact
//	@Tags			relationships
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Relationship ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships/{id} [delete]
func (h *RelationshipHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_relationship_id", nil)
	}

	if err := h.relationshipService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrRelationshipNotFound) {
			return response.NotFound(c, "err.relationship_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_relationship")
	}
	return response.NoContent(c)
}

// GetContactGraph godoc
//
//	@Summary		Get contact relationship graph
//	@Description	Return nodes and edges for the contact's relationship network
//	@Tags			relationships
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactGraphResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships/graph [get]
func (h *RelationshipHandler) GetContactGraph(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	graph, err := h.relationshipService.GetContactGraph(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_graph")
	}
	return response.OK(c, graph)
}

// CalculateKinship godoc
//
//	@Summary		Calculate kinship degree between two contacts
//	@Description	Compute the shortest kinship path between two contacts using Dijkstra's algorithm
//	@Tags			relationships
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id			path		string	true	"Vault ID"
//	@Param			contact_id			path		string	true	"Contact ID"
//	@Param			related_contact_id	path		string	true	"Related Contact ID"
//	@Success		200					{object}	response.APIResponse{data=dto.KinshipResponse}
//	@Failure		401					{object}	response.APIResponse
//	@Failure		404					{object}	response.APIResponse
//	@Failure		500					{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/relationships/kinship/{related_contact_id} [get]
func (h *RelationshipHandler) CalculateKinship(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	relatedContactID := c.Param("related_contact_id")
	kinship, err := h.relationshipService.CalculateKinship(contactID, relatedContactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_calculate_kinship")
	}
	return response.OK(c, kinship)
}
