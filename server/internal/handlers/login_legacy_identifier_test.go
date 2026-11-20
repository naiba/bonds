package handlers_test

import (
	"net/http"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

// Registration did not check the email format before the validator learned the
// "email" rule, so accounts whose login identifier is not an RFC address can
// exist. Format rules belong on requests that introduce a new address; applying
// them to login would lock those accounts out.
func TestLogin_AcceptsLegacyIdentifierThatIsNotAnEmailAddress(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "legacy-login@example.com")
	if err := ts.db.Model(&models.User{}).Where("email = ?", "legacy-login@example.com").Update("email", "legacyadmin").Error; err != nil {
		t.Fatalf("rewrite identifier: %v", err)
	}

	rec := ts.doRequest(http.MethodPost, "/api/auth/login", `{"email":"legacyadmin","password":"password123"}`, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("legacy identifier login: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !parseResponse(t, rec).Success {
		t.Fatal("login response not marked successful")
	}
}
