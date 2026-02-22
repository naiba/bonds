package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type AuthHandler struct {
	authService    *services.AuthService
	settingService *services.SystemSettingService
}

func NewAuthHandler(authService *services.AuthService, settingService *services.SystemSettingService) *AuthHandler {
	return &AuthHandler{authService: authService, settingService: settingService}
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Create a new account with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Registration details"
//	@Success		201		{object}	response.APIResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		409		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	locale := middleware.GetLocale(c)
	result, err := h.authService.Register(req, locale)
	if err != nil {
		if errors.Is(err, services.ErrEmailExists) {
			return response.Conflict(c, "err.email_already_exists")
		}
		if errors.Is(err, services.ErrRegistrationDisabled) {
			return response.Forbidden(c, "err.registration_disabled")
		}
		return response.InternalError(c, "err.failed_to_register")
	}

	return response.Created(c, result)
}

// Login godoc
//
//	@Summary		Log in
//	@Description	Authenticate with email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Login credentials"
//	@Success		200		{object}	response.APIResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	if h.settingService != nil && !h.settingService.GetBool("auth.password.enabled", true) {
		return response.Forbidden(c, "err.password_auth_disabled")
	}

	result, err := h.authService.Login(req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return response.Unauthorized(c, "err.invalid_email_or_password")
		}
		if errors.Is(err, services.ErrUserDisabled) {
			return response.Forbidden(c, "err.user_account_disabled")
		}
		return response.InternalError(c, "err.failed_to_login")
	}

	return response.OK(c, result)
}

// Refresh godoc
//
//	@Summary		Refresh JWT token
//	@Description	Obtain a new JWT token using the current valid token
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.AuthResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "err.invalid_token")
	}

	result, err := h.authService.RefreshToken(claims)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.Unauthorized(c, "err.user_not_found")
		}
		if errors.Is(err, services.ErrUserDisabled) {
			return response.Forbidden(c, "err.user_account_disabled")
		}
		return response.InternalError(c, "err.failed_to_refresh_token")
	}

	return response.OK(c, result)
}

// Me godoc
//
//	@Summary		Get current user
//	@Description	Return the authenticated user's profile
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.UserResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/auth/me [get]
func (h *AuthHandler) Me(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	user, err := h.authService.GetCurrentUser(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_user")
	}

	return response.OK(c, user)
}

// VerifyEmail godoc
//
//	@Summary		Verify email address
//	@Description	Verify user's email with token from verification email
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.VerifyEmailRequest	true	"Verification token"
//	@Success		200		{object}	response.APIResponse{data=dto.VerifyEmailResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Router			/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req dto.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	user, err := h.authService.VerifyEmail(req.Token)
	if err != nil {
		if errors.Is(err, services.ErrInvalidEmailVerificationToken) {
			return response.NotFound(c, "err.invalid_verification_token")
		}
		if errors.Is(err, services.ErrEmailAlreadyVerified) {
			return response.BadRequest(c, "err.email_already_verified", nil)
		}
		return response.InternalError(c, "err.failed_to_verify_email")
	}
	return response.OK(c, dto.VerifyEmailResponse{
		Message: "Email verified successfully",
		User:    *user,
	})
}

// ResendVerification godoc
//
//	@Summary		Resend verification email
//	@Description	Resend email verification link
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Router			/auth/resend-verification [post]
func (h *AuthHandler) ResendVerification(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.authService.ResendVerification(userID); err != nil {
		if errors.Is(err, services.ErrEmailAlreadyVerified) {
			return response.BadRequest(c, "err.email_already_verified", nil)
		}
		return response.InternalError(c, "err.failed_to_resend_verification")
	}
	return response.OK(c, map[string]string{"message": "Verification email sent"})
}
