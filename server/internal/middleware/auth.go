package middleware

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/pkg/response"
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
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{secret: []byte(secret)}
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

		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Set("email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("is_instance_admin", claims.IsInstanceAdmin)
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
