package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type UserManagementHandler struct {
	userManagementService *services.UserManagementService
}

func NewUserManagementHandler(svc *services.UserManagementService) *UserManagementHandler {
	return &UserManagementHandler{userManagementService: svc}
}

// List godoc
//
//	@Summary		List managed users
//	@Description	Return all users in the account
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.UserManagementResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/users [get]
func (h *UserManagementHandler) List(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	users, err := h.userManagementService.List(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_users")
	}
	return response.OK(c, users)
}

// Create godoc
//
//	@Summary		Create a managed user
//	@Description	Create a new user in the account
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateManagedUserRequest	true	"User details"
//	@Success		201		{object}	response.APIResponse{data=dto.UserManagementResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/users [post]
func (h *UserManagementHandler) Create(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	var req dto.CreateManagedUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	user, err := h.userManagementService.Create(accountID, req)
	if err != nil {
		if errors.Is(err, services.ErrEmailExists) {
			return response.BadRequest(c, "err.email_already_exists", nil)
		}
		return response.InternalError(c, "err.failed_to_create_user")
	}
	return response.Created(c, user)
}

// Update godoc
//
//	@Summary		Update a managed user
//	@Description	Update an existing user in the account
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"User ID"
//	@Param			request	body		dto.UpdateManagedUserRequest	true	"User details"
//	@Success		200		{object}	response.APIResponse{data=dto.UserManagementResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/users/{id} [put]
func (h *UserManagementHandler) Update(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	id := c.Param("id")
	var req dto.UpdateManagedUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	user, err := h.userManagementService.Update(id, accountID, req)
	if err != nil {
		if errors.Is(err, services.ErrManagedUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_user")
	}
	return response.OK(c, user)
}

// Delete godoc
//
//	@Summary		Delete a managed user
//	@Description	Delete a user from the account
//	@Tags			users
//	@Security		BearerAuth
//	@Param			id	path	string	true	"User ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/users/{id} [delete]
func (h *UserManagementHandler) Delete(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	userID := middleware.GetUserID(c)
	id := c.Param("id")
	if err := h.userManagementService.Delete(id, accountID, userID); err != nil {
		if errors.Is(err, services.ErrCannotDeleteSelf) {
			return response.BadRequest(c, "err.cannot_delete_self", nil)
		}
		if errors.Is(err, services.ErrManagedUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_user")
	}
	return response.NoContent(c)
}
