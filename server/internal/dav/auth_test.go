package dav

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
