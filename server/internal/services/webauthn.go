package services

import (
	"errors"
	"net/http"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrWebAuthnNotConfigured      = errors.New("webauthn is not configured")
	ErrWebAuthnUserNotFound       = errors.New("webauthn user not found")
	ErrWebAuthnSessionNotFound    = errors.New("webauthn session not found")
	ErrWebAuthnNoCredentials      = errors.New("user has no webauthn credentials")
	ErrWebAuthnCredentialNotFound = errors.New("webauthn credential not found")
)

type WebAuthnService struct {
	db       *gorm.DB
	webauthn *webauthn.WebAuthn
	sessions sync.Map // map[string]*webauthn.SessionData
	settings *SystemSettingService
}

// NewWebAuthnService creates a new WebAuthnService.
// Always returns a non-nil service. If RPID is empty, the service is created
// in disabled mode (webauthn field is nil) — use IsEnabled() to check.
func NewWebAuthnService(db *gorm.DB, cfg *config.WebAuthnConfig) (*WebAuthnService, error) {
	svc := &WebAuthnService{db: db}

	if cfg.RPID == "" {
		return svc, nil
	}

	wconfig := &webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return svc, err
	}
	svc.webauthn = w

	return svc, nil
}

func (s *WebAuthnService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

// IsEnabled returns true if WebAuthn is configured (either via constructor or DB settings).
func (s *WebAuthnService) IsEnabled() bool {
	if s == nil {
		return false
	}
	if s.webauthn != nil {
		return true
	}
	// Check DB settings — may have been configured after startup
	if s.settings != nil {
		rpID := s.settings.GetWithDefault("webauthn.rp_id", "")
		return rpID != ""
	}
	return false
}

func (s *WebAuthnService) ReloadConfig() error {
	if s.settings == nil {
		return nil
	}
	rpID := s.settings.GetWithDefault("webauthn.rp_id", "")
	if rpID == "" {
		return nil
	}
	displayName := s.settings.GetWithDefault("webauthn.rp_display_name", "Bonds")
	originsStr := s.settings.GetWithDefault("webauthn.rp_origins", "")
	var origins []string
	if originsStr != "" {
		for _, o := range splitComma(originsStr) {
			if o != "" {
				origins = append(origins, o)
			}
		}
	}

	wconfig := &webauthn.Config{
		RPDisplayName: displayName,
		RPID:          rpID,
		RPOrigins:     origins,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return err
	}
	s.webauthn = w
	return nil
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func (s *WebAuthnService) BeginRegistration(userID string) (*protocol.CredentialCreation, error) {
	if !s.IsEnabled() {
		return nil, ErrWebAuthnNotConfigured
	}
	user, creds, err := s.loadUserWithCredentials(userID)
	if err != nil {
		return nil, err
	}

	wUser := newWebAuthnUser(user, creds)
	options, session, err := s.webauthn.BeginRegistration(wUser,
		webauthn.WithExclusions(wUser.CredentialExcludeList()),
	)
	if err != nil {
		return nil, err
	}

	s.sessions.Store("reg_"+userID, session)
	return options, nil
}

func (s *WebAuthnService) FinishRegistration(userID string, r *http.Request) error {
	if !s.IsEnabled() {
		return ErrWebAuthnNotConfigured
	}
	user, creds, err := s.loadUserWithCredentials(userID)
	if err != nil {
		return err
	}

	sessionVal, ok := s.sessions.LoadAndDelete("reg_" + userID)
	if !ok {
		return ErrWebAuthnSessionNotFound
	}
	session := sessionVal.(*webauthn.SessionData)

	wUser := newWebAuthnUser(user, creds)
	credential, err := s.webauthn.FinishRegistration(wUser, *session, r)
	if err != nil {
		return err
	}

	cred := models.WebAuthnCredential{
		UserID:          userID,
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		Name:            "Security Key",
	}
	return s.db.Create(&cred).Error
}

func (s *WebAuthnService) BeginLogin(email string) (*protocol.CredentialAssertion, error) {
	if !s.IsEnabled() {
		return nil, ErrWebAuthnNotConfigured
	}
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebAuthnUserNotFound
		}
		return nil, err
	}

	var creds []models.WebAuthnCredential
	if err := s.db.Where("user_id = ?", user.ID).Find(&creds).Error; err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return nil, ErrWebAuthnNoCredentials
	}

	wUser := newWebAuthnUser(&user, creds)
	options, session, err := s.webauthn.BeginLogin(wUser)
	if err != nil {
		return nil, err
	}

	s.sessions.Store("login_"+email, session)
	return options, nil
}

func (s *WebAuthnService) FinishLogin(email string, r *http.Request) (string, error) {
	if !s.IsEnabled() {
		return "", ErrWebAuthnNotConfigured
	}
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWebAuthnUserNotFound
		}
		return "", err
	}

	var creds []models.WebAuthnCredential
	if err := s.db.Where("user_id = ?", user.ID).Find(&creds).Error; err != nil {
		return "", err
	}

	sessionVal, ok := s.sessions.LoadAndDelete("login_" + email)
	if !ok {
		return "", ErrWebAuthnSessionNotFound
	}
	session := sessionVal.(*webauthn.SessionData)

	wUser := newWebAuthnUser(&user, creds)
	credential, err := s.webauthn.FinishLogin(wUser, *session, r)
	if err != nil {
		return "", err
	}

	// Update sign count
	for _, c := range creds {
		if string(c.CredentialID) == string(credential.ID) {
			s.db.Model(&c).Update("sign_count", credential.Authenticator.SignCount)
			break
		}
	}

	return user.ID, nil
}

func (s *WebAuthnService) ListCredentials(userID string) ([]dto.WebAuthnCredentialResponse, error) {
	var creds []models.WebAuthnCredential
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&creds).Error; err != nil {
		return nil, err
	}

	result := make([]dto.WebAuthnCredentialResponse, len(creds))
	for i, c := range creds {
		result[i] = dto.WebAuthnCredentialResponse{
			ID:        c.ID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt,
		}
	}
	return result, nil
}

func (s *WebAuthnService) DeleteCredential(id uint, userID string) error {
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.WebAuthnCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWebAuthnCredentialNotFound
	}
	return nil
}

func (s *WebAuthnService) loadUserWithCredentials(userID string) (*models.User, []models.WebAuthnCredential, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrWebAuthnUserNotFound
		}
		return nil, nil, err
	}

	var creds []models.WebAuthnCredential
	if err := s.db.Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		return nil, nil, err
	}

	return &user, creds, nil
}
