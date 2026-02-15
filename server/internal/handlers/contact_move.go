package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactMoveHandler struct {
	contactMoveService *services.ContactMoveService
}

func NewContactMoveHandler(contactMoveService *services.ContactMoveService) *ContactMoveHandler {
	return &ContactMoveHandler{contactMoveService: contactMoveService}
}

// Move godoc
//
//	@Summary		Move a contact to another vault
//	@Description	Move a contact from one vault to another
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Source Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.MoveContactRequest	true	"Move details"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/move [post]
func (h *ContactMoveHandler) Move(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.MoveContactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	contact, err := h.contactMoveService.Move(contactID, vaultID, req.TargetVaultID, userID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrTargetVaultNotFound) {
			return response.NotFound(c, "err.target_vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_move_contact")
	}
	return response.OK(c, contact)
}
