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
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	result, err := h.authService.Register(req)
	if err != nil {
		if errors.Is(err, services.ErrEmailExists) {
			return response.Conflict(c, "err.email_already_exists")
		}
		return response.InternalError(c, "err.failed_to_register")
	}

	return response.Created(c, result)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	result, err := h.authService.Login(req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return response.Unauthorized(c, "err.invalid_email_or_password")
		}
		return response.InternalError(c, "err.failed_to_login")
	}

	return response.OK(c, result)
}

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
		return response.InternalError(c, "err.failed_to_refresh_token")
	}

	return response.OK(c, result)
}

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
