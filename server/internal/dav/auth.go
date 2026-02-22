package dav

import (
	"net/http"

	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BasicAuthMiddleware returns an HTTP middleware that validates Basic Auth
// credentials against the User model and sets user_id/account_id in context.
func BasicAuthMiddleware(db *gorm.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			if user.Disabled {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ctx := WithUserID(r.Context(), user.ID)
			ctx = WithAccountID(ctx, user.AccountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
