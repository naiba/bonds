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

// Enable godoc
//
//	@Summary		Enable two-factor authentication
//	@Description	Generate TOTP secret and recovery codes for 2FA setup
//	@Tags			two-factor
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.TwoFactorSetupResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/2fa/enable [post]
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

// Confirm godoc
//
//	@Summary		Confirm two-factor authentication
//	@Description	Confirm 2FA setup by verifying a TOTP code
//	@Tags			two-factor
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.TwoFactorVerifyRequest	true	"TOTP code"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/2fa/confirm [post]
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

// Disable godoc
//
//	@Summary		Disable two-factor authentication
//	@Description	Disable 2FA by verifying a TOTP code
//	@Tags			two-factor
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.TwoFactorVerifyRequest	true	"TOTP code"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/2fa/disable [post]
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

// Status godoc
//
//	@Summary		Get two-factor authentication status
//	@Description	Check if 2FA is enabled for the current user
//	@Tags			two-factor
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.TwoFactorStatusResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/2fa/status [get]
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
