package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PetHandler struct {
	petService *services.PetService
}

func NewPetHandler(petService *services.PetService) *PetHandler {
	return &PetHandler{petService: petService}
}

// List godoc
//
//	@Summary		List pets for a contact
//	@Description	Return all pets belonging to a contact
//	@Tags			pets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.PetResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/pets [get]
func (h *PetHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	pets, err := h.petService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_pets")
	}
	return response.OK(c, pets)
}

// Create godoc
//
//	@Summary		Create a pet
//	@Description	Create a new pet for a contact
//	@Tags			pets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreatePetRequest	true	"Pet details"
//	@Success		201			{object}	response.APIResponse{data=dto.PetResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/pets [post]
func (h *PetHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreatePetRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	pet, err := h.petService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_pet")
	}
	return response.Created(c, pet)
}

// Update godoc
//
//	@Summary		Update a pet
//	@Description	Update an existing pet
//	@Tags			pets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Pet ID"
//	@Param			request		body		dto.UpdatePetRequest	true	"Pet details"
//	@Success		200			{object}	response.APIResponse{data=dto.PetResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/pets/{id} [put]
func (h *PetHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_pet_id", nil)
	}

	var req dto.UpdatePetRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	pet, err := h.petService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrPetNotFound) {
			return response.NotFound(c, "err.pet_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_pet")
	}
	return response.OK(c, pet)
}

// Delete godoc
//
//	@Summary		Delete a pet
//	@Description	Delete a pet from a contact
//	@Tags			pets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Pet ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/pets/{id} [delete]
func (h *PetHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_pet_id", nil)
	}

	if err := h.petService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrPetNotFound) {
			return response.NotFound(c, "err.pet_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_pet")
	}
	return response.NoContent(c)
}
