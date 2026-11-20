package services

import (
	"errors"
	"strings"
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
	// EmailVerified 记录身份提供方是否声明该邮箱已验证。只有为 true 时，
	// 这个邮箱才能被当作账号归属的证据（自动绑定、免验证注册）。
	EmailVerified bool
}

type OAuthService struct {
	db   *gorm.DB
	jwt  *config.JWTConfig
	auth *AuthService
}

// SetAuthService 注入应用里那个已经配置好系统设置和邮件发送的 AuthService。
// OAuth 注册必须复用它：临时 new 出来的 AuthService 没有设置项，会无视
// “关闭注册”开关，也不会要求邮箱验证。
func (s *OAuthService) SetAuthService(auth *AuthService) {
	s.auth = auth
}

func (s *OAuthService) authService() *AuthService {
	if s.auth != nil {
		return s.auth
	}
	return NewAuthService(s.db, s.jwt)
}

func NewOAuthService(db *gorm.DB, jwt *config.JWTConfig) *OAuthService {
	return &OAuthService{db: db, jwt: jwt}
}

// FindOrCreateUser looks up a user by OAuth provider+providerUserID.
// If found, returns auth. If not, checks by email and auto-links.
// If no matching email exists, returns ErrOAuthAccountNotLinked instead of
// auto-creating an account — the caller must redirect the user to the
// account-binding flow (login existing account or register new one).
//
// 按邮箱自动绑定只在 emailVerified 为 true 时进行（判定规则见 EmailVerifiedByProvider）。
// 有的提供方（例如 Discord）允许用任意未验证的邮箱注册：不看这个标志的话，攻击者
// 在提供方用受害者的邮箱注册一个号，就能直接登录受害者的账号，密码和 2FA 都被绕过。
// 未验证时退回绑定流程，由用户登录已有账号来证明归属。
func (s *OAuthService) FindOrCreateUser(provider, providerUserID, email string, emailVerified bool, name, locale string) (*dto.AuthResponse, *OAuthLinkInfo, error) {
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

	linkInfo := &OAuthLinkInfo{
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		Name:           name,
		EmailVerified:  emailVerified,
	}
	if !emailVerified {
		return nil, linkInfo, ErrOAuthAccountNotLinked
	}

	var existingUser models.User
	err = s.db.Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).First(&existingUser).Error
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
	EmailVerified  bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

func (s *OAuthService) GenerateLinkToken(info *OAuthLinkInfo) (string, error) {
	claims := &oauthLinkClaims{
		Provider:       info.Provider,
		ProviderUserID: info.ProviderUserID,
		Email:          info.Email,
		Name:           info.Name,
		EmailVerified:  info.EmailVerified,
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
		EmailVerified:  claims.EmailVerified,
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

	authResp, err := s.authService().Register(dto.RegisterRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	}, locale)
	if err != nil {
		return nil, err
	}

	// 只有“提供方声明已验证”且“注册用的就是这个邮箱”时才能免验证。
	// 注册邮箱来自请求体，由调用者任填：无条件标记为已验证的话，任何人都能拿一个
	// OAuth 身份给别人的邮箱地址盖上“已验证”。其余情况走普通的邮箱验证流程。
	if info.EmailVerified && strings.EqualFold(strings.TrimSpace(req.Email), strings.TrimSpace(info.Email)) {
		now := time.Now()
		if err := s.db.Model(&models.User{}).Where("id = ?", authResp.User.ID).
			Updates(map[string]interface{}{"email_verified_at": now, "email_verification_token": nil}).Error; err != nil {
			return nil, err
		}
	}

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
	return s.authService().generateAuthResponse(user)
}

// EmailVerifiedByProvider 判断能否把提供方返回的邮箱当作账号归属的证据。
// rawData 是 goth 从提供方 userinfo / ID token 取回的原始字段。
//
// 只有提供方明确声明“未验证”时才返回 false：
// OIDC 的 email_verified、Google userinfo v2 的 verified_email、Discord 的 verified
// 为 false，或 GitLab 的 confirmed_at 为空。
//
// 提供方根本不下发这类字段时返回 true，保持原有的自动绑定行为。不少 OIDC 服务
// 默认不带 email_verified，一刀切按未验证处理会让这些部署的单点登录无法再自动
// 绑定已有账号（关闭密码登录时用户甚至无法自行绑定）。代价是：对这类提供方，
// 邮箱是否可信仍取决于提供方自身，接入允许自填未验证邮箱的 OIDC 服务前要先确认这一点。
func (s *OAuthService) EmailVerifiedByProvider(providerName string, rawData map[string]interface{}) bool {
	var provider models.OAuthProvider
	providerType := providerName
	if err := s.db.Where("name = ?", providerName).First(&provider).Error; err == nil {
		providerType = provider.Type
	}
	return oauthEmailVerified(providerType, rawData)
}

func oauthEmailVerified(providerType string, rawData map[string]interface{}) bool {
	if providerType == "gitlab" {
		if confirmedAt, present := rawData["confirmed_at"]; present {
			confirmed, _ := confirmedAt.(string)
			return confirmed != ""
		}
	}
	explicitlyUnverified := false
	for _, claim := range []string{"email_verified", "verified_email", "verified"} {
		switch verified := rawData[claim].(type) {
		case bool:
			if verified {
				return true
			}
			explicitlyUnverified = true
		case string:
			if strings.EqualFold(verified, "true") {
				return true
			}
			if strings.EqualFold(verified, "false") {
				explicitlyUnverified = true
			}
		}
	}
	return !explicitlyUnverified
}
