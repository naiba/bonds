package handlers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
)

func generateJWT(userID, accountID, email string, isAdmin bool, twoFactorPending bool) string {
	claims := &middleware.JWTClaims{
		UserID:           userID,
		AccountID:        accountID,
		Email:            email,
		IsAdmin:          isAdmin,
		TwoFactorPending: twoFactorPending,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret-key"))
	if err != nil {
		panic("failed to sign test JWT: " + err.Error())
	}
	return signed
}

func createSecondUser(t *testing.T, ts *testServer, accountID, email string, isAdmin bool) models.User {
	t.Helper()
	firstName := "Second"
	lastName := "User"
	password := "$2a$10$eImDhVHVc96dqKMpMfyMruPLaGrGPR6caDyqnCVq1G1u5IUXY1C5e" // bcrypt("password123")
	user := models.User{
		AccountID:              accountID,
		FirstName:              &firstName,
		LastName:               &lastName,
		Email:                  email,
		Password:               &password,
		IsAccountAdministrator: isAdmin,
	}
	if err := ts.db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}
	return user
}

func addUserToVault(t *testing.T, ts *testServer, userID, vaultID string, permission int) {
	t.Helper()
	uv := models.UserVault{
		VaultID:    vaultID,
		UserID:     userID,
		ContactID:  "",
		Permission: permission,
	}
	if err := ts.db.Create(&uv).Error; err != nil {
		t.Fatalf("failed to create user_vault: %v", err)
	}
}

func TestTwoFactorPendingTokenBlocked(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "2fa-block@example.com")

	pendingToken := generateJWT(auth.User.ID, auth.User.AccountID, auth.User.Email, auth.User.IsAdmin, true)

	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", pendingToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for 2FA-pending token on GET /api/vaults, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if resp.Success {
		t.Error("expected success=false for 2FA-pending token")
	}

	rec = ts.doRequest(http.MethodGet, "/api/auth/me", "", pendingToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for 2FA-pending token on GET /api/auth/me, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateContact(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "viewer-test-admin@example.com")
	vault := ts.createTestVault(t, token1, "Viewer Test Vault")
	ts.createTestContact(t, token1, vault.ID, "ExistingContact")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "viewer-test-user2@example.com", false)
	addUserToVault(t, ts, user2.ID, vault.ID, models.PermissionViewer)

	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token2)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading contacts, got %d: %s", rec.Code, rec.Body.String())
	}

	body := `{"first_name":"NewContact","last_name":"Doe"}`
	rec = ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts", body, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating contact, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCanCreateContact(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "editor-test-admin@example.com")
	vault := ts.createTestVault(t, token1, "Editor Test Vault")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "editor-test-user2@example.com", false)
	addUserToVault(t, ts, user2.ID, vault.ID, models.PermissionEditor)

	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	body := `{"first_name":"EditorContact","last_name":"Doe"}`
	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts", body, token2)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for Editor creating contact, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor deleting vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotAccessPersonalize(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "personalize-admin@example.com")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "personalize-nonadmin@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodGet, "/api/settings/personalize/genders", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin accessing personalize, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/settings/personalize/genders", "", token1)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin accessing personalize, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotAccessInvitations(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "invitations-admin@example.com")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "invitations-nonadmin@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodGet, "/api/settings/invitations", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin accessing invitations, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/settings/invitations", "", token1)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin accessing invitations, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactMustBelongToVault(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault@example.com")

	vault1 := ts.createTestVault(t, token, "Vault One")
	contact := ts.createTestContact(t, token, vault1.ID, "CrossVaultContact")

	vault2 := ts.createTestVault(t, token, "Vault Two")

	noteBody := `{"title":"Test Note","body":"Should fail"}`
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vault2.ID, contact.ID)
	rec := ts.doRequest(http.MethodPost, path, noteBody, token)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault contact access, got %d: %s", rec.Code, rec.Body.String())
	}
}
