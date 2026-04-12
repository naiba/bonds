package dav

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestBasicAuth_ValidCredentials(t *testing.T) {
	db := testutil.SetupTestDB(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	user := models.User{
		AccountID: "test-account",
		Email:     "dav@example.com",
		Password:  &hashedStr,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	mw := BasicAuthMiddleware(db)

	var gotUserID, gotAccountID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		gotAccountID = AccountIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("dav@example.com", "password123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotUserID != user.ID {
		t.Errorf("expected user ID %q, got %q", user.ID, gotUserID)
	}
	if gotAccountID != user.AccountID {
		t.Errorf("expected account ID %q, got %q", user.AccountID, gotAccountID)
	}
}

func TestBasicAuth_InvalidPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	user := models.User{
		AccountID: "test-account",
		Email:     "dav@example.com",
		Password:  &hashedStr,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("dav@example.com", "wrong-password")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestBasicAuth_UserNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("nonexistent@example.com", "password123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestBasicAuth_NoCredentials(t *testing.T) {
	db := testutil.SetupTestDB(t)

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	// No SetBasicAuth — no credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("expected WWW-Authenticate header, got empty")
	}
}

func TestBasicAuth_NilPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// OAuth-only user with no password
	user := models.User{
		AccountID: "test-account",
		Email:     "oauth@example.com",
		Password:  nil,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("oauth@example.com", "anything")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestBasicAuth_DisabledUser(t *testing.T) {
	db := testutil.SetupTestDB(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	user := models.User{
		AccountID: "test-account",
		Email:     "disabled@example.com",
		Password:  &hashedStr,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Disable user after creation (GORM skips false zero-value on Create)
	if err := db.Model(&user).Update("disabled", true).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("disabled@example.com", "password123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rec.Code)
	}
}

func TestBasicAuth_TwoFactorEnabledUser(t *testing.T) {
	db := testutil.SetupTestDB(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	user := models.User{
		AccountID:            "test-account",
		Email:                "2fa-dav@example.com",
		Password:             &hashedStr,
		TwoFactorSecret:      &secret,
		TwoFactorConfirmedAt: &now,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	mw := BasicAuthMiddleware(db)

	var gotUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("2fa-dav@example.com", "password123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (2FA should not block DAV BasicAuth), got %d", rec.Code)
	}
	if gotUserID != user.ID {
		t.Errorf("expected user ID %q, got %q", user.ID, gotUserID)
	}
}

func TestBasicAuth_PAT_Valid(t *testing.T) {
	db := testutil.SetupTestDB(t)

	user := models.User{
		AccountID: "test-account",
		Email:     "pat@example.com",
		Password:  nil,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken := "bonds_abc123testtoken"
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	pat := models.PersonalAccessToken{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Name:      "test-pat",
		TokenHash: tokenHash,
		TokenHint: "bonds_abc...ken",
	}
	if err := db.Create(&pat).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	var gotUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("pat@example.com", rawToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotUserID != user.ID {
		t.Errorf("expected user ID %q, got %q", user.ID, gotUserID)
	}

	var updated models.PersonalAccessToken
	db.First(&updated, pat.ID)
	if updated.LastUsedAt == nil {
		t.Error("expected LastUsedAt to be updated after PAT authentication")
	}
}

func TestBasicAuth_PAT_Expired(t *testing.T) {
	db := testutil.SetupTestDB(t)

	user := models.User{
		AccountID: "test-account",
		Email:     "pat-expired@example.com",
		Password:  nil,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken := "bonds_expired123token"
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	expired := time.Now().Add(-24 * time.Hour)
	pat := models.PersonalAccessToken{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Name:      "expired-pat",
		TokenHash: tokenHash,
		TokenHint: "bonds_exp...ken",
		ExpiresAt: &expired,
	}
	if err := db.Create(&pat).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("pat-expired@example.com", rawToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired PAT, got %d", rec.Code)
	}
}

func TestBasicAuth_PAT_WrongUser(t *testing.T) {
	db := testutil.SetupTestDB(t)

	user1 := models.User{AccountID: "acct-1", Email: "user1@example.com"}
	user2 := models.User{AccountID: "acct-2", Email: "user2@example.com"}
	db.Create(&user1)
	db.Create(&user2)

	rawToken := "bonds_wronguser123"
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	pat := models.PersonalAccessToken{
		UserID:    user1.ID,
		AccountID: user1.AccountID,
		Name:      "user1-pat",
		TokenHash: tokenHash,
		TokenHint: "bonds_wro...123",
	}
	db.Create(&pat)

	mw := BasicAuthMiddleware(db)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("user2@example.com", rawToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when PAT belongs to different user, got %d", rec.Code)
	}
}

func TestBasicAuth_PAT_TwoFactorEnabled(t *testing.T) {
	db := testutil.SetupTestDB(t)

	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	user := models.User{
		AccountID:            "test-account",
		Email:                "2fa-pat@example.com",
		Password:             nil,
		TwoFactorSecret:      &secret,
		TwoFactorConfirmedAt: &now,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rawToken := "bonds_2fapat123token"
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	pat := models.PersonalAccessToken{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Name:      "2fa-pat",
		TokenHash: tokenHash,
		TokenHint: "bonds_2fa...ken",
	}
	if err := db.Create(&pat).Error; err != nil {
		t.Fatalf("create PAT: %v", err)
	}

	mw := BasicAuthMiddleware(db)
	var gotUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dav/", nil)
	req.SetBasicAuth("2fa-pat@example.com", rawToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (2FA should not block PAT-based DAV auth), got %d", rec.Code)
	}
	if gotUserID != user.ID {
		t.Errorf("expected user ID %q, got %q", user.ID, gotUserID)
	}
}
