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

type InvitationHandler struct {
	invitationService *services.InvitationService
}

func NewInvitationHandler(invitationService *services.InvitationService) *InvitationHandler {
	return &InvitationHandler{invitationService: invitationService}
}

func (h *InvitationHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	invitations, err := h.invitationService.List(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_invitations")
	}
	return response.OK(c, invitations)
}

func (h *InvitationHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	userID := middleware.GetUserID(c)

	var req dto.CreateInvitationRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	invitation, err := h.invitationService.Create(accountID, userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return response.BadRequest(c, "err.user_already_exists", nil)
		}
		return response.InternalError(c, "err.failed_to_create_invitation")
	}
	return response.Created(c, invitation)
}

func (h *InvitationHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_invitation_id", nil)
	}

	if err := h.invitationService.Delete(uint(id), accountID); err != nil {
		if errors.Is(err, services.ErrInvitationNotFound) {
			return response.NotFound(c, "err.invitation_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_invitation")
	}
	return response.NoContent(c)
}

func (h *InvitationHandler) Accept(c echo.Context) error {
	var req dto.AcceptInvitationRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	invitation, err := h.invitationService.Accept(req)
	if err != nil {
		if errors.Is(err, services.ErrInvitationNotFound) {
			return response.NotFound(c, "err.invitation_not_found")
		}
		if errors.Is(err, services.ErrInvitationExpired) {
			return response.BadRequest(c, "err.invitation_expired", nil)
		}
		return response.InternalError(c, "err.failed_to_accept_invitation")
	}
	return response.OK(c, invitation)
}
