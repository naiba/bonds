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

func (h *PetHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	pets, err := h.petService.List(contactID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_pets")
	}
	return response.OK(c, pets)
}

func (h *PetHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")

	var req dto.CreatePetRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	pet, err := h.petService.Create(contactID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_pet")
	}
	return response.Created(c, pet)
}

func (h *PetHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
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

	pet, err := h.petService.Update(uint(id), contactID, req)
	if err != nil {
		if errors.Is(err, services.ErrPetNotFound) {
			return response.NotFound(c, "err.pet_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_pet")
	}
	return response.OK(c, pet)
}

func (h *PetHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_pet_id", nil)
	}

	if err := h.petService.Delete(uint(id), contactID); err != nil {
		if errors.Is(err, services.ErrPetNotFound) {
			return response.NotFound(c, "err.pet_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_pet")
	}
	return response.NoContent(c)
}
