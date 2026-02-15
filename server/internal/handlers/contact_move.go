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
