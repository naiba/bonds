package dav

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

			if user.Disabled {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			if strings.HasPrefix(password, "bonds_") {
				if !authenticateWithPAT(db, password, user.ID) {
					w.Header().Set("WWW-Authenticate", `Basic realm="Bonds DAV"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			} else {
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
			ctx = WithAccountID(ctx, user.AccountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticateWithPAT(db *gorm.DB, rawToken, userID string) bool {
	h := sha256.Sum256([]byte(rawToken))
	hash := hex.EncodeToString(h[:])

	var pat models.PersonalAccessToken
	if err := db.Where("token_hash = ? AND user_id = ?", hash, userID).First(&pat).Error; err != nil {
		return false
	}

	if pat.ExpiresAt != nil && time.Now().After(*pat.ExpiresAt) {
		return false
	}

	now := time.Now()
	db.Model(&pat).Update("last_used_at", &now)

	return true
}
