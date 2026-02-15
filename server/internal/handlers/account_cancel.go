package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type AccountCancelHandler struct {
	accountCancelService *services.AccountCancelService
}

func NewAccountCancelHandler(svc *services.AccountCancelService) *AccountCancelHandler {
	return &AccountCancelHandler{accountCancelService: svc}
}

func (h *AccountCancelHandler) Cancel(c echo.Context) error {
	userID := middleware.GetUserID(c)
	accountID := middleware.GetAccountID(c)
	var req dto.CancelAccountRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.accountCancelService.Cancel(userID, accountID, req.Password); err != nil {
		if errors.Is(err, services.ErrPasswordMismatch) {
			return response.BadRequest(c, "err.password_mismatch", nil)
		}
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_cancel_account")
	}
	return response.NoContent(c)
}
