package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrOAuthProviderNotFound = errors.New("oauth provider not found")
	ErrOAuthAccountNotLinked = errors.New("oauth account not linked to any existing account")
	ErrOAuthLinkTokenInvalid = errors.New("oauth link token is invalid or expired")
	ErrOAuthAlreadyLinked    = errors.New("oauth provider already linked to another account")
)

// OAuthLinkInfo holds OAuth provider info extracted from a link token.
// Used to pass OAuth context through the account-binding flow when a user
// authenticates via OAuth but has no matching email account.
type OAuthLinkInfo struct {
	Provider       string
	ProviderUserID string
	Email          string
	Name           string
}

type OAuthService struct {
	db       *gorm.DB
	jwt      *config.JWTConfig
	appURL   string
	settings *SystemSettingService
}

func NewOAuthService(db *gorm.DB, jwt *config.JWTConfig, appURL string) *OAuthService {
	return &OAuthService{db: db, jwt: jwt, appURL: appURL}
}

func (s *OAuthService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

func (s *OAuthService) getAppURL() string {
	if s.settings != nil {
		return s.settings.GetWithDefault("app.url", s.appURL)
	}
	return s.appURL
}

// FindOrCreateUser looks up a user by OAuth provider+providerUserID.
// If found, returns auth. If not, checks by email and auto-links.
// If no matching email exists, returns ErrOAuthAccountNotLinked instead of
// auto-creating an account — the caller must redirect the user to the
// account-binding flow (login existing account or register new one).
func (s *OAuthService) FindOrCreateUser(provider, providerUserID, email, name, locale string) (*dto.AuthResponse, *OAuthLinkInfo, error) {
	var token models.UserToken
	err := s.db.Where("driver = ? AND driver_id = ?", provider, providerUserID).First(&token).Error
	if err == nil {
		var user models.User
		if err := s.db.First(&user, "id = ?", token.UserID).Error; err != nil {
			return nil, nil, err
		}
		resp, err := s.generateAuthResponse(&user)
		return resp, nil, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	var existingUser models.User
	err = s.db.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		if existingUser.EmailVerifiedAt == nil {
			verifiedNow := time.Now()
			s.db.Model(&existingUser).Update("email_verified_at", verifiedNow)
			existingUser.EmailVerifiedAt = &verifiedNow
		}
		newToken := models.UserToken{
			UserID:   existingUser.ID,
			Driver:   provider,
			DriverID: providerUserID,
			Format:   "oauth2",
			Email:    &email,
			Token:    "",
		}
		if err := s.db.Create(&newToken).Error; err != nil {
			return nil, nil, err
		}
		resp, err := s.generateAuthResponse(&existingUser)
		return resp, nil, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	// No matching email — do NOT auto-create account.
	// Return link info so the handler can redirect to the binding flow.
	linkInfo := &OAuthLinkInfo{
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		Name:           name,
	}
	return nil, linkInfo, ErrOAuthAccountNotLinked
}

// SaveToken upserts an OAuth token for a user.
func (s *OAuthService) SaveToken(userID, provider, providerUserID, accessToken, refreshToken string, expiresIn int) error {
	var token models.UserToken
	err := s.db.Where("driver = ? AND driver_id = ?", provider, providerUserID).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			email := ""
			var expiresInUint *uint64
			if expiresIn > 0 {
				v := uint64(expiresIn)
				expiresInUint = &v
			}
			token = models.UserToken{
				UserID:    userID,
				Driver:    provider,
				DriverID:  providerUserID,
				Format:    "oauth2",
				Email:     &email,
				Token:     accessToken,
				ExpiresIn: expiresInUint,
			}
			if refreshToken != "" {
				token.RefreshToken = &refreshToken
			}
			return s.db.Create(&token).Error
		}
		return err
	}

	updates := map[string]interface{}{
		"token":   accessToken,
		"user_id": userID,
	}
	if refreshToken != "" {
		updates["refresh_token"] = refreshToken
	}
	if expiresIn > 0 {
		updates["expires_in"] = uint64(expiresIn)
	}
	return s.db.Model(&token).Updates(updates).Error
}

type oauthLinkClaims struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	jwt.RegisteredClaims
}

func (s *OAuthService) GenerateLinkToken(info *OAuthLinkInfo) (string, error) {
	claims := &oauthLinkClaims{
		Provider:       info.Provider,
		ProviderUserID: info.ProviderUserID,
		Email:          info.Email,
		Name:           info.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwt.Secret))
}

func (s *OAuthService) ParseLinkToken(linkToken string) (*OAuthLinkInfo, error) {
	claims := &oauthLinkClaims{}
	token, err := jwt.ParseWithClaims(linkToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwt.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrOAuthLinkTokenInvalid
	}
	return &OAuthLinkInfo{
		Provider:       claims.Provider,
		ProviderUserID: claims.ProviderUserID,
		Email:          claims.Email,
		Name:           claims.Name,
	}, nil
}

func (s *OAuthService) LinkOAuthToUser(linkToken, userID string) (*dto.AuthResponse, error) {
	info, err := s.ParseLinkToken(linkToken)
	if err != nil {
		return nil, err
	}

	// Prevent duplicate binding: if this OAuth identity is already linked to another user, reject
	var existing models.UserToken
	if err := s.db.Where("driver = ? AND driver_id = ?", info.Provider, info.ProviderUserID).First(&existing).Error; err == nil {
		if existing.UserID != userID {
			return nil, ErrOAuthAlreadyLinked
		}
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	newToken := models.UserToken{
		UserID:   userID,
		Driver:   info.Provider,
		DriverID: info.ProviderUserID,
		Format:   "oauth2",
		Email:    &info.Email,
		Token:    "",
	}
	if err := s.db.Create(&newToken).Error; err != nil {
		return nil, err
	}

	return s.generateAuthResponse(&user)
}

func (s *OAuthService) LinkOAuthAndRegister(linkToken string, req dto.OAuthLinkRegisterRequest, locale string) (*dto.AuthResponse, error) {
	info, err := s.ParseLinkToken(linkToken)
	if err != nil {
		return nil, err
	}

	var existing models.UserToken
	if err := s.db.Where("driver = ? AND driver_id = ?", info.Provider, info.ProviderUserID).First(&existing).Error; err == nil {
		return nil, ErrOAuthAlreadyLinked
	}

	authSvc := NewAuthService(s.db, s.jwt)
	authResp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	}, locale)
	if err != nil {
		return nil, err
	}

	// OAuth emails are trusted — auto-verify
	now := time.Now()
	s.db.Model(&models.User{}).Where("id = ?", authResp.User.ID).Update("email_verified_at", now)

	newToken := models.UserToken{
		UserID:   authResp.User.ID,
		Driver:   info.Provider,
		DriverID: info.ProviderUserID,
		Format:   "oauth2",
		Email:    &info.Email,
		Token:    "",
	}
	if err := s.db.Create(&newToken).Error; err != nil {
		return nil, err
	}

	return authResp, nil
}

func (s *OAuthService) generateAuthResponse(user *models.User) (*dto.AuthResponse, error) {
	authSvc := NewAuthService(s.db, s.jwt)
	return authSvc.generateAuthResponse(user)
}

func parseName(name string) (string, string) {
	if name == "" {
		return "", ""
	}
	for i, ch := range name {
		if ch == ' ' {
			return name[:i], name[i+1:]
		}
	}
	return name, ""
}
