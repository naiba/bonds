package handlers_test

import (
	"encoding/json"
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

func createTestNote(t *testing.T, ts *testServer, token, vaultID, contactID string) string {
	t.Helper()
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Test Note","body":"body"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestNote failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse note data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

func createTestReminder(t *testing.T, ts *testServer, token, vaultID, contactID string) string {
	t.Helper()
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/reminders", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Test Reminder","day":1,"month":1,"type":"one_time"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestReminder failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse reminder data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

func createTestTask(t *testing.T, ts *testServer, token, vaultID, contactID string) string {
	t.Helper()
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/tasks", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Test Task","description":"desc"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestTask failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse task data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

// ==================== A. Cross-Account Isolation ====================

func TestCrossAccountVaultIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-vault-owner@example.com")
	vault := ts.createTestVault(t, token1, "Account1 Vault")

	token2, _ := ts.registerTestUser(t, "cross-acct-vault-intruder@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account GET vault, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPut, "/api/vaults/"+vault.ID, `{"name":"Hacked","description":"nope"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account PUT vault, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account DELETE vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountContactIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-contact-owner@example.com")
	vault := ts.createTestVault(t, token1, "Contact Owner Vault")
	ts.createTestContact(t, token1, vault.ID, "Private")

	token2, _ := ts.registerTestUser(t, "cross-acct-contact-intruder@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account GET contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountSettingsIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-settings-admin@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/settings/personalize/genders", "", token1)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin accessing own personalize, got %d: %s", rec.Code, rec.Body.String())
	}

	token2, _ := ts.registerTestUser(t, "cross-acct-settings-other@example.com")

	rec = ts.doRequest(http.MethodGet, "/api/settings/personalize/genders", "", token2)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for other admin accessing own personalize, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== B. Cross-Vault Data Isolation ====================

func TestCrossVaultContactReadBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-read@example.com")
	vault1 := ts.createTestVault(t, token, "Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "ContactA")

	vault2 := ts.createTestVault(t, token, "Vault B")
	ts.createTestContact(t, token, vault2.ID, "ContactB")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for reading notes via wrong vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultNoteWriteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-note@example.com")
	vault1 := ts.createTestVault(t, token, "Note Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "NoteContact")

	vault2 := ts.createTestVault(t, token, "Note Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Leak","body":"Should fail"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault note write, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultReminderBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-reminder@example.com")
	vault1 := ts.createTestVault(t, token, "Reminder Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "ReminderContact")

	vault2 := ts.createTestVault(t, token, "Reminder Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/reminders", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault reminder list, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPost, path, `{"label":"Leak","day":1,"month":1,"type":"one_time"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault reminder create, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultTaskBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-task@example.com")
	vault1 := ts.createTestVault(t, token, "Task Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "TaskContact")

	vault2 := ts.createTestVault(t, token, "Task Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/tasks", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault task list, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPost, path, `{"label":"Leak","description":"fail"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault task create, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultContactUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-update@example.com")
	vault1 := ts.createTestVault(t, token, "Update Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "UpdateContact")

	vault2 := ts.createTestVault(t, token, "Update Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Cross vault","body":"leak"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault note create via wrong vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultContactDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-delete@example.com")
	vault1 := ts.createTestVault(t, token, "Delete Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "DeleteContact")

	vault2 := ts.createTestVault(t, token, "Delete Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/tasks", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Cross vault","description":"leak"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault task create via wrong vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultDirectContactGetBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-direct-get@example.com")
	vault1 := ts.createTestVault(t, token, "Direct Get Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "DirectGetContact")

	vault2 := ts.createTestVault(t, token, "Direct Get Vault B")

	// Try to GET contact1 via vault2's URL — should be blocked
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault direct contact GET, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultDirectContactUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-direct-update@example.com")
	vault1 := ts.createTestVault(t, token, "Direct Update Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "DirectUpdateContact")

	vault2 := ts.createTestVault(t, token, "Direct Update Vault B")

	// Try to UPDATE contact1 via vault2's URL — should be blocked
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPut, path, `{"first_name":"Hacked","last_name":"Nope"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault direct contact UPDATE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultDirectContactDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-direct-delete@example.com")
	vault1 := ts.createTestVault(t, token, "Direct Delete Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "DirectDeleteContact")

	vault2 := ts.createTestVault(t, token, "Direct Delete Vault B")

	// Try to DELETE contact1 via vault2's URL — should be blocked
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodDelete, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault direct contact DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultToggleArchiveBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-archive@example.com")
	vault1 := ts.createTestVault(t, token, "Archive Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "ArchiveContact")

	vault2 := ts.createTestVault(t, token, "Archive Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/archive", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPut, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault toggle archive, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultToggleFavoriteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-favorite@example.com")
	vault1 := ts.createTestVault(t, token, "Favorite Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "FavoriteContact")

	vault2 := ts.createTestVault(t, token, "Favorite Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/favorite", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPut, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault toggle favorite, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== C. Viewer Permission Enforcement ====================

func setupViewerTest(t *testing.T) (ts *testServer, adminToken string, viewerToken string, vaultID string, contactID string) {
	t.Helper()
	ts = setupTestServer(t)
	adminToken, auth := ts.registerTestUser(t, "viewer-perm-admin@example.com")
	vault := ts.createTestVault(t, adminToken, "Viewer Perm Vault")
	contact := ts.createTestContact(t, adminToken, vault.ID, "ViewerTarget")

	viewer := createSecondUser(t, ts, auth.User.AccountID, "viewer-perm-user@example.com", false)
	addUserToVault(t, ts, viewer.ID, vault.ID, models.PermissionViewer)
	viewerToken = generateJWT(viewer.ID, viewer.AccountID, viewer.Email, false, false)

	return ts, adminToken, viewerToken, vault.ID, contact.ID
}

func TestViewerCanReadNotes(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, contactID := setupViewerTest(t)
	createTestNote(t, ts, adminToken, vaultID, contactID)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vaultID, contactID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading notes, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateNote(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Nope","body":"Blocked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating note, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateNote(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, contactID := setupViewerTest(t)
	noteID := createTestNote(t, ts, adminToken, vaultID, contactID)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes/%s", vaultID, contactID, noteID)
	rec := ts.doRequest(http.MethodPut, path, `{"title":"Updated","body":"Nope"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating note, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteNote(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, contactID := setupViewerTest(t)
	noteID := createTestNote(t, ts, adminToken, vaultID, contactID)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes/%s", vaultID, contactID, noteID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting note, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateReminder(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/reminders", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Nope","day":1,"month":1,"type":"one_time"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating reminder, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateTask(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/tasks", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Nope","description":"blocked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating task, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateContact(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, `{"first_name":"Hacked","last_name":"Nope"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating contact, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteContact(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s", vaultID, contactID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting contact, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotImportVCard(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vaultID+"/contacts/import", "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer importing vCard, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUploadFile(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vaultID+"/files", "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer uploading file, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCanReadExport(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vaultID+"/contacts/export", "", viewerToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading export, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== D. Editor vs Manager Boundary ====================

func TestEditorCannotDeleteVault(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "editor-del-vault-admin@example.com")
	vault := ts.createTestVault(t, token1, "Editor Delete Vault")

	editor := createSecondUser(t, ts, auth1.User.AccountID, "editor-del-vault-user@example.com", false)
	addUserToVault(t, ts, editor.ID, vault.ID, models.PermissionEditor)
	editorToken := generateJWT(editor.ID, editor.AccountID, editor.Email, false, false)

	rec := ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID, "", editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor deleting vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCanCreateUpdateDeleteNotes(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "editor-notes-admin@example.com")
	vault := ts.createTestVault(t, token1, "Editor Notes Vault")
	contact := ts.createTestContact(t, token1, vault.ID, "EditorNoteTarget")

	editor := createSecondUser(t, ts, auth1.User.AccountID, "editor-notes-user@example.com", false)
	addUserToVault(t, ts, editor.ID, vault.ID, models.PermissionEditor)
	editorToken := generateJWT(editor.ID, editor.AccountID, editor.Email, false, false)

	basePath := fmt.Sprintf("/api/vaults/%s/contacts/%s/notes", vault.ID, contact.ID)

	rec := ts.doRequest(http.MethodPost, basePath, `{"title":"Editor Note","body":"content"}`, editorToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Editor creating note, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var noteData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &noteData); err != nil {
		t.Fatalf("failed to parse note: %v", err)
	}
	noteID := fmt.Sprintf("%v", noteData["id"])

	rec = ts.doRequest(http.MethodPut, basePath+"/"+noteID, `{"title":"Updated","body":"new"}`, editorToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Editor updating note, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodDelete, basePath+"/"+noteID, "", editorToken)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for Editor deleting note, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCanCreateUpdateDeleteContacts(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "editor-contacts-admin@example.com")
	vault := ts.createTestVault(t, token1, "Editor Contacts Vault")

	editor := createSecondUser(t, ts, auth1.User.AccountID, "editor-contacts-user@example.com", false)
	addUserToVault(t, ts, editor.ID, vault.ID, models.PermissionEditor)
	editorToken := generateJWT(editor.ID, editor.AccountID, editor.Email, false, false)

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts",
		`{"first_name":"EditorCreated","last_name":"Contact"}`, editorToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Editor creating contact, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var contactResp contactData
	if err := json.Unmarshal(resp.Data, &contactResp); err != nil {
		t.Fatalf("failed to parse contact: %v", err)
	}

	rec = ts.doRequest(http.MethodPut, "/api/vaults/"+vault.ID+"/contacts/"+contactResp.ID,
		`{"first_name":"Renamed","last_name":"Contact"}`, editorToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Editor updating contact, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID+"/contacts/"+contactResp.ID, "", editorToken)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for Editor deleting contact, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== E. No-Vault-Access User ====================

func TestNoVaultAccessUserCannotRead(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "no-vault-access-admin@example.com")
	vault := ts.createTestVault(t, token1, "Restricted Vault")
	ts.createTestContact(t, token1, vault.ID, "Restricted")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "no-vault-access-user@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for user without vault access reading contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNoVaultAccessUserCannotWrite(t *testing.T) {
	ts := setupTestServer(t)

	token1, auth1 := ts.registerTestUser(t, "no-vault-write-admin@example.com")
	vault := ts.createTestVault(t, token1, "Write Restricted Vault")

	user2 := createSecondUser(t, ts, auth1.User.AccountID, "no-vault-write-user@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts",
		`{"first_name":"Blocked","last_name":"User"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for user without vault access writing contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== F. 2FA Token Comprehensive ====================

func TestTwoFactorPendingCannotAccessSettings(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "2fa-settings-block@example.com")
	pendingToken := generateJWT(auth.User.ID, auth.User.AccountID, auth.User.Email, auth.User.IsAdmin, true)

	rec := ts.doRequest(http.MethodGet, "/api/settings/preferences", "", pendingToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for 2FA-pending on settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTwoFactorPendingCannotCreateVault(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "2fa-vault-block@example.com")
	pendingToken := generateJWT(auth.User.ID, auth.User.AccountID, auth.User.Email, auth.User.IsAdmin, true)

	rec := ts.doRequest(http.MethodPost, "/api/vaults", `{"name":"Blocked","description":"nope"}`, pendingToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for 2FA-pending creating vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTwoFactorPendingCannotAccessContacts(t *testing.T) {
	ts := setupTestServer(t)

	token, auth := ts.registerTestUser(t, "2fa-contacts-block@example.com")
	vault := ts.createTestVault(t, token, "2FA Contact Vault")

	pendingToken := generateJWT(auth.User.ID, auth.User.AccountID, auth.User.Email, auth.User.IsAdmin, true)

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", pendingToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for 2FA-pending accessing contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== G. Admin-Only Settings Comprehensive ====================

func TestNonAdminCannotCreatePersonalize(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "nonadmin-create-pers-admin@example.com")
	user2 := createSecondUser(t, ts, auth.User.AccountID, "nonadmin-create-pers@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodPost, "/api/settings/personalize/genders", `{"name":"Custom"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin POST personalize, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotUpdatePersonalize(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "nonadmin-update-pers-admin@example.com")
	user2 := createSecondUser(t, ts, auth.User.AccountID, "nonadmin-update-pers@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodPut, "/api/settings/personalize/genders/1", `{"name":"Updated"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin PUT personalize, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotDeletePersonalize(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "nonadmin-delete-pers-admin@example.com")
	user2 := createSecondUser(t, ts, auth.User.AccountID, "nonadmin-delete-pers@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodDelete, "/api/settings/personalize/genders/1", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin DELETE personalize, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotCreateInvitation(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "nonadmin-create-inv-admin@example.com")
	user2 := createSecondUser(t, ts, auth.User.AccountID, "nonadmin-create-inv@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodPost, "/api/settings/invitations",
		`{"email":"invited@example.com","permission":200}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin POST invitation, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotDeleteInvitation(t *testing.T) {
	ts := setupTestServer(t)

	_, auth := ts.registerTestUser(t, "nonadmin-delete-inv-admin@example.com")
	user2 := createSecondUser(t, ts, auth.User.AccountID, "nonadmin-delete-inv@example.com", false)
	token2 := generateJWT(user2.ID, user2.AccountID, user2.Email, false, false)

	rec := ts.doRequest(http.MethodDelete, "/api/settings/invitations/1", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin DELETE invitation, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCanAccessAllSettings(t *testing.T) {
	ts := setupTestServer(t)

	adminToken, _ := ts.registerTestUser(t, "admin-settings-full@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/settings/personalize/genders", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin GET personalize, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/settings/invitations", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin GET invitations, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/settings/preferences", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin GET preferences, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== H. Unauthenticated Access ====================

func TestUnauthenticatedCannotAccessVaults(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated GET /api/vaults, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnauthenticatedCannotAccessSettings(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/settings/preferences", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated GET /api/settings/preferences, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnauthenticatedCannotAccessContacts(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "unauth-contacts@example.com")
	vault := ts.createTestVault(t, token, "Unauth Contacts Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated GET contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}
