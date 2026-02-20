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

var _ dto.AuthResponse

type WebAuthnHandler struct {
	webauthnService *services.WebAuthnService
	authService     *services.AuthService
}

func NewWebAuthnHandler(webauthnService *services.WebAuthnService, authService *services.AuthService) *WebAuthnHandler {
	return &WebAuthnHandler{webauthnService: webauthnService, authService: authService}
}

// BeginRegistration godoc
//
//	@Summary		Begin WebAuthn registration
//	@Description	Start WebAuthn credential registration ceremony
//	@Tags			webauthn
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/webauthn/register/begin [post]
func (h *WebAuthnHandler) BeginRegistration(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	options, err := h.webauthnService.BeginRegistration(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_begin_webauthn_registration")
	}

	return response.OK(c, options)
}

// FinishRegistration godoc
//
//	@Summary		Finish WebAuthn registration
//	@Description	Complete WebAuthn credential registration ceremony
//	@Tags			webauthn
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/webauthn/register/finish [post]
func (h *WebAuthnHandler) FinishRegistration(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	if err := h.webauthnService.FinishRegistration(userID, c.Request()); err != nil {
		if errors.Is(err, services.ErrWebAuthnSessionNotFound) {
			return response.BadRequest(c, "err.webauthn_session_expired", nil)
		}
		return response.InternalError(c, "err.failed_to_finish_webauthn_registration")
	}

	return response.Created(c, map[string]string{"status": "ok"})
}

// BeginLogin godoc
//
//	@Summary		Begin WebAuthn login
//	@Description	Start WebAuthn login ceremony
//	@Tags			webauthn
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		422	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/auth/webauthn/login/begin [post]
func (h *WebAuthnHandler) BeginLogin(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.Email == "" {
		return response.ValidationError(c, map[string]string{"email": "email is required"})
	}

	options, err := h.webauthnService.BeginLogin(req.Email)
	if err != nil {
		if errors.Is(err, services.ErrWebAuthnUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		if errors.Is(err, services.ErrWebAuthnNoCredentials) {
			return response.BadRequest(c, "err.no_webauthn_credentials", nil)
		}
		return response.InternalError(c, "err.failed_to_begin_webauthn_login")
	}

	return response.OK(c, options)
}

// FinishLogin godoc
//
//	@Summary		Finish WebAuthn login
//	@Description	Complete WebAuthn login ceremony and return JWT
//	@Tags			webauthn
//	@Accept			json
//	@Produce		json
//	@Param			email	query	string	false	"User email"
//	@Success		200		{object}	response.APIResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/auth/webauthn/login/finish [post]
func (h *WebAuthnHandler) FinishLogin(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	var req struct {
		Email string `json:"email"`
	}
	// Try to get email from query parameter (WebAuthn flow)
	email := c.QueryParam("email")
	if email == "" {
		// Peek at JSON body — but WebAuthn FinishLogin needs the raw body for parsing.
		// We pass email via query param instead.
		if err := c.Bind(&req); err == nil && req.Email != "" {
			email = req.Email
		}
	}
	if email == "" {
		return response.ValidationError(c, map[string]string{"email": "email is required"})
	}

	userID, err := h.webauthnService.FinishLogin(email, c.Request())
	if err != nil {
		if errors.Is(err, services.ErrWebAuthnUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		if errors.Is(err, services.ErrWebAuthnSessionNotFound) {
			return response.BadRequest(c, "err.webauthn_session_expired", nil)
		}
		return response.InternalError(c, "err.failed_to_finish_webauthn_login")
	}

	authResp, err := h.authService.RefreshToken(&middleware.JWTClaims{UserID: userID})
	if err != nil {
		return response.InternalError(c, "err.failed_to_generate_token")
	}

	return response.OK(c, authResp)
}

// ListCredentials godoc
//
//	@Summary		List WebAuthn credentials
//	@Description	Return all WebAuthn credentials for the current user
//	@Tags			webauthn
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.WebAuthnCredentialResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/webauthn/credentials [get]
func (h *WebAuthnHandler) ListCredentials(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	creds, err := h.webauthnService.ListCredentials(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_webauthn_credentials")
	}

	return response.OK(c, creds)
}

// DeleteCredential godoc
//
//	@Summary		Delete a WebAuthn credential
//	@Description	Delete a WebAuthn credential by ID
//	@Tags			webauthn
//	@Security		BearerAuth
//	@Param			id	path	integer	true	"Credential ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/webauthn/credentials/{id} [delete]
func (h *WebAuthnHandler) DeleteCredential(c echo.Context) error {
	if !h.webauthnService.IsEnabled() {
		return response.BadRequest(c, "err.webauthn_not_configured", nil)
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		return response.Unauthorized(c, "err.invalid_user")
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return response.BadRequest(c, "err.invalid_id", nil)
	}

	if err := h.webauthnService.DeleteCredential(uint(id), userID); err != nil {
		if errors.Is(err, services.ErrWebAuthnCredentialNotFound) {
			return response.NotFound(c, "err.webauthn_credential_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_webauthn_credential")
	}

	return response.NoContent(c)
}
