package middleware

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
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

func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
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
		if err := m.db.Select("disabled, email_verified_at").Where("id = ?", claims.UserID).First(user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return response.Unauthorized(c, "err.user_not_found")
			}
			return response.InternalError(c, "err.database_error")
		}
		if user.Disabled {
			return response.Forbidden(c, "err.user_account_disabled")
		}

		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Set("email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("is_instance_admin", claims.IsInstanceAdmin)
		c.Set("email_verified", user.EmailVerifiedAt != nil)
		c.Set("claims", claims)

		return next(c)
	}
}

func (m *AuthMiddleware) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		isAdmin, ok := c.Get("is_admin").(bool)
		if !ok || !isAdmin {
			return response.Forbidden(c, "err.administrator_access_required")
		}
		return next(c)
	}
}

func (m *AuthMiddleware) RequireInstanceAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		isInstanceAdmin, ok := c.Get("is_instance_admin").(bool)
		if !ok || !isInstanceAdmin {
			return response.Forbidden(c, "err.instance_admin_access_required")
		}
		return next(c)
	}
}

func GetUserID(c echo.Context) string {
	id, _ := c.Get("user_id").(string)
	return id
}

func GetAccountID(c echo.Context) string {
	id, _ := c.Get("account_id").(string)
	return id
}

func GetClaims(c echo.Context) *JWTClaims {
	claims, _ := c.Get("claims").(*JWTClaims)
	return claims
}

func RequireEmailVerification(isRequired func() bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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
