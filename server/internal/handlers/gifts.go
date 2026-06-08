package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type GiftHandler struct {
	giftService *services.GiftService
}

func NewGiftHandler(giftService *services.GiftService) *GiftHandler {
	return &GiftHandler{giftService: giftService}
}

// List godoc
//
//	@Summary		List gifts for a contact
//	@Description	Return all gifts belonging to a contact
//	@Tags			gifts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.GiftResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/gifts [get]
func (h *GiftHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	gifts, err := h.giftService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_gifts")
	}
	return response.OK(c, gifts)
}

// Create godoc
//
//	@Summary		Create a gift
//	@Description	Create a new gift for a contact
//	@Tags			gifts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreateGiftRequest	true	"Gift details"
//	@Success		201			{object}	response.APIResponse{data=dto.GiftResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/gifts [post]
func (h *GiftHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateGiftRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	gift, err := h.giftService.Create(contactID, vaultID, req)
	if err != nil {
		return handleGiftServiceError(c, err, "err.failed_to_create_gift")
	}
	return response.Created(c, gift)
}

// Update godoc
//
//	@Summary		Update a gift
//	@Description	Update an existing gift
//	@Tags			gifts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Gift ID"
//	@Param			request		body		dto.UpdateGiftRequest	true	"Gift details"
//	@Success		200			{object}	response.APIResponse{data=dto.GiftResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/gifts/{id} [put]
func (h *GiftHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_gift_id", nil)
	}

	var req dto.UpdateGiftRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	gift, err := h.giftService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		return handleGiftServiceError(c, err, "err.failed_to_update_gift")
	}
	return response.OK(c, gift)
}

// Delete godoc
//
//	@Summary		Delete a gift
//	@Description	Delete a gift from a contact
//	@Tags			gifts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Gift ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/gifts/{id} [delete]
func (h *GiftHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_gift_id", nil)
	}

	if err := h.giftService.Delete(uint(id), contactID, vaultID); err != nil {
		return handleGiftServiceError(c, err, "err.failed_to_delete_gift")
	}
	return response.NoContent(c)
}

func handleGiftServiceError(c echo.Context, err error, fallback string) error {
	if errors.Is(err, services.ErrContactNotFound) {
		return response.NotFound(c, "err.contact_not_found")
	}
	if errors.Is(err, services.ErrGiftNotFound) {
		return response.NotFound(c, "err.gift_not_found")
	}
	if errors.Is(err, services.ErrGiftNameRequired) {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if errors.Is(err, services.ErrGiftOccasionNotFound) {
		return response.NotFound(c, "err.gift_occasion_not_found")
	}
	if errors.Is(err, services.ErrGiftStateNotFound) {
		return response.NotFound(c, "err.gift_state_not_found")
	}
	return response.InternalError(c, fallback)
}
