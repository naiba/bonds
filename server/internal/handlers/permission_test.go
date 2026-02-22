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

// ==================== Helper Functions for New Sections ====================

func createTestJournal(t *testing.T, ts *testServer, token, vaultID string) string {
	t.Helper()
	path := fmt.Sprintf("/api/vaults/%s/journals", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Test Journal"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestJournal failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse journal data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

func createTestPost(t *testing.T, ts *testServer, token, vaultID, journalID string) string {
	t.Helper()
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts", vaultID, journalID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Test Post","written_at":"2024-01-01T00:00:00Z","sections":[{"position":1,"label":"Section","content":"Content"}]}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestPost failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse post data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

func createTestGroup(t *testing.T, ts *testServer, vaultID, name string) uint {
	t.Helper()
	group := models.Group{VaultID: vaultID, Name: name}
	if err := ts.db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	return group.ID
}

func createTestFile(t *testing.T, ts *testServer, vaultID string) uint {
	t.Helper()
	file := models.File{VaultID: vaultID, UUID: "test-uuid-perm", Name: "test.pdf", MimeType: "application/pdf", Type: "document", Size: 100}
	if err := ts.db.Create(&file).Error; err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	return file.ID
}

func createTestNotificationChannel(t *testing.T, ts *testServer, token string) string {
	t.Helper()
	rec := ts.doRequest(http.MethodPost, "/api/settings/notifications",
		`{"type":"email","content":"test-notif@example.com"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestNotificationChannel failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse notification data: %v", err)
	}
	return fmt.Sprintf("%v", data["id"])
}

// ==================== I. Cross-Vault IDOR Tests for Journals ====================

func TestCrossVaultJournalGetBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-journal-get@example.com")
	vault1 := ts.createTestVault(t, token, "Journal Vault A")
	vault2 := ts.createTestVault(t, token, "Journal Vault B")

	journalID := createTestJournal(t, ts, token, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s", vault2.ID, journalID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault journal GET, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultJournalUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-journal-update@example.com")
	vault1 := ts.createTestVault(t, token, "Journal Update Vault A")
	vault2 := ts.createTestVault(t, token, "Journal Update Vault B")

	journalID := createTestJournal(t, ts, token, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s", vault2.ID, journalID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault journal UPDATE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultJournalDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-journal-delete@example.com")
	vault1 := ts.createTestVault(t, token, "Journal Delete Vault A")
	vault2 := ts.createTestVault(t, token, "Journal Delete Vault B")

	journalID := createTestJournal(t, ts, token, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s", vault2.ID, journalID)
	rec := ts.doRequest(http.MethodDelete, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault journal DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== J. Cross-Vault IDOR Tests for Posts ====================

func TestCrossVaultPostGetBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-post-get@example.com")
	vault1 := ts.createTestVault(t, token, "Post Vault A")
	vault2 := ts.createTestVault(t, token, "Post Vault B")

	journal1ID := createTestJournal(t, ts, token, vault1.ID)
	post1ID := createTestPost(t, ts, token, vault1.ID, journal1ID)

	journal2ID := createTestJournal(t, ts, token, vault2.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s", vault2.ID, journal2ID, post1ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault post GET, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultPostUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-post-update@example.com")
	vault1 := ts.createTestVault(t, token, "Post Update Vault A")
	vault2 := ts.createTestVault(t, token, "Post Update Vault B")

	journal1ID := createTestJournal(t, ts, token, vault1.ID)
	post1ID := createTestPost(t, ts, token, vault1.ID, journal1ID)

	journal2ID := createTestJournal(t, ts, token, vault2.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s", vault2.ID, journal2ID, post1ID)
	rec := ts.doRequest(http.MethodPut, path, `{"title":"Hacked","written_at":"2024-01-01T00:00:00Z","sections":[{"position":1,"label":"S","content":"C"}]}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault post UPDATE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultPostDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-post-delete@example.com")
	vault1 := ts.createTestVault(t, token, "Post Delete Vault A")
	vault2 := ts.createTestVault(t, token, "Post Delete Vault B")

	journal1ID := createTestJournal(t, ts, token, vault1.ID)
	post1ID := createTestPost(t, ts, token, vault1.ID, journal1ID)

	journal2ID := createTestJournal(t, ts, token, vault2.ID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s", vault2.ID, journal2ID, post1ID)
	rec := ts.doRequest(http.MethodDelete, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault post DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== K. Cross-Vault IDOR Tests for Groups ====================

func TestCrossVaultGroupGetBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-group-get@example.com")
	vault1 := ts.createTestVault(t, token, "Group Vault A")
	vault2 := ts.createTestVault(t, token, "Group Vault B")

	groupID := createTestGroup(t, ts, vault1.ID, "Test Group A")

	path := fmt.Sprintf("/api/vaults/%s/groups/%d", vault2.ID, groupID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault group GET, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultGroupUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-group-update@example.com")
	vault1 := ts.createTestVault(t, token, "Group Update Vault A")
	vault2 := ts.createTestVault(t, token, "Group Update Vault B")

	groupID := createTestGroup(t, ts, vault1.ID, "Test Group Update")

	path := fmt.Sprintf("/api/vaults/%s/groups/%d", vault2.ID, groupID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault group UPDATE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultGroupDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-group-delete@example.com")
	vault1 := ts.createTestVault(t, token, "Group Delete Vault A")
	vault2 := ts.createTestVault(t, token, "Group Delete Vault B")

	groupID := createTestGroup(t, ts, vault1.ID, "Test Group Delete")

	path := fmt.Sprintf("/api/vaults/%s/groups/%d", vault2.ID, groupID)
	rec := ts.doRequest(http.MethodDelete, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault group DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== L. Cross-Vault IDOR Tests for Files ====================

func TestCrossVaultFileDownloadBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-file-download@example.com")
	vault1 := ts.createTestVault(t, token, "File Download Vault A")
	vault2 := ts.createTestVault(t, token, "File Download Vault B")

	fileID := createTestFile(t, ts, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/files/%d/download", vault2.ID, fileID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault file download, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultFileDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-file-delete@example.com")
	vault1 := ts.createTestVault(t, token, "File Delete Vault A")
	vault2 := ts.createTestVault(t, token, "File Delete Vault B")

	fileID := createTestFile(t, ts, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/files/%d", vault2.ID, fileID)
	rec := ts.doRequest(http.MethodDelete, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault file DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== M. Cross-User Notification Channel Isolation ====================

func TestCrossUserNotificationToggleBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "notif-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)

	token2, _ := ts.registerTestUser(t, "notif-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s/toggle", channelID)
	rec := ts.doRequest(http.MethodPut, path, "", token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification toggle, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossUserNotificationDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "notif-del-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)

	token2, _ := ts.registerTestUser(t, "notif-del-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s", channelID)
	rec := ts.doRequest(http.MethodDelete, path, "", token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== N. Cross-Vault VCard Export Blocked ====================

func TestCrossVaultVCardExportBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-vcard@example.com")
	vault1 := ts.createTestVault(t, token, "VCard Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "VCardContact")

	vault2 := ts.createTestVault(t, token, "VCard Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/vcard", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault vcard export, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== O. Viewer Cannot Write Journals/Posts/Groups ====================

func TestViewerCannotCreateJournal(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	path := fmt.Sprintf("/api/vaults/%s/journals", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Blocked Journal"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating journal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateJournal(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s", vaultID, journalID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating journal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteJournal(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s", vaultID, journalID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting journal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreatePost(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts", vaultID, journalID)
	rec := ts.doRequest(http.MethodPost, path, `{"title":"Blocked Post","written_at":"2024-01-01T00:00:00Z","sections":[{"position":1,"label":"S","content":"C"}]}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating post, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdatePost(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)
	postID := createTestPost(t, ts, adminToken, vaultID, journalID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s", vaultID, journalID, postID)
	rec := ts.doRequest(http.MethodPut, path, `{"title":"Hacked","written_at":"2024-01-01T00:00:00Z","sections":[{"position":1,"label":"S","content":"C"}]}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating post, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeletePost(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)
	postID := createTestPost(t, ts, adminToken, vaultID, journalID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s", vaultID, journalID, postID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting post, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateGroup(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	groupID := createTestGroup(t, ts, vaultID, "Viewer Group Update")

	path := fmt.Sprintf("/api/vaults/%s/groups/%d", vaultID, groupID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating group, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteGroup(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	groupID := createTestGroup(t, ts, vaultID, "Viewer Group Delete")

	path := fmt.Sprintf("/api/vaults/%s/groups/%d", vaultID, groupID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting group, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCanReadJournals(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	createTestJournal(t, ts, adminToken, vaultID)

	path := fmt.Sprintf("/api/vaults/%s/journals", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading journals, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCanReadGroups(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)

	createTestGroup(t, ts, vaultID, "Viewer Read Group")

	path := fmt.Sprintf("/api/vaults/%s/groups", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading groups, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCanReadPosts(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)

	journalID := createTestJournal(t, ts, adminToken, vaultID)
	createTestPost(t, ts, adminToken, vaultID, journalID)

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts", vaultID, journalID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Viewer reading posts, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== P. Cross-Vault Quick Facts IDOR ====================

func TestCrossVaultQuickFactUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-quickfact@example.com")
	vault1 := ts.createTestVault(t, token, "QuickFact Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "QFContact1")

	// Find a quick facts template for vault1 (seeded by SeedVaultDefaults)
	var template1 models.VaultQuickFactsTemplate
	if err := ts.db.Where("vault_id = ?", vault1.ID).First(&template1).Error; err != nil {
		t.Fatalf("failed to find quick facts template for vault1: %v", err)
	}

	// Create a quick fact for contact1
	createPath := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/%d", vault1.ID, contact1.ID, template1.ID)
	rec := ts.doRequest(http.MethodPost, createPath, `{"content":"Likes hiking"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create quick fact: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var qfData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &qfData); err != nil {
		t.Fatalf("failed to parse quick fact: %v", err)
	}
	qfID := fmt.Sprintf("%v", qfData["id"])

	vault2 := ts.createTestVault(t, token, "QuickFact Vault B")
	contact2 := ts.createTestContact(t, token, vault2.ID, "QFContact2")

	// Find a template for vault2
	var template2 models.VaultQuickFactsTemplate
	if err := ts.db.Where("vault_id = ?", vault2.ID).First(&template2).Error; err != nil {
		t.Fatalf("failed to find quick facts template for vault2: %v", err)
	}

	// Try to update quickfact1 via vault2/contact2 — should fail because qf belongs to contact1
	updatePath := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/%d/%s", vault2.ID, contact2.ID, template2.ID, qfID)
	rec = ts.doRequest(http.MethodPut, updatePath, `{"content":"Hacked"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault quick fact UPDATE, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultQuickFactDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-quickfact-del@example.com")
	vault1 := ts.createTestVault(t, token, "QF Del Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "QFDelContact1")

	var template1 models.VaultQuickFactsTemplate
	if err := ts.db.Where("vault_id = ?", vault1.ID).First(&template1).Error; err != nil {
		t.Fatalf("failed to find quick facts template: %v", err)
	}

	createPath := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/%d", vault1.ID, contact1.ID, template1.ID)
	rec := ts.doRequest(http.MethodPost, createPath, `{"content":"To delete"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create quick fact: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var qfData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &qfData); err != nil {
		t.Fatalf("failed to parse quick fact: %v", err)
	}
	qfID := fmt.Sprintf("%v", qfData["id"])

	vault2 := ts.createTestVault(t, token, "QF Del Vault B")
	contact2 := ts.createTestContact(t, token, vault2.ID, "QFDelContact2")

	var template2 models.VaultQuickFactsTemplate
	if err := ts.db.Where("vault_id = ?", vault2.ID).First(&template2).Error; err != nil {
		t.Fatalf("failed to find quick facts template for vault2: %v", err)
	}

	deletePath := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/%d/%s", vault2.ID, contact2.ID, template2.ID, qfID)
	rec = ts.doRequest(http.MethodDelete, deletePath, "", token)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-vault quick fact DELETE, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Q. Cross-Account Isolation for Journals/Files ====================

func TestCrossAccountJournalIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-journal-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Owner Journal Vault")
	createTestJournal(t, ts, token1, vault1.ID)

	token2, _ := ts.registerTestUser(t, "cross-acct-journal-intruder@example.com")

	path := fmt.Sprintf("/api/vaults/%s/journals", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account journal list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountFileIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-file-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Owner File Vault")
	createTestFile(t, ts, vault1.ID)

	token2, _ := ts.registerTestUser(t, "cross-acct-file-intruder@example.com")

	path := fmt.Sprintf("/api/vaults/%s/files", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account file list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountJournalCreateBlocked(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-journal-create-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Blocked Journal Create Vault")

	token2, _ := ts.registerTestUser(t, "cross-acct-journal-create-intruder@example.com")

	path := fmt.Sprintf("/api/vaults/%s/journals", vault1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Intruder Journal"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account journal create, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountGroupIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-group-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Owner Group Vault")
	createTestGroup(t, ts, vault1.ID, "Owner Group")

	token2, _ := ts.registerTestUser(t, "cross-acct-group-intruder@example.com")

	path := fmt.Sprintf("/api/vaults/%s/groups", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account group list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossAccountPostIsolation(t *testing.T) {
	ts := setupTestServer(t)

	token1, _ := ts.registerTestUser(t, "cross-acct-post-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Owner Post Vault")
	journalID := createTestJournal(t, ts, token1, vault1.ID)
	createTestPost(t, ts, token1, vault1.ID, journalID)

	token2, _ := ts.registerTestUser(t, "cross-acct-post-intruder@example.com")

	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts", vault1.ID, journalID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account post list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultJournalListIsolated(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-journal-list@example.com")
	vault1 := ts.createTestVault(t, token, "Journal List Vault A")
	vault2 := ts.createTestVault(t, token, "Journal List Vault B")

	createTestJournal(t, ts, token, vault1.ID)

	// List journals in vault2 — should NOT see vault1's journal
	path := fmt.Sprintf("/api/vaults/%s/journals", vault2.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for journal list, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var journals []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &journals); err != nil {
		t.Fatalf("failed to parse journals: %v", err)
	}
	if len(journals) != 0 {
		t.Errorf("expected 0 journals in vault2, got %d", len(journals))
	}
}

func TestCrossVaultGroupListIsolated(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-group-list@example.com")
	vault1 := ts.createTestVault(t, token, "Group List Vault A")
	vault2 := ts.createTestVault(t, token, "Group List Vault B")

	createTestGroup(t, ts, vault1.ID, "Isolated Group")

	// Count groups in vault2 — groups from seed only, not from vault1
	path := fmt.Sprintf("/api/vaults/%s/groups", vault2.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for group list, got %d: %s", rec.Code, rec.Body.String())
	}
	// Just verify it doesn't contain our custom group name
	resp := parseResponse(t, rec)
	var groups []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &groups); err != nil {
		t.Fatalf("failed to parse groups: %v", err)
	}
	for _, g := range groups {
		if g["name"] == "Isolated Group" {
			t.Error("vault2 should not contain vault1's 'Isolated Group'")
		}
	}
}

func TestCrossVaultFileListIsolated(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-file-list@example.com")
	vault1 := ts.createTestVault(t, token, "File List Vault A")
	vault2 := ts.createTestVault(t, token, "File List Vault B")

	createTestFile(t, ts, vault1.ID)

	path := fmt.Sprintf("/api/vaults/%s/files", vault2.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for file list, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var files []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &files); err != nil {
		t.Fatalf("failed to parse files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in vault2, got %d", len(files))
	}
}

func TestCrossVaultPostListIsolated(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "cross-vault-post-list@example.com")
	vault1 := ts.createTestVault(t, token, "Post List Vault A")
	vault2 := ts.createTestVault(t, token, "Post List Vault B")

	journal1ID := createTestJournal(t, ts, token, vault1.ID)
	createTestPost(t, ts, token, vault1.ID, journal1ID)

	journal2ID := createTestJournal(t, ts, token, vault2.ID)

	// List posts in vault2/journal2 — should NOT see vault1's posts
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts", vault2.ID, journal2ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for post list, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var posts []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &posts); err != nil {
		t.Fatalf("failed to parse posts: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts in vault2's journal, got %d", len(posts))
	}
}

// ==================== R. Vault Settings Manager-Only ====================

func setupEditorTest(t *testing.T) (ts *testServer, adminToken string, editorToken string, vaultID string, contactID string) {
	t.Helper()
	ts = setupTestServer(t)
	adminToken, auth := ts.registerTestUser(t, "editor-perm-admin@example.com")
	vault := ts.createTestVault(t, adminToken, "Editor Perm Vault")
	contact := ts.createTestContact(t, adminToken, vault.ID, "EditorTarget")

	editor := createSecondUser(t, ts, auth.User.AccountID, "editor-perm-user@example.com", false)
	addUserToVault(t, ts, editor.ID, vault.ID, models.PermissionEditor)
	editorToken = generateJWT(editor.ID, editor.AccountID, editor.Email, false, false)

	return ts, adminToken, editorToken, vault.ID, contact.ID
}

func TestViewerCannotAccessVaultSettings(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer accessing vault settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateVaultSettings(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings", vaultID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating vault settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotListVaultUsers(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer listing vault users, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotAddVaultUser(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"email":"intruder@example.com","permission":300}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer adding vault user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotListVaultLabels(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/labels", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer listing vault labels, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateVaultLabel(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/labels", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Hacked","color":"#ff0000"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating vault label, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotAccessVaultSettings(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings", vaultID)
	rec := ts.doRequest(http.MethodGet, path, "", editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor accessing vault settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotUpdateVaultSettings(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings", vaultID)
	rec := ts.doRequest(http.MethodPut, path, `{"name":"Hacked"}`, editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor updating vault settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotManageVaultUsers(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"email":"intruder@example.com","permission":300}`, editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor managing vault users, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== S. Viewer Cannot Write Sub-Resources ====================

func TestViewerCannotCreateCall(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/calls", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"called_at":"2024-01-01T00:00:00Z","description":"test"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating call, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateAddress(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/addresses", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"city":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating address, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateContactInfo(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/contactInformation", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"type_id":1,"content":"hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating contact info, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateLoan(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/loans", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Hacked","amount":100,"direction":"loaned"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating loan, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreatePet(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/pets", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Hacked","pet_category_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating pet, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateRelationship(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/relationships", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"related_contact_id":"fake","relationship_type_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating relationship, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateGoal(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/goals", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating goal, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateLifeEvent(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/timelineEvents", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Hacked","started_at":"2024-01-01T00:00:00Z"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating life event, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateMoodEvent(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/moodTrackingEvents", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"rated_at":"2024-01-01T00:00:00Z","parameter_id":1,"rating":3}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating mood event, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUploadPhoto(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/photos", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer uploading photo, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUploadDocument(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/documents", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer uploading document, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateQuickFact(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/1", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"content":"Hacked"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating quick fact, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateImportantDate(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/dates", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Hacked","date":"2024-01-01","type_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating important date, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateContactLabels(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/labels", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"label_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating contact labels, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateReligion(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/religion", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, `{"religion_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating religion, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateJobInfo(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/jobInformation", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, `{"company":"Hacked","title":"CEO"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating job info, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== T. Disabled User Access ====================

func TestDisabledUserCannotAccessVaults(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "disabled-perm-vaults@example.com")
	ts.db.Model(&models.User{}).Where("id = ?", auth.User.ID).Update("disabled", true)
	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disabled user accessing vaults, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDisabledUserCannotAccessContacts(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "disabled-perm-contacts@example.com")
	vault := ts.createTestVault(t, token, "Disabled Vault")
	ts.db.Model(&models.User{}).Where("id = ?", auth.User.ID).Update("disabled", true)
	path := fmt.Sprintf("/api/vaults/%s/contacts", vault.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disabled user accessing contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDisabledUserCannotAccessSettings(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "disabled-perm-settings@example.com")
	ts.db.Model(&models.User{}).Where("id = ?", auth.User.ID).Update("disabled", true)
	rec := ts.doRequest(http.MethodGet, "/api/settings/preferences", "", token)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disabled user accessing settings, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== V. Permission Escalation Prevention ====================

func TestViewerCannotAddUserToVault(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"email":"intruder@example.com","permission":300}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer adding user to vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotAddUserToVault(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users", vaultID)
	rec := ts.doRequest(http.MethodPost, path, `{"email":"intruder@example.com","permission":300}`, editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor adding user to vault, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotUpdateVaultUserPermission(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users/fake-id", vaultID)
	rec := ts.doRequest(http.MethodPut, path, `{"permission":100}`, editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor updating vault user permission, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEditorCannotRemoveVaultUser(t *testing.T) {
	ts, _, editorToken, vaultID, _ := setupEditorTest(t)
	path := fmt.Sprintf("/api/vaults/%s/settings/users/fake-id", vaultID)
	rec := ts.doRequest(http.MethodDelete, path, "", editorToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Editor removing vault user, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== W. Cross-Vault IDOR — DAV Subscriptions ====================

func TestCrossVaultDavSubscriptionListBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-dav-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "DAV Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-dav-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account DAV subscription list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultDavSubscriptionCreateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-dav-create-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "DAV Create Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-dav-create-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"url":"https://example.com/dav","username":"u","password":"p"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account DAV subscription create, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== X. Cross-Vault IDOR — Life Metrics ====================

func TestCrossVaultLifeMetricListBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-lm-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "LM Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-lm-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/lifeMetrics", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account life metric list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultLifeMetricCreateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-lm-create-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "LM Create Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-lm-create-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/lifeMetrics", vault1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Hacked"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account life metric create, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Y. Cross-Vault IDOR — Companies ====================

func TestCrossVaultCompanyListBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-co-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Company Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-co-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/companies", vault1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account company list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultCompanyCreateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-co-create-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Company Create Vault A")
	token2, _ := ts.registerTestUser(t, "cross-vault-co-create-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/companies", vault1.ID)
	rec := ts.doRequest(http.MethodPost, path, `{"name":"Hacked Corp"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account company create, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Z. Cross-Vault IDOR — Slices of Life ====================

func TestCrossVaultSliceOfLifeBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-slice-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "Slice Vault A")
	journalID := createTestJournal(t, ts, token1, vault1.ID)

	token2, _ := ts.registerTestUser(t, "cross-vault-slice-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/slices", vault1.ID, journalID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account slice of life list, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AA. Cross-Vault IDOR — Journal Metrics ====================

func TestCrossVaultJournalMetricBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-jm-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "JM Vault A")
	journalID := createTestJournal(t, ts, token1, vault1.ID)

	token2, _ := ts.registerTestUser(t, "cross-vault-jm-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/metrics", vault1.ID, journalID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account journal metric list, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossVaultJournalMetricCreateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-jm-create-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "JM Create Vault A")
	journalID := createTestJournal(t, ts, token1, vault1.ID)

	token2, _ := ts.registerTestUser(t, "cross-vault-jm-create-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/metrics", vault1.ID, journalID)
	rec := ts.doRequest(http.MethodPost, path, `{"label":"Hacked"}`, token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account journal metric create, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AB. Cross-Vault IDOR — Post Tags ====================

func TestCrossVaultPostTagBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "cross-vault-pt-owner@example.com")
	vault1 := ts.createTestVault(t, token1, "PT Vault A")
	journalID := createTestJournal(t, ts, token1, vault1.ID)
	postID := createTestPost(t, ts, token1, vault1.ID, journalID)

	token2, _ := ts.registerTestUser(t, "cross-vault-pt-intruder@example.com")
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s/tags", vault1.ID, journalID, postID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-account post tag list, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AC. Cross-User Isolation — WebAuthn/OAuth/Preferences ====================

func TestCrossUserWebAuthnCredentialDeleteBlocked(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "webauthn-owner@example.com")
	token2, _ := ts.registerTestUser(t, "webauthn-intruder@example.com")
	rec := ts.doRequest(http.MethodDelete, "/api/settings/webauthn/credentials/9999", "", token2)
	if rec.Code == http.StatusOK || rec.Code == http.StatusNoContent {
		t.Errorf("expected non-success for cross-user webauthn credential delete, got %d", rec.Code)
	}
}

func TestCrossUserOAuthUnlinkBlocked(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "oauth-owner@example.com")
	token2, _ := ts.registerTestUser(t, "oauth-intruder@example.com")
	rec := ts.doRequest(http.MethodDelete, "/api/settings/oauth/github", "", token2)
	if rec.Code == http.StatusOK || rec.Code == http.StatusNoContent {
		t.Errorf("expected non-success for cross-user OAuth unlink, got %d", rec.Code)
	}
}

func TestCrossUserPreferencesIsolated(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "prefs-user1@example.com")
	token2, _ := ts.registerTestUser(t, "prefs-user2@example.com")

	ts.doRequest(http.MethodPost, "/api/settings/preferences/locale", `{"locale":"zh"}`, token1)

	rec := ts.doRequest(http.MethodGet, "/api/settings/preferences", "", token2)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for user2 reading own prefs, got %d", rec.Code)
	}
	resp := parseResponse(t, rec)
	var prefs map[string]interface{}
	if err := json.Unmarshal(resp.Data, &prefs); err != nil {
		t.Fatalf("failed to parse prefs: %v", err)
	}
	if locale, ok := prefs["locale"].(string); ok && locale == "zh" {
		t.Error("user2 should not see user1's locale preference")
	}
}

// ==================== AD. Viewer Cannot Write — Additional Sub-Resources ====================

func TestViewerCannotToggleArchive(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/archive", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer toggling archive, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotSortContacts(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/sort", vaultID)
	rec := ts.doRequest(http.MethodPut, path, `{"sort":"first_name","direction":"asc"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer sorting contacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotMoveContact(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/move", vaultID, contactID)
	rec := ts.doRequest(http.MethodPost, path, `{"target_vault_id":"fake"}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer moving contact, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateContactTemplate(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/template", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, `{"template_id":1}`, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating contact template, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateAvatar(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/avatar", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating avatar, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteAvatar(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/avatar", vaultID, contactID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting avatar, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteJobInfo(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/jobInformation", vaultID, contactID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting job info, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotToggleQuickFacts(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/quickFacts/toggle", vaultID, contactID)
	rec := ts.doRequest(http.MethodPut, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer toggling quick facts, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AE. Cross-User Notification Operations ====================

func TestCrossUserNotificationUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "notif-upd-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)
	token2, _ := ts.registerTestUser(t, "notif-upd-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s", channelID)
	body := `{"label":"hacked","type":"email","content":"hacker@example.com"}`
	rec := ts.doRequest(http.MethodPut, path, body, token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification update, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossUserNotificationSendTestBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "notif-test-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)
	token2, _ := ts.registerTestUser(t, "notif-test-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s/test", channelID)
	rec := ts.doRequest(http.MethodPost, path, "", token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification test, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossUserNotificationListLogsBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "notif-logs-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)
	token2, _ := ts.registerTestUser(t, "notif-logs-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s/logs", channelID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification logs, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCrossUserNotificationVerifyBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "notif-verify-owner@example.com")
	channelID := createTestNotificationChannel(t, ts, token1)
	token2, _ := ts.registerTestUser(t, "notif-verify-intruder@example.com")

	path := fmt.Sprintf("/api/settings/notifications/%s/verify/fake-token", channelID)
	rec := ts.doRequest(http.MethodGet, path, "", token2)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user notification verify, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AF. Non-Admin Cannot Manage Users ====================

func TestNonAdminCannotListUsers(t *testing.T) {
	ts := setupTestServer(t)
	_, authResp := ts.registerTestUser(t, "admin-user-mgmt@example.com")
	nonAdminUser := createSecondUser(t, ts, authResp.User.AccountID, "nonadmin-user-mgmt@example.com", false)
	nonAdminToken := generateJWT(nonAdminUser.ID, nonAdminUser.AccountID, nonAdminUser.Email, false, false)

	rec := ts.doRequest(http.MethodGet, "/api/settings/users", "", nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin listing users, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotUpdateUser(t *testing.T) {
	ts := setupTestServer(t)
	_, authResp := ts.registerTestUser(t, "admin-usr-upd@example.com")
	nonAdminUser := createSecondUser(t, ts, authResp.User.AccountID, "nonadmin-usr-upd@example.com", false)
	nonAdminToken := generateJWT(nonAdminUser.ID, nonAdminUser.AccountID, nonAdminUser.Email, false, false)

	path := fmt.Sprintf("/api/settings/users/%s", nonAdminUser.ID)
	body := `{"first_name":"Hacked"}`
	rec := ts.doRequest(http.MethodPut, path, body, nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin updating user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNonAdminCannotDeleteUser(t *testing.T) {
	ts := setupTestServer(t)
	_, authResp := ts.registerTestUser(t, "admin-usr-del@example.com")
	nonAdminUser := createSecondUser(t, ts, authResp.User.AccountID, "nonadmin-usr-del@example.com", false)
	nonAdminToken := generateJWT(nonAdminUser.ID, nonAdminUser.AccountID, nonAdminUser.Email, false, false)

	path := fmt.Sprintf("/api/settings/users/%s", nonAdminUser.ID)
	rec := ts.doRequest(http.MethodDelete, path, "", nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin deleting user, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AG. Non-Admin Cannot Cancel Account ====================

func TestNonAdminCannotCancelAccount(t *testing.T) {
	ts := setupTestServer(t)
	_, authResp := ts.registerTestUser(t, "admin-cancel@example.com")
	nonAdminUser := createSecondUser(t, ts, authResp.User.AccountID, "nonadmin-cancel@example.com", false)
	nonAdminToken := generateJWT(nonAdminUser.ID, nonAdminUser.AccountID, nonAdminUser.Email, false, false)

	body := `{"password":"password123"}`
	rec := ts.doRequest(http.MethodDelete, "/api/settings/account", body, nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin canceling account, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AH. Cross-Vault Contact Sub-Resource IDOR ====================

func TestCrossVaultCallBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-call@example.com")
	vault1 := ts.createTestVault(t, token, "Call Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "CallContact")
	vault2 := ts.createTestVault(t, token, "Call Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/calls", vault2.ID, contact1.ID)
	body := `{"called_at":"2025-01-01T00:00:00Z","description":"test"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault call create, got %d", rec.Code)
	}
}

func TestCrossVaultAddressBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-addr@example.com")
	vault1 := ts.createTestVault(t, token, "Addr Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "AddrContact")
	vault2 := ts.createTestVault(t, token, "Addr Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/addresses", vault2.ID, contact1.ID)
	body := `{"street":"123 Main St","city":"Test","country":"US"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault address create, got %d", rec.Code)
	}
}

func TestCrossVaultContactInfoBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-cinfo@example.com")
	vault1 := ts.createTestVault(t, token, "CInfo Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "CInfoContact")
	vault2 := ts.createTestVault(t, token, "CInfo Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/contactInformation", vault2.ID, contact1.ID)
	body := `{"contact_information_type_id":1,"data":"test@test.com"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault contact info create, got %d", rec.Code)
	}
}

func TestCrossVaultLoanBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-loan@example.com")
	vault1 := ts.createTestVault(t, token, "Loan Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "LoanContact")
	vault2 := ts.createTestVault(t, token, "Loan Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/loans", vault2.ID, contact1.ID)
	body := `{"type":"debt","name":"Test Loan","amount_lent":100}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault loan create, got %d", rec.Code)
	}
}

func TestCrossVaultPetBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-pet@example.com")
	vault1 := ts.createTestVault(t, token, "Pet Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "PetContact")
	vault2 := ts.createTestVault(t, token, "Pet Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/pets", vault2.ID, contact1.ID)
	body := `{"pet_category_id":1,"name":"Rex"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault pet create, got %d", rec.Code)
	}
}

func TestCrossVaultRelationshipBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-rel@example.com")
	vault1 := ts.createTestVault(t, token, "Rel Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "RelContact1")
	contact2 := ts.createTestContact(t, token, vault1.ID, "RelContact2")
	vault2 := ts.createTestVault(t, token, "Rel Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/relationships", vault2.ID, contact1.ID)
	body := fmt.Sprintf(`{"relationship_type_id":1,"related_contact_id":"%s"}`, contact2.ID)
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault relationship create, got %d", rec.Code)
	}
}

func TestCrossVaultGoalBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-goal@example.com")
	vault1 := ts.createTestVault(t, token, "Goal Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "GoalContact")
	vault2 := ts.createTestVault(t, token, "Goal Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/goals", vault2.ID, contact1.ID)
	body := `{"name":"Test Goal"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault goal create, got %d", rec.Code)
	}
}

func TestCrossVaultTimelineEventBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-timeline@example.com")
	vault1 := ts.createTestVault(t, token, "Timeline Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "TimelineContact")
	vault2 := ts.createTestVault(t, token, "Timeline Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/timelineEvents", vault2.ID, contact1.ID)
	body := `{"label":"Test Event","started_at":"2025-01-01"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault timeline event create, got %d", rec.Code)
	}
}

func TestCrossVaultMoodEventBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-mood@example.com")
	vault1 := ts.createTestVault(t, token, "Mood Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "MoodContact")
	vault2 := ts.createTestVault(t, token, "Mood Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/moodTrackingEvents", vault2.ID, contact1.ID)
	body := `{"mood_tracking_parameter_id":1,"rated_at":"2025-01-01","note":"test"}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault mood event create, got %d", rec.Code)
	}
}

func TestCrossVaultImportantDateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-date@example.com")
	vault1 := ts.createTestVault(t, token, "Date Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "DateContact")
	vault2 := ts.createTestVault(t, token, "Date Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/dates", vault2.ID, contact1.ID)
	body := `{"contact_important_date_type_id":1,"label":"Birthday","day":1,"month":1,"year":2000}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault important date create, got %d", rec.Code)
	}
}

func TestCrossVaultContactLabelBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-clabel@example.com")
	vault1 := ts.createTestVault(t, token, "CLabel Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "CLabelContact")
	vault2 := ts.createTestVault(t, token, "CLabel Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/labels", vault2.ID, contact1.ID)
	body := `{"label_id":1}`
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault contact label create, got %d", rec.Code)
	}
}

func TestCrossVaultReligionUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-religion@example.com")
	vault1 := ts.createTestVault(t, token, "Religion Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "ReligionContact")
	vault2 := ts.createTestVault(t, token, "Religion Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/religion", vault2.ID, contact1.ID)
	body := `{"religion_id":1}`
	rec := ts.doRequest(http.MethodPut, path, body, token)
	if rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault religion update, got %d", rec.Code)
	}
}

func TestCrossVaultJobInfoUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-job@example.com")
	vault1 := ts.createTestVault(t, token, "Job Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "JobContact")
	vault2 := ts.createTestVault(t, token, "Job Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/jobInformation", vault2.ID, contact1.ID)
	body := `{"job_title":"CEO","company":"Test Corp"}`
	rec := ts.doRequest(http.MethodPut, path, body, token)
	if rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault job info update, got %d", rec.Code)
	}
}

func TestCrossVaultAvatarUpdateBlocked(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-avatar@example.com")
	vault1 := ts.createTestVault(t, token, "Avatar Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "AvatarContact")
	vault2 := ts.createTestVault(t, token, "Avatar Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/avatar", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodPut, path, "", token)
	if rec.Code == http.StatusOK {
		t.Errorf("expected error for cross-vault avatar update, got %d", rec.Code)
	}
}

func TestCrossVaultContactFeedReturnsEmpty(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "xvault-feed@example.com")
	vault1 := ts.createTestVault(t, token, "Feed Vault A")
	contact1 := ts.createTestContact(t, token, vault1.ID, "FeedContact")
	vault2 := ts.createTestVault(t, token, "Feed Vault B")

	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/feed", vault2.ID, contact1.ID)
	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (empty feed) for cross-vault contact feed, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== AI. Viewer Cannot Write Vault-Scoped Resources ====================

func TestViewerCannotCreateDavSubscription(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vaultID)
	body := `{"name":"Test","url":"https://example.com/dav","username":"user","password":"pass"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating DAV subscription, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateLifeMetric(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/lifeMetrics", vaultID)
	body := `{"label":"Test Metric"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating life metric, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateCompany(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/companies", vaultID)
	body := `{"name":"Test Corp","type":"employer"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating company, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateSliceOfLife(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)
	journalID := createTestJournal(t, ts, adminToken, vaultID)
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/slices", vaultID, journalID)
	body := `{"name":"Test Slice"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating slice of life, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateJournalMetric(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)
	journalID := createTestJournal(t, ts, adminToken, vaultID)
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/metrics", vaultID, journalID)
	body := `{"label":"Test Metric"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating journal metric, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreatePostMetric(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)
	journalID := createTestJournal(t, ts, adminToken, vaultID)
	postID := createTestPost(t, ts, adminToken, vaultID, journalID)
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s/metrics", vaultID, journalID, postID)
	body := `{"journal_metric_id":1,"value":5}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating post metric, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreatePostTag(t *testing.T) {
	ts, adminToken, viewerToken, vaultID, _ := setupViewerTest(t)
	journalID := createTestJournal(t, ts, adminToken, vaultID)
	postID := createTestPost(t, ts, adminToken, vaultID, journalID)
	path := fmt.Sprintf("/api/vaults/%s/journals/%s/posts/%s/tags", vaultID, journalID, postID)
	body := `{"name":"test-tag"}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer creating post tag, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotUpdateDefaultTab(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/defaultTab", vaultID)
	body := `{"default_tab":"feed"}`
	rec := ts.doRequest(http.MethodPut, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer updating default tab, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotAddContactToGroup(t *testing.T) {
	ts, _, viewerToken, vaultID, contactID := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/contacts/%s/groups", vaultID, contactID)
	body := `{"group_id":1}`
	rec := ts.doRequest(http.MethodPost, path, body, viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer adding contact to group, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteDavSubscription(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/dav/subscriptions/999", vaultID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting DAV subscription, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteLifeMetric(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/lifeMetrics/999", vaultID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting life metric, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotDeleteCompany(t *testing.T) {
	ts, _, viewerToken, vaultID, _ := setupViewerTest(t)
	path := fmt.Sprintf("/api/vaults/%s/companies/999", vaultID)
	rec := ts.doRequest(http.MethodDelete, path, "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for Viewer deleting company, got %d: %s", rec.Code, rec.Body.String())
	}
}
