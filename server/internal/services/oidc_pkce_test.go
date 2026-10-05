package services

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/naiba/bonds/internal/models"
)

func TestOIDCPKCEExchangeAndUserInfo(t *testing.T) {
	var issuer, challenge string
	var tokenCalls, userinfoCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/discovery":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer": issuer, "authorization_endpoint": issuer + "/authorize",
				"token_endpoint": issuer + "/token", "userinfo_endpoint": issuer + "/userinfo",
				"code_challenge_methods_supported": []string{"S256"},
			})
		case "/token":
			tokenCalls++
			_ = r.ParseForm()
			id, secret, ok := r.BasicAuth()
			verifier := r.PostForm.Get("code_verifier")
			sum := sha256.Sum256([]byte(verifier))
			if !ok || id != "client" || secret != "secret" || len(verifier) < 43 ||
				base64.RawURLEncoding.EncodeToString(sum[:]) != challenge ||
				r.PostForm.Get("redirect_uri") != "https://bonds.test/api/auth/blog/callback" {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": "reader", "iss": issuer, "aud": "client", "exp": time.Now().Add(time.Hour).Unix(),
			}).SignedString([]byte("test-idp-key"))
			if err != nil {
				t.Error(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "access", "token_type": "Bearer", "expires_in": 3600, "id_token": token})
		case "/userinfo":
			userinfoCalls++
			if r.Header.Get("Authorization") != "Bearer access" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"sub":"reader","email":"reader@example.test","email_verified":true,"name":"Blog Reader"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	issuer = server.URL
	provider, err := createGothProvider(models.OAuthProvider{Type: "oidc", Name: "blog", ClientID: "client", ClientSecret: "secret", DiscoveryURL: issuer + "/discovery"}, "https://bonds.test")
	if err != nil {
		t.Fatal(err)
	}
	first, err := provider.BeginAuth("state-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.BeginAuth("state-2")
	if err != nil {
		t.Fatal(err)
	}
	firstURL, _ := first.GetAuthURL()
	secondURL, _ := second.GetAuthURL()
	auth, _ := url.Parse(firstURL)
	other, _ := url.Parse(secondURL)
	challenge = auth.Query().Get("code_challenge")
	if len(challenge) != 43 || auth.Query().Get("code_challenge_method") != "S256" ||
		challenge == other.Query().Get("code_challenge") || auth.Query().Get("state") != "state-1" {
		t.Fatal("PKCE/state must be unique to each login")
	}
	for _, scope := range []string{"openid", "email", "profile"} {
		if !strings.Contains(" "+auth.Query().Get("scope")+" ", " "+scope+" ") {
			t.Fatalf("missing scope %s", scope)
		}
	}
	if auth.Query().Has("code_verifier") {
		t.Fatal("authorization URL exposed verifier")
	}
	// Gothic serializes the session into its cookie between authorization and
	// callback; a second login must not invalidate the first one's verifier.
	restored, err := provider.UnmarshalSession(first.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	_, err = restored.Authorize(provider, url.Values{"code": {"valid-code"}, "code_verifier": {"attacker"}, "redirect_uri": {"https://attacker.test"}})
	if err != nil {
		t.Fatalf("PKCE exchange: %v", err)
	}
	user, err := provider.FetchUser(restored)
	if err != nil || user.UserID != "reader" || user.Email != "reader@example.test" || user.Name != "Blog Reader" {
		t.Fatalf("userinfo identity was not preserved: %v", err)
	}
	if tokenCalls != 1 || userinfoCalls != 1 {
		t.Fatal("expected real token exchange and userinfo requests")
	}
	if _, err := restored.Authorize(provider, url.Values{"error": {"access_denied"}}); err == nil || tokenCalls != 1 {
		t.Fatal("denied authorization must not exchange tokens")
	}
	if _, err := second.Authorize(provider, url.Values{"code": {"first-login-code"}}); err == nil {
		t.Fatal("another login's verifier must fail")
	}
	for _, raw := range []string{`{`, `{}`, `{"AuthURL":"https://example.test"}`} {
		if _, err := provider.UnmarshalSession(raw); err == nil {
			t.Fatal("invalid or missing PKCE session accepted")
		}
	}
	custom, err := createGothProvider(models.OAuthProvider{Type: "oidc", Name: "custom", ClientID: "client", ClientSecret: "secret", DiscoveryURL: issuer + "/discovery", Scopes: "email, profile, offline_access"}, "https://bonds.test")
	if err != nil {
		t.Fatal(err)
	}
	session, _ := custom.BeginAuth("custom-state")
	raw, _ := session.GetAuthURL()
	parsed, _ := url.Parse(raw)
	if !strings.Contains(parsed.Query().Get("scope"), "offline_access") || !strings.Contains(parsed.Query().Get("scope"), "openid") {
		t.Fatal("custom scopes ignored or mandatory openid scope missing")
	}
}
