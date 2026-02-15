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

// List godoc
//
//	@Summary		List invitations
//	@Description	Return all invitations for the account
//	@Tags			invitations
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.InvitationResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/invitations [get]
func (h *InvitationHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	invitations, err := h.invitationService.List(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_invitations")
	}
	return response.OK(c, invitations)
}

// Create godoc
//
//	@Summary		Create an invitation
//	@Description	Create a new invitation and send it via email
//	@Tags			invitations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateInvitationRequest	true	"Invitation details"
//	@Success		201		{object}	response.APIResponse{data=dto.InvitationResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/invitations [post]
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

// Delete godoc
//
//	@Summary		Delete an invitation
//	@Description	Delete an invitation by ID
//	@Tags			invitations
//	@Security		BearerAuth
//	@Param			id	path	integer	true	"Invitation ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/invitations/{id} [delete]
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

// Accept godoc
//
//	@Summary		Accept an invitation
//	@Description	Accept an invitation and create a new user account
//	@Tags			invitations
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.AcceptInvitationRequest	true	"Acceptance details"
//	@Success		200		{object}	response.APIResponse{data=dto.InvitationResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/invitations/accept [post]
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
