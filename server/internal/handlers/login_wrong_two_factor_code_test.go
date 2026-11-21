package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
)

// The OAuth binding page submits the 2FA code together with the password. A
// mistyped code is the user's input, not a server failure.
func TestLogin_WrongTwoFactorCodeIsRejectedAsBadRequest(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "two-factor-login@example.com")
	if err := ts.db.Model(&models.User{}).Where("email = ?", "two-factor-login@example.com").Updates(map[string]interface{}{
		"two_factor_secret": "JBSWY3DPEHPK3PXP", "two_factor_confirmed_at": time.Now(),
	}).Error; err != nil {
		t.Fatalf("switch on 2FA: %v", err)
	}

	rec := ts.doRequest(http.MethodPost, "/api/auth/login",
		`{"email":"two-factor-login@example.com","password":"password123","totp_code":"000000"}`, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("a wrong 2FA code produced a successful login")
	}
}
