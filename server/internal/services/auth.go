package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	userTimezone "github.com/naiba/bonds/internal/timezone"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists                   = errors.New("email already exists")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrUserNotFound                  = errors.New("user not found")
	ErrEmailNotVerified              = errors.New("email not verified")
	ErrInvalidTempToken              = errors.New("invalid or expired temp token")
	ErrTwoFactorNotPending           = errors.New("two factor not pending for this token")
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
	// Localize on user.Locale so accounts created from a non-English UI get
	// the verification mail in the language they actually read. Until
	// Register persists the requested locale onto user.Locale this falls
	// back to "" → English; the i18n fallback chain then guarantees a
	// readable email even if the bundle is missing the key for that locale.
	locale := user.Locale
	subject := i18n.T(locale, "email.verify.subject")
	body := i18n.Tt(locale, "email.verify.body", map[string]string{"link": verifyLink})
	if err := s.mailer.Send(user.Email, subject, body); err != nil {
		fmt.Printf("[AuthService] Failed to send verification email to %s: %v\n", user.Email, err)
	}
}

func (s *AuthService) Register(req dto.RegisterRequest, locale string) (*dto.AuthResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	// Check if registration is enabled (first user always allowed)
	// 计数失败必须中止注册：忽略错误时 userCount 保持 0，注册者会被当成首个用户，
	// 绕过“关闭注册”并拿到实例管理员。
	var userCount int64
	if err := s.db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return nil, err
	}
	isFirstUser := userCount == 0
	if !isFirstUser && s.settings != nil && !s.settings.GetBool("registration.enabled", true) {
		return nil, ErrRegistrationDisabled
	}

	var existing models.User
	if err := s.db.Where("LOWER(email) = ?", req.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	hashedStr := string(hashedPassword)
	defaultTimezone := userTimezone.Default
	account := models.Account{}
	user := models.User{
		FirstName:               &req.FirstName,
		LastName:                &req.LastName,
		Email:                   req.Email,
		Password:                &hashedStr,
		Timezone:                &defaultTimezone,
		Locale:                  locale,
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
		if err := tx.Create(&models.AccountMembership{
			AccountID: account.ID, UserID: user.ID, IsAdmin: true,
		}).Error; err != nil {
			return err
		}
		return models.SeedAccountDefaults(tx, account.ID, user.ID, req.Email, locale)
	})
	if err != nil {
		return nil, err
	}
	if isFirstUser {
		// Persist the bootstrap instance-admin bit explicitly because follow-up
		// auth reads reload the user from the database rather than this struct.
		if err := s.db.Model(&user).Update("is_instance_administrator", true).Error; err != nil {
			return nil, err
		}
		user.IsInstanceAdministrator = true
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
	if err := s.db.Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 账号不存在时也做一次等价的 bcrypt 比较，避免靠响应耗时分辨邮箱是否注册过。
			_ = bcrypt.CompareHashAndPassword(loginTimingEqualizerHash, []byte(req.Password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Password == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// “账号已禁用”必须放在密码校验之后：先于密码返回的话，任何人都能用它
	// 确认某个邮箱在本实例注册过。
	if user.Disabled {
		return nil, ErrUserDisabled
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
		AuthVersion:      user.AuthVersion,
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
	// WebAuthn login starts without an account-scoped JWT. Issue its initial
	// session for the user's home account; explicit account switches still
	// preserve their selected account in subsequent token refreshes.
	if claims.AccountID == "" {
		return s.generateAuthResponse(&user)
	}
	return s.generateAuthResponseForAccount(&user, claims.AccountID)
}

// VerifyTwoFactor validates a TOTP code against the temp_token issued during
// the 2FA-pending login flow. On success, returns a full JWT (no TwoFactorPending claim).
// Bug #78: this endpoint was missing — the temp_token had nowhere to be redeemed.
func (s *AuthService) VerifyTwoFactor(req dto.TwoFactorLoginVerifyRequest) (*dto.AuthResponse, error) {
	claims := &middleware.JWTClaims{}
	token, err := jwt.ParseWithClaims(req.TempToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidTempToken
	}
	if !claims.TwoFactorPending {
		return nil, ErrTwoFactorNotPending
	}

	twoFactorSvc := NewTwoFactorService(s.db)
	valid, err := twoFactorSvc.Validate(claims.UserID, req.Code)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidTOTPCode
	}

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
	return s.generateAuthResponseForAccount(user, user.AccountID)
}

func (s *AuthService) generateAuthResponseForAccount(user *models.User, accountID string) (*dto.AuthResponse, error) {
	var membership models.AccountMembership
	if err := s.db.Where("account_id = ? AND user_id = ?", accountID, user.ID).First(&membership).Error; err != nil {
		return nil, ErrUserNotFound
	}
	if accountID == user.AccountID {
		membership.IsAdmin = user.IsAccountAdministrator
	}
	expiresAt := time.Now().Add(time.Duration(s.cfg.ExpiryHrs) * time.Hour)
	claims := &middleware.JWTClaims{
		UserID:          user.ID,
		AccountID:       accountID,
		Email:           user.Email,
		IsAdmin:         membership.IsAdmin,
		IsInstanceAdmin: user.IsInstanceAdministrator,
		AuthVersion:     user.AuthVersion,
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
		User:      *userResponseForAccount(user, accountID, membership.IsAdmin),
	}, nil
}

func (s *AuthService) ListAccounts(userID string) ([]dto.AccountMembershipResponse, error) {
	var user models.User
	if err := s.db.Select("account_id, is_account_administrator").First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	var memberships []models.AccountMembership
	if err := s.db.Where("user_id = ?", userID).Order("created_at ASC").Find(&memberships).Error; err != nil {
		return nil, err
	}
	result := make([]dto.AccountMembershipResponse, 0, len(memberships))
	for _, membership := range memberships {
		admin := membership.IsAdmin
		if membership.AccountID == user.AccountID {
			admin = user.IsAccountAdministrator
		}
		result = append(result, dto.AccountMembershipResponse{
			AccountID: membership.AccountID, IsAdmin: admin, IsHome: membership.AccountID == user.AccountID,
		})
	}
	return result, nil
}

func (s *AuthService) SwitchAccount(userID, accountID string) (*dto.AuthResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return s.generateAuthResponseForAccount(&user, accountID)
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

func userResponseForAccount(user *models.User, accountID string, isAdmin bool) *dto.UserResponse {
	resp := toUserResponse(user)
	resp.AccountID = accountID
	resp.IsAdmin = isAdmin
	return resp
}

// loginTimingEqualizerHash 是一个固定口令的 bcrypt 哈希，只用于让“账号不存在”的
// 登录请求消耗和真实校验相近的时间。
var loginTimingEqualizerHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("bonds-login-timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}
	return hash
}()
