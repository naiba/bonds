package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.UserManagementResponse

// Get godoc
//
//	@Summary		Get a managed user
//	@Description	Return a user by ID
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	response.APIResponse{data=dto.UserManagementResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/users/{id} [get]
func (h *UserManagementHandler) Get(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	id := c.Param("id")
	user, err := h.userManagementService.Get(id, accountID)
	if err != nil {
		if errors.Is(err, services.ErrManagedUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_user")
	}
	return response.OK(c, user)
}
