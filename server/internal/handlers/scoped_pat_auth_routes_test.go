package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func createScopedCalendarPAT(t *testing.T, ts *testServer, jwtToken string) string {
	t.Helper()
	rec := ts.doRequest(http.MethodPost, "/api/settings/tokens",
		`{"name":"Calendar feed","scopes":["calendar:read"]}`, jwtToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create scoped PAT: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(parseResponse(t, rec).Data, &created); err != nil {
		t.Fatalf("parse scoped PAT: %v", err)
	}
	if !strings.HasPrefix(created.Token, "bonds_") {
		t.Fatalf("unexpected PAT format %q", created.Token)
	}
	return created.Token
}

// A calendar:read PAT must not reach the /api/auth routes that sit outside the
// DenyScopedPAT-protected group: switch-account mints a full-access JWT and
// oauth/link binds a login identity, so either one escalates a leaked feed
// token into full account access.
func TestScopedPATCannotUseAuthSessionRoutes(t *testing.T) {
	ts := setupTestServer(t)
	jwtToken, auth := ts.registerTestUser(t, "scoped-auth-routes@example.com")
	scopedToken := createScopedCalendarPAT(t, ts, jwtToken)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/auth/switch-account", fmt.Sprintf(`{"account_id":%q}`, auth.User.AccountID)},
		{http.MethodPost, "/api/auth/oauth/link", `{"link_token":"attacker-controlled"}`},
		{http.MethodPost, "/api/auth/refresh", ""},
		{http.MethodGet, "/api/auth/me", ""},
		{http.MethodGet, "/api/auth/accounts", ""},
		{http.MethodPost, "/api/auth/resend-verification", ""},
	}
	for _, request := range requests {
		rec := ts.doRequest(request.method, request.path, request.body, scopedToken)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s with scoped PAT: expected 403, got %d: %s", request.method, request.path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), `"token"`) {
			t.Errorf("%s %s with scoped PAT leaked a token: %s", request.method, request.path, rec.Body.String())
		}
	}

	// Full-access credentials keep working on the same routes.
	rec := ts.doRequest(http.MethodPost, "/api/auth/switch-account", fmt.Sprintf(`{"account_id":%q}`, auth.User.AccountID), jwtToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("switch-account with JWT: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = ts.doRequest(http.MethodGet, "/api/auth/me", "", jwtToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me with JWT: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// The scoped token still serves the one endpoint it was issued for.
	vault := ts.createTestVault(t, jwtToken, "Scoped Feed Vault")
	rec = ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/calendar.ics?token="+scopedToken, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("calendar.ics with scoped PAT: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "BEGIN:VCALENDAR") {
		t.Fatalf("calendar.ics body is not an iCalendar feed: %s", rec.Body.String())
	}
}
