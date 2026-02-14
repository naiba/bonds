package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type TwoFactorHandler struct {
	twoFactorService *services.TwoFactorService
}

func NewTwoFactorHandler(twoFactorService *services.TwoFactorService) *TwoFactorHandler {
	return &TwoFactorHandler{twoFactorService: twoFactorService}
}

func (h *TwoFactorHandler) Enable(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	result, err := h.twoFactorService.Enable(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_enable_2fa")
	}

	return response.OK(c, result)
}

func (h *TwoFactorHandler) Confirm(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	var req dto.TwoFactorVerifyRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	err := h.twoFactorService.Confirm(userID, req.Code)
	if err != nil {
		if errors.Is(err, services.ErrInvalidTOTPCode) {
			return response.BadRequest(c, "err.invalid_totp_code", nil)
		}
		if errors.Is(err, services.ErrTwoFactorNotSet) {
			return response.BadRequest(c, "err.two_factor_not_set", nil)
		}
		return response.InternalError(c, "err.failed_to_confirm_2fa")
	}

	return response.OK(c, map[string]bool{"confirmed": true})
}

func (h *TwoFactorHandler) Disable(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	var req dto.TwoFactorVerifyRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	err := h.twoFactorService.Disable(userID, req.Code)
	if err != nil {
		if errors.Is(err, services.ErrInvalidTOTPCode) {
			return response.BadRequest(c, "err.invalid_totp_code", nil)
		}
		if errors.Is(err, services.ErrTwoFactorNotSet) {
			return response.BadRequest(c, "err.two_factor_not_set", nil)
		}
		return response.InternalError(c, "err.failed_to_disable_2fa")
	}

	return response.OK(c, map[string]bool{"disabled": true})
}

func (h *TwoFactorHandler) Status(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	enabled, err := h.twoFactorService.IsEnabled(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_2fa_status")
	}

	return response.OK(c, dto.TwoFactorStatusResponse{Enabled: enabled})
}
