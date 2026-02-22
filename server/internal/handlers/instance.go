package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.InstanceInfoResponse

type InstanceHandler struct {
	settingService  *services.SystemSettingService
	oauthService    *services.OAuthService
	webauthnService *services.WebAuthnService
	version         string
}

func NewInstanceHandler(
	settingService *services.SystemSettingService,
	oauthService *services.OAuthService,
	webauthnService *services.WebAuthnService,
	version string,
) *InstanceHandler {
	return &InstanceHandler{
		settingService:  settingService,
		oauthService:    oauthService,
		webauthnService: webauthnService,
		version:         version,
	}
}

// GetInfo godoc
//
//	@Summary		Get instance info
//	@Description	Get public instance information (version, enabled auth methods)
//	@Tags			instance
//	@Produce		json
//	@Success		200	{object}	response.APIResponse{data=dto.InstanceInfoResponse}
//	@Router			/instance/info [get]
func (h *InstanceHandler) GetInfo(c echo.Context) error {
	registrationEnabled := h.settingService.GetBool("registration.enabled", true)
	passwordAuthEnabled := h.settingService.GetBool("auth.password.enabled", true)
	requireEmailVerification := h.settingService.GetBool("auth.require_email_verification", false)
	smtpConfigured := h.settingService.GetWithDefault("smtp.host", "") != ""
	emailVerificationActive := requireEmailVerification && smtpConfigured
	appName := h.settingService.GetWithDefault("app.name", "Bonds")

	providers := h.oauthService.ListAvailableProviders()
	oauthNames := make([]string, len(providers))
	for i, p := range providers {
		if dn, ok := p["display_name"]; ok && dn != "" {
			oauthNames[i] = dn
		} else {
			oauthNames[i] = p["name"]
		}
	}

	webauthnEnabled := h.webauthnService.IsEnabled()

	info := dto.InstanceInfoResponse{
		Version:                  h.version,
		RegistrationEnabled:      registrationEnabled,
		PasswordAuthEnabled:      passwordAuthEnabled,
		OAuthProviders:           oauthNames,
		WebAuthnEnabled:          webauthnEnabled,
		AppName:                  appName,
		RequireEmailVerification: emailVerificationActive,
	}

	return response.OK(c, info)
}
