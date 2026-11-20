package dav

import (
	"net/http"
	"strings"

	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthOption 调整 DAV Basic 认证的行为。
type AuthOption func(*basicAuthPolicy)

type basicAuthPolicy struct {
	passwordAuthEnabled func() bool
}

// WithPasswordAuthPolicy 让 DAV 遵守实例的“密码登录”开关。enabled 每次请求都会被调用，
// 管理员修改设置后立即生效。
func WithPasswordAuthPolicy(enabled func() bool) AuthOption {
	return func(policy *basicAuthPolicy) { policy.passwordAuthEnabled = enabled }
}

func BasicAuthMiddleware(db *gorm.DB, options ...AuthOption) func(next http.Handler) http.Handler {
	policy := basicAuthPolicy{}
	for _, option := range options {
		option(&policy)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				// DAV clients discover capabilities with unauthenticated OPTIONS; real DAV methods remain behind Basic Auth below.
				next.ServeHTTP(w, r)
				return
			}

			email, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			var user models.User
			if err := db.Where("email = ?", email).First(&user).Error; err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if user.Disabled {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			accountID := user.AccountID
			if strings.HasPrefix(password, "bonds_") {
				identity, ok := authenticateWithPAT(db, password, user.ID)
				if !ok {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				// The PAT is bound to one account; DAV must act in that account, not the user's home account.
				accountID = identity.Token.AccountID
			} else {
				if user.TwoFactorConfirmedAt != nil {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "2FA enabled: use a Personal Access Token instead of password", http.StatusUnauthorized)
					return
				}
				// 用账号密码做 Basic 认证也是一次密码登录。实例关闭密码登录（通常是为了
				// 强制走 SSO）之后这里必须同样拒绝，否则被停用的凭据在 DAV 上照样能用。
				// 个人访问令牌不受影响。
				if policy.passwordAuthEnabled != nil && !policy.passwordAuthEnabled() {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "Password login is disabled: use a Personal Access Token instead of password", http.StatusUnauthorized)
					return
				}
				if user.Password == nil {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}

			ctx := WithUserID(r.Context(), user.ID)
			ctx = WithAccountID(ctx, accountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// authenticateWithPAT delegates to the shared REST validator so DAV gets the
// same expiry, disabled-user and account-membership checks. Scoped PATs are
// rejected outright: DAV exposes full read-write access to every address book
// and calendar, while scopes (calendar:read) are only meant for the .ics feed
// URL. Accepting them here let a leaked feed token read and rewrite all
// contacts (GHSA-hc8x-cmcm-2jwf).
func authenticateWithPAT(db *gorm.DB, rawToken, userID string) (*middleware.PATIdentity, bool) {
	identity, err := middleware.ResolvePersonalAccessToken(db, rawToken)
	if err != nil || identity.User.ID != userID || identity.IsScoped() {
		return nil, false
	}
	return identity, true
}
