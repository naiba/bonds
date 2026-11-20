package dav

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func davRequestWithAccountPassword(t *testing.T, options ...AuthOption) *httptest.ResponseRecorder {
	t.Helper()
	db := testutil.SetupTestDB(t)
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	hashedStr := string(hashed)
	if err := db.Create(&models.User{AccountID: "test-account", Email: "dav-policy@example.com", Password: &hashedStr}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := BasicAuthMiddleware(db, options...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("PROPFIND", "/dav/", nil)
	req.SetBasicAuth("dav-policy@example.com", "password123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// Turning password login off is how an instance enforces SSO. DAV Basic auth
// with the account password is a password login too; accepting it would leave
// the very credential the administrator disabled fully usable.
func TestBasicAuth_RejectsAccountPasswordWhenPasswordLoginIsDisabled(t *testing.T) {
	rec := davRequestWithAccountPassword(t, WithPasswordAuthPolicy(func() bool { return false }))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Personal Access Token") {
		t.Fatalf("body = %q, want it to point the user to a Personal Access Token", rec.Body.String())
	}
}

func TestBasicAuth_AcceptsAccountPasswordWhenPasswordLoginIsEnabled(t *testing.T) {
	for name, options := range map[string][]AuthOption{
		"policy says enabled": {WithPasswordAuthPolicy(func() bool { return true })},
		"no policy given":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			if rec := davRequestWithAccountPassword(t, options...); rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
