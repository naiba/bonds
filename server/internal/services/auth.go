package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.JWTConfig
}

func NewAuthService(db *gorm.DB, cfg *config.JWTConfig) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Register(req dto.RegisterRequest, locale string) (*dto.AuthResponse, error) {
	var existing models.User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// First registered user becomes instance administrator
	var userCount int64
	s.db.Model(&models.User{}).Count(&userCount)
	isFirstUser := userCount == 0

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
		CreatedAt:               user.CreatedAt,
	}
}
