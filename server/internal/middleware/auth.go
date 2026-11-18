package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

type JWTClaims struct {
	UserID           string `json:"user_id"`
	AccountID        string `json:"account_id"`
	Email            string `json:"email"`
	IsAdmin          bool   `json:"is_admin"`
	IsInstanceAdmin  bool   `json:"is_instance_admin"`
	AuthVersion      uint   `json:"auth_version,omitempty"`
	TwoFactorPending bool   `json:"two_factor_pending,omitempty"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	secret []byte
	db     *gorm.DB
}

func NewAuthMiddleware(secret string, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{secret: []byte(secret), db: db}
}

const patPrefix = "bonds_"

const ScopeCalendarRead = "calendar:read"

const (
	ctxPATScopes   = "pat_scopes"
	ctxIsScopedPAT = "is_scoped_pat"
)

func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var tokenString string

		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return response.Unauthorized(c, "err.invalid_authorization_format")
			}
		} else if qt := c.QueryParam("token"); qt != "" {
			tokenString = qt
		} else {
			return response.Unauthorized(c, "err.missing_authorization_header")
		}

		if strings.HasPrefix(tokenString, patPrefix) {
			return m.authenticateWithPAT(c, next, tokenString)
		}

		return m.authenticateWithJWT(c, next, tokenString)
	}
}

func (m *AuthMiddleware) authenticateWithJWT(c *echo.Context, next echo.HandlerFunc, tokenString string) error {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secret, nil
	})

	if err != nil || !token.Valid {
		return response.Unauthorized(c, "err.invalid_or_expired_token")
	}

	// Block access if 2FA verification is still pending.
	// The temp token is only valid for the /auth/2fa/verify endpoint.
	if claims.TwoFactorPending {
		return response.Forbidden(c, "err.two_factor_required")
	}

	user := &models.User{}
	if err := m.db.Select("id, account_id, email, disabled, auth_version, email_verified_at, is_account_administrator, is_instance_administrator").Where("id = ?", claims.UserID).First(user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Unauthorized(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.database_error")
	}
	if user.Disabled {
		return response.Forbidden(c, "err.user_account_disabled")
	}
	if user.AuthVersion != claims.AuthVersion {
		return response.Unauthorized(c, "err.invalid_or_expired_token")
	}
	admin, ok, err := accountMembershipAdmin(m.db, user, claims.AccountID)
	if err != nil {
		return response.InternalError(c, "err.database_error")
	}
	if !ok {
		return response.Unauthorized(c, "err.invalid_or_expired_token")
	}

	c.Set("user_id", claims.UserID)
	c.Set("account_id", claims.AccountID)
	c.Set("email", user.Email)
	// Database privileges are authoritative so promotions and revocations take effect before JWT expiry.
	c.Set("is_admin", admin)
	c.Set("is_instance_admin", user.IsInstanceAdministrator)
	c.Set("email_verified", user.EmailVerifiedAt != nil)
	c.Set("claims", claims)

	return next(c)
}

var (
	ErrPATInvalid      = errors.New("personal access token invalid or expired")
	ErrPATUserNotFound = errors.New("personal access token user not found")
	ErrPATUserDisabled = errors.New("personal access token user disabled")
)

// PATIdentity is the result of validating a personal access token.
type PATIdentity struct {
	Token   models.PersonalAccessToken
	User    models.User
	IsAdmin bool
}

// IsScoped reports whether the token is limited to specific scopes. Scoped
// tokens must only reach endpoints that opt in via RequireScope.
func (p *PATIdentity) IsScoped() bool {
	return strings.TrimSpace(p.Token.Scopes) != ""
}

// ResolvePersonalAccessToken is the single PAT validation path shared by REST
// and DAV. DAV used to keep its own copy that skipped the account-membership
// check and ignored scopes (GHSA-hc8x-cmcm-2jwf); keep every PAT consumer on
// this helper so the checks cannot drift apart again.
func ResolvePersonalAccessToken(db *gorm.DB, rawToken string) (*PATIdentity, error) {
	var pat models.PersonalAccessToken
	if err := db.Where("token_hash = ?", sha256Hash(rawToken)).First(&pat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPATInvalid
		}
		return nil, err
	}
	if pat.ExpiresAt != nil && time.Now().After(*pat.ExpiresAt) {
		return nil, ErrPATInvalid
	}

	var user models.User
	if err := db.Where("id = ?", pat.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPATUserNotFound
		}
		return nil, err
	}
	if user.Disabled {
		return nil, ErrPATUserDisabled
	}
	admin, ok, err := accountMembershipAdmin(db, &user, pat.AccountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPATInvalid
	}

	now := time.Now()
	db.Model(&pat).Update("last_used_at", &now)

	return &PATIdentity{Token: pat, User: user, IsAdmin: admin}, nil
}

func (m *AuthMiddleware) authenticateWithPAT(c *echo.Context, next echo.HandlerFunc, rawToken string) error {
	identity, err := ResolvePersonalAccessToken(m.db, rawToken)
	switch {
	case errors.Is(err, ErrPATInvalid):
		return response.Unauthorized(c, "err.invalid_or_expired_token")
	case errors.Is(err, ErrPATUserNotFound):
		return response.Unauthorized(c, "err.user_not_found")
	case errors.Is(err, ErrPATUserDisabled):
		return response.Forbidden(c, "err.user_account_disabled")
	case err != nil:
		return response.InternalError(c, "err.database_error")
	}

	c.Set("user_id", identity.User.ID)
	c.Set("account_id", identity.Token.AccountID)
	c.Set("email", identity.User.Email)
	c.Set("is_admin", identity.IsAdmin)
	c.Set("is_instance_admin", identity.User.IsInstanceAdministrator)
	c.Set("email_verified", identity.User.EmailVerifiedAt != nil)
	c.Set("auth_type", "pat")
	c.Set(ctxPATScopes, identity.Token.Scopes)
	c.Set(ctxIsScopedPAT, identity.IsScoped())

	return next(c)
}

func accountMembershipAdmin(db *gorm.DB, user *models.User, accountID string) (bool, bool, error) {
	if accountID == "" {
		return false, false, nil
	}
	var membership models.AccountMembership
	err := db.Where("account_id = ? AND user_id = ?", accountID, user.ID).First(&membership).Error
	if err == nil {
		// Keep the original user's home-account administrator bit authoritative
		// while old clients and account-management endpoints still use it.
		if accountID == user.AccountID {
			return user.IsAccountAdministrator, true, nil
		}
		return membership.IsAdmin, true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, false, err
	}
	return false, false, nil
}

func patHasScope(c *echo.Context, scope string) bool {
	raw, _ := c.Get(ctxPATScopes).(string)
	for _, s := range strings.Split(raw, ",") {
		if strings.TrimSpace(s) == scope {
			return true
		}
	}
	return false
}

func isScopedPAT(c *echo.Context) bool {
	v, _ := c.Get(ctxIsScopedPAT).(bool)
	return v
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (m *AuthMiddleware) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		isAdmin, ok := c.Get("is_admin").(bool)
		if !ok || !isAdmin {
			return response.Forbidden(c, "err.administrator_access_required")
		}
		return next(c)
	}
}

func (m *AuthMiddleware) RequireInstanceAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		isInstanceAdmin, ok := c.Get("is_instance_admin").(bool)
		if !ok || !isInstanceAdmin {
			return response.Forbidden(c, "err.instance_admin_access_required")
		}
		return next(c)
	}
}

func GetUserID(c *echo.Context) string {
	id, _ := c.Get("user_id").(string)
	return id
}

func GetAccountID(c *echo.Context) string {
	id, _ := c.Get("account_id").(string)
	return id
}

// GetVaultAccountID is only valid after the vault permission middleware.
// A cross-account vault invitation never grants access to account settings.
func GetVaultAccountID(c *echo.Context) string {
	id, _ := c.Get("vault_account_id").(string)
	return id
}

func GetClaims(c *echo.Context) *JWTClaims {
	claims, _ := c.Get("claims").(*JWTClaims)
	return claims
}

func ParseJWTClaims(tokenString string, secret []byte) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	if claims.TwoFactorPending {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func RequireEmailVerification(isRequired func() bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !isRequired() {
				return next(c)
			}
			verified, _ := c.Get("email_verified").(bool)
			if !verified {
				return response.Forbidden(c, "err.email_not_verified")
			}
			return next(c)
		}
	}
}

// DenyScopedPAT blocks scope-limited PATs from reaching endpoints that do not
// declare a scope. Full-access tokens (JWT or unscoped PAT) pass through. This
// is the default-deny half of the scope model: an endpoint must opt in via
// RequireScope, otherwise scoped tokens are rejected.
func DenyScopedPAT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if isScopedPAT(c) {
			return response.Forbidden(c, "err.insufficient_token_scope")
		}
		return next(c)
	}
}

// RequireScope allows an endpoint to be reached by a scope-limited PAT only if
// the token carries the given scope. Full-access tokens (JWT or unscoped PAT)
// always pass.
func RequireScope(scope string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !isScopedPAT(c) {
				return next(c)
			}
			if patHasScope(c, scope) {
				return next(c)
			}
			return response.Forbidden(c, "err.insufficient_token_scope")
		}
	}
}
