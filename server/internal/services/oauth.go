package services

import (
	"errors"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrOAuthProviderNotFound = errors.New("oauth provider not found")
)

type OAuthService struct {
	db     *gorm.DB
	jwt    *config.JWTConfig
	appURL string
}

func NewOAuthService(db *gorm.DB, jwt *config.JWTConfig, appURL string) *OAuthService {
	return &OAuthService{db: db, jwt: jwt, appURL: appURL}
}

// FindOrCreateUser looks up a user by OAuth provider+providerUserID.
// If found, returns the user. If not, checks by email and either links or creates a new account.
func (s *OAuthService) FindOrCreateUser(provider, providerUserID, email, name string) (*dto.AuthResponse, error) {
	// Look up existing UserToken by Driver+DriverID
	var token models.UserToken
	err := s.db.Where("driver = ? AND driver_id = ?", provider, providerUserID).First(&token).Error
	if err == nil {
		// Found existing token — load user and return
		var user models.User
		if err := s.db.First(&user, "id = ?", token.UserID).Error; err != nil {
			return nil, err
		}
		return s.generateAuthResponse(&user)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// No existing token — check if user exists by email
	var existingUser models.User
	err = s.db.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		// User exists — link OAuth token to existing user
		newToken := models.UserToken{
			UserID:   existingUser.ID,
			Driver:   provider,
			DriverID: providerUserID,
			Format:   "oauth2",
			Email:    &email,
			Token:    "",
		}
		if err := s.db.Create(&newToken).Error; err != nil {
			return nil, err
		}
		return s.generateAuthResponse(&existingUser)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// No existing user — create new account + user + token
	randomPassword, err := bcrypt.GenerateFromPassword([]byte(providerUserID+email), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashedStr := string(randomPassword)

	account := models.Account{}
	user := models.User{
		Email:                  email,
		Password:               &hashedStr,
		IsAccountAdministrator: true,
	}

	// Parse name into first/last
	firstName, lastName := parseName(name)
	user.FirstName = &firstName
	user.LastName = &lastName

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		user.AccountID = account.ID
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := models.SeedAccountDefaults(tx, account.ID, user.ID, email); err != nil {
			return err
		}
		newToken := models.UserToken{
			UserID:   user.ID,
			Driver:   provider,
			DriverID: providerUserID,
			Format:   "oauth2",
			Email:    &email,
			Token:    "",
		}
		return tx.Create(&newToken).Error
	})
	if err != nil {
		return nil, err
	}

	return s.generateAuthResponse(&user)
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
