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
}

// NewWebAuthnService creates a new WebAuthnService.
// Returns nil, nil if RPID is empty (WebAuthn disabled).
func NewWebAuthnService(db *gorm.DB, cfg *config.WebAuthnConfig) (*WebAuthnService, error) {
	if cfg.RPID == "" {
		return nil, nil
	}

	wconfig := &webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, err
	}

	return &WebAuthnService{
		db:       db,
		webauthn: w,
	}, nil
}

func (s *WebAuthnService) BeginRegistration(userID string) (*protocol.CredentialCreation, error) {
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

// FinishLogin completes the WebAuthn login ceremony and returns the userID.
func (s *WebAuthnService) FinishLogin(email string, r *http.Request) (string, error) {
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
