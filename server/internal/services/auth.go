package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists                   = errors.New("email already exists")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrUserNotFound                  = errors.New("user not found")
	ErrEmailNotVerified              = errors.New("email not verified")
	ErrInvalidEmailVerificationToken = errors.New("invalid email verification token")
	ErrEmailAlreadyVerified          = errors.New("email already verified")
	ErrRegistrationDisabled          = errors.New("registration is disabled")
	ErrPasswordAuthDisabled          = errors.New("password authentication is disabled")
)

type AuthService struct {
	db       *gorm.DB
	cfg      *config.JWTConfig
	mailer   Mailer
	settings *SystemSettingService
}

func NewAuthService(db *gorm.DB, cfg *config.JWTConfig) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) SetMailer(mailer Mailer) {
	s.mailer = mailer
}

func (s *AuthService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

func (s *AuthService) isEmailVerificationRequired() bool {
	if s.settings == nil {
		return false
	}
	if !s.settings.GetBool("auth.require_email_verification", true) {
		return false
	}
	smtpHost := s.settings.GetWithDefault("smtp.host", "")
	return smtpHost != ""
}

func (s *AuthService) sendVerificationEmail(user *models.User) {
	if s.mailer == nil || s.settings == nil {
		return
	}
	token := uuid.New().String()
	user.EmailVerificationToken = &token
	s.db.Model(user).Update("email_verification_token", token)

	appURL := s.settings.GetWithDefault("app.url", "http://localhost:8080")
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", appURL, token)
	subject := "Verify your email address"
	body := fmt.Sprintf(
		`<h2>Verify your email</h2>
<p>Please click the link below to verify your email address:</p>
<p><a href="%s">Verify Email</a></p>`,
		verifyLink,
	)
	if err := s.mailer.Send(user.Email, subject, body); err != nil {
		fmt.Printf("[AuthService] Failed to send verification email to %s: %v\n", user.Email, err)
	}
}

func (s *AuthService) Register(req dto.RegisterRequest, locale string) (*dto.AuthResponse, error) {
	// Check if registration is enabled (first user always allowed)
	var userCount int64
	s.db.Model(&models.User{}).Count(&userCount)
	isFirstUser := userCount == 0
	if !isFirstUser && s.settings != nil && !s.settings.GetBool("registration.enabled", true) {
		return nil, ErrRegistrationDisabled
	}

	var existing models.User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	hashedStr := string(hashedPassword)
	account := models.Account{}
	user := models.User{
		FirstName:               &req.FirstName,
		LastName:                &req.LastName,
		Email:                   req.Email,
		Password:                &hashedStr,
		IsAccountAdministrator:  true,
		IsInstanceAdministrator: isFirstUser,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		user.AccountID = account.ID
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return models.SeedAccountDefaults(tx, account.ID, user.ID, req.Email, locale)
	})
	if err != nil {
		return nil, err
	}

	if isFirstUser || !s.isEmailVerificationRequired() {
		now := time.Now()
		user.EmailVerifiedAt = &now
		s.db.Model(&user).Update("email_verified_at", now)
	} else {
		s.sendVerificationEmail(&user)
	}

	return s.generateAuthResponse(&user)
}

func (s *AuthService) Login(req dto.LoginRequest) (*dto.AuthResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Disabled {
		return nil, ErrUserDisabled
	}

	if user.Password == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.TwoFactorConfirmedAt != nil {
		if req.TOTPCode == "" {
			tempResp, err := s.generateTempAuthResponse(&user)
			if err != nil {
				return nil, err
			}
			return tempResp, nil
		}
		twoFactorSvc := NewTwoFactorService(s.db)
		valid, err := twoFactorSvc.Validate(user.ID, req.TOTPCode)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidTOTPCode
		}
	}

	return s.generateAuthResponse(&user)
}

func (s *AuthService) generateTempAuthResponse(user *models.User) (*dto.AuthResponse, error) {
	expiresAt := time.Now().Add(5 * time.Minute)
	claims := &middleware.JWTClaims{
		UserID:           user.ID,
		AccountID:        user.AccountID,
		Email:            user.Email,
		IsAdmin:          user.IsAccountAdministrator,
		IsInstanceAdmin:  user.IsInstanceAdministrator,
		TwoFactorPending: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		RequiresTwoFactor: true,
		TempToken:         tokenString,
		User:              *toUserResponse(user),
	}, nil
}

func (s *AuthService) RefreshToken(claims *middleware.JWTClaims) (*dto.AuthResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.Disabled {
		return nil, ErrUserDisabled
	}
	return s.generateAuthResponse(&user)
}

func (s *AuthService) GetCurrentUser(userID string) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUserResponse(&user), nil
}

func (s *AuthService) VerifyEmail(token string) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.Where("email_verification_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidEmailVerificationToken
		}
		return nil, err
	}
	if user.EmailVerifiedAt != nil {
		return nil, ErrEmailAlreadyVerified
	}
	now := time.Now()
	s.db.Model(&user).Updates(map[string]interface{}{
		"email_verified_at":        now,
		"email_verification_token": nil,
	})
	user.EmailVerifiedAt = &now
	return toUserResponse(&user), nil
}

func (s *AuthService) ResendVerification(userID string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return ErrUserNotFound
	}
	if user.EmailVerifiedAt != nil {
		return ErrEmailAlreadyVerified
	}
	s.sendVerificationEmail(&user)
	return nil
}

func (s *AuthService) generateAuthResponse(user *models.User) (*dto.AuthResponse, error) {
	expiresAt := time.Now().Add(time.Duration(s.cfg.ExpiryHrs) * time.Hour)
	claims := &middleware.JWTClaims{
		UserID:          user.ID,
		AccountID:       user.AccountID,
		Email:           user.Email,
		IsAdmin:         user.IsAccountAdministrator,
		IsInstanceAdmin: user.IsInstanceAdministrator,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User:      *toUserResponse(user),
	}, nil
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toUserResponse(user *models.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:                      user.ID,
		AccountID:               user.AccountID,
		FirstName:               ptrToStr(user.FirstName),
		LastName:                ptrToStr(user.LastName),
		Email:                   user.Email,
		IsAdmin:                 user.IsAccountAdministrator,
		IsInstanceAdministrator: user.IsInstanceAdministrator,
		EmailVerifiedAt:         user.EmailVerifiedAt,
		CreatedAt:               user.CreatedAt,
	}
}
