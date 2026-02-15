package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

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
