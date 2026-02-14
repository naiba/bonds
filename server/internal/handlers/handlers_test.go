package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/handlers"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

type testServer struct {
	e   *echo.Echo
	db  *gorm.DB
	cfg *config.Config
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *apiError       `json:"error,omitempty"`
	Meta    *apiMeta        `json:"meta,omitempty"`
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type apiMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type authData struct {
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expires_at"`
	User      userData `json:"user"`
}

type userData struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
}

type vaultData struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type contactData struct {
	ID         string `json:"id"`
	VaultID    string `json:"vault_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname"`
	IsArchived bool   `json:"is_archived"`
	IsFavorite bool   `json:"is_favorite"`
}

func setupTestServer(t *testing.T) *testServer {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		Server:   config.ServerConfig{Port: "8080", Host: "localhost"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"},
		JWT: config.JWTConfig{
			Secret:     "test-secret-key",
			ExpiryHrs:  24,
			RefreshHrs: 168,
		},
		App: config.AppConfig{Name: "Monica Test", Env: "test", URL: "http://localhost:8080"},
	}

	e := echo.New()
	handlers.RegisterRoutes(e, db, cfg)

	return &testServer{e: e, db: db, cfg: cfg}
}

func (ts *testServer) doRequest(method, path, body string, token string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	ts.e.ServeHTTP(rec, req)
	return rec
}

func (ts *testServer) doMultipartUpload(t *testing.T, path, token, fieldName, fileName, mimeType string, fileData []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	h.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create form part: %v", err)
	}
	if _, err := part.Write(fileData); err != nil {
		t.Fatalf("failed to write file data: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	ts.e.ServeHTTP(rec, req)
	return rec
}

func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if rec.Body.Len() == 0 {
		return resp
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, rec.Body.String())
	}
	return resp
}

func (ts *testServer) registerTestUser(t *testing.T, email string) (string, authData) {
	t.Helper()
	body := `{"first_name":"Test","last_name":"User","email":"` + email + `","password":"password123"}`
	rec := ts.doRequest(http.MethodPost, "/api/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data authData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse auth data: %v", err)
	}
	return data.Token, data
}

func (ts *testServer) createTestVault(t *testing.T, token, name string) vaultData {
	t.Helper()
	body := `{"name":"` + name + `","description":"test vault"}`
	rec := ts.doRequest(http.MethodPost, "/api/vaults", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create vault failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data vaultData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse vault data: %v", err)
	}
	return data
}

func (ts *testServer) createTestContact(t *testing.T, token, vaultID, firstName string) contactData {
	t.Helper()
	body := `{"first_name":"` + firstName + `","last_name":"Doe"}`
	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vaultID+"/contacts", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create contact failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data contactData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse contact data: %v", err)
	}
	return data
}

func TestRegister_Success(t *testing.T) {
	ts := setupTestServer(t)
	rec := ts.doRequest(http.MethodPost, "/api/auth/register",
		`{"first_name":"John","last_name":"Doe","email":"john@example.com","password":"password123"}`, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data authData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse auth data: %v", err)
	}
	if data.Token == "" {
		t.Error("expected non-empty token")
	}
	if data.User.Email != "john@example.com" {
		t.Errorf("expected email=john@example.com, got %s", data.User.Email)
	}
	if data.User.FirstName != "John" {
		t.Errorf("expected first_name=John, got %s", data.User.FirstName)
	}
	if data.User.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "dup@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/auth/register",
		`{"first_name":"Jane","last_name":"Doe","email":"dup@example.com","password":"password123"}`, "")

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "CONFLICT" {
		t.Errorf("expected error code CONFLICT, got %+v", resp.Error)
	}
}

func TestRegister_ValidationError(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/auth/register",
		`{"email":"","password":""}`, "")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %+v", resp.Error)
	}
}

func TestLogin_Success(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "login@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/auth/login",
		`{"email":"login@example.com","password":"password123"}`, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data authData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse auth data: %v", err)
	}
	if data.Token == "" {
		t.Error("expected non-empty token")
	}
	if data.User.Email != "login@example.com" {
		t.Errorf("expected email=login@example.com, got %s", data.User.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "wrongpw@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/auth/login",
		`{"email":"wrongpw@example.com","password":"wrongpassword"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %+v", resp.Error)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/auth/login",
		`{"email":"nobody@example.com","password":"password123"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %+v", resp.Error)
	}
}

func TestMe_WithValidToken(t *testing.T) {
	ts := setupTestServer(t)
	token, regData := ts.registerTestUser(t, "me@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/auth/me", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data userData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse user data: %v", err)
	}
	if data.Email != "me@example.com" {
		t.Errorf("expected email=me@example.com, got %s", data.Email)
	}
	if data.ID != regData.User.ID {
		t.Errorf("expected id=%s, got %s", regData.User.ID, data.ID)
	}
}

func TestMe_WithoutToken(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/auth/me", "", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMe_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/auth/me", "", "invalid-token-value")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "refresh@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/auth/refresh", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data authData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse auth data: %v", err)
	}
	if data.Token == "" {
		t.Error("expected non-empty refreshed token")
	}
	if data.User.Email != "refresh@example.com" {
		t.Errorf("expected email=refresh@example.com, got %s", data.User.Email)
	}
}

func TestRefresh_WithoutToken(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/auth/refresh", "", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVaultList_Empty(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vlist@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var vaults []vaultData
	if err := json.Unmarshal(resp.Data, &vaults); err != nil {
		t.Fatalf("failed to parse vault list: %v", err)
	}
	if len(vaults) != 0 {
		t.Errorf("expected 0 vaults, got %d", len(vaults))
	}
}

func TestVaultList_WithVaults(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vlist2@example.com")
	ts.createTestVault(t, token, "Vault A")
	ts.createTestVault(t, token, "Vault B")

	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var vaults []vaultData
	if err := json.Unmarshal(resp.Data, &vaults); err != nil {
		t.Fatalf("failed to parse vault list: %v", err)
	}
	if len(vaults) != 2 {
		t.Errorf("expected 2 vaults, got %d", len(vaults))
	}
}

func TestVaultCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vcreate@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/vaults",
		`{"name":"My Vault","description":"A test vault"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data vaultData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse vault data: %v", err)
	}
	if data.Name != "My Vault" {
		t.Errorf("expected name=My Vault, got %s", data.Name)
	}
	if data.Description != "A test vault" {
		t.Errorf("expected description=A test vault, got %s", data.Description)
	}
	if data.ID == "" {
		t.Error("expected non-empty vault ID")
	}
}

func TestVaultCreate_ValidationError(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vval@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/vaults",
		`{"name":"","description":"no name"}`, token)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %+v", resp.Error)
	}
}

func TestVaultCreate_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/vaults",
		`{"name":"Fail","description":"no auth"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVaultGet_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vget@example.com")
	vault := ts.createTestVault(t, token, "Get Me")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data vaultData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse vault data: %v", err)
	}
	if data.Name != "Get Me" {
		t.Errorf("expected name=Get Me, got %s", data.Name)
	}
}

func TestVaultGet_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vnotfound@example.com")
	ts.createTestVault(t, token, "Exists")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/nonexistent-id", "", token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVaultUpdate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vupdate@example.com")
	vault := ts.createTestVault(t, token, "Old Name")

	rec := ts.doRequest(http.MethodPut, "/api/vaults/"+vault.ID,
		`{"name":"New Name","description":"updated"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data vaultData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse vault data: %v", err)
	}
	if data.Name != "New Name" {
		t.Errorf("expected name=New Name, got %s", data.Name)
	}
	if data.Description != "updated" {
		t.Errorf("expected description=updated, got %s", data.Description)
	}
}

func TestVaultDelete_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vdelete@example.com")
	vault := ts.createTestVault(t, token, "Delete Me")

	rec := ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID, "", token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID, "", token)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after delete, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestContactList_Empty(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "clist@example.com")
	vault := ts.createTestVault(t, token, "Contact Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var contacts []contactData
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(contacts))
	}
	if resp.Meta == nil {
		t.Fatal("expected meta in paginated response")
	}
	if resp.Meta.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Meta.Total)
	}
}

func TestContactList_WithContacts(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "clist2@example.com")
	vault := ts.createTestVault(t, token, "Contact Vault")
	ts.createTestContact(t, token, vault.ID, "Alice")
	ts.createTestContact(t, token, vault.ID, "Bob")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var contacts []contactData
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(contacts))
	}
	if resp.Meta == nil {
		t.Fatal("expected meta in paginated response")
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Meta.Total)
	}
}

func TestContactCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "ccreate@example.com")
	vault := ts.createTestVault(t, token, "Create Contact Vault")

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts",
		`{"first_name":"Jane","last_name":"Smith","nickname":"Janey"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data contactData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse contact data: %v", err)
	}
	if data.FirstName != "Jane" {
		t.Errorf("expected first_name=Jane, got %s", data.FirstName)
	}
	if data.LastName != "Smith" {
		t.Errorf("expected last_name=Smith, got %s", data.LastName)
	}
	if data.Nickname != "Janey" {
		t.Errorf("expected nickname=Janey, got %s", data.Nickname)
	}
	if data.VaultID != vault.ID {
		t.Errorf("expected vault_id=%s, got %s", vault.ID, data.VaultID)
	}
	if data.ID == "" {
		t.Error("expected non-empty contact ID")
	}
}

func TestContactCreate_ValidationError(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cval@example.com")
	vault := ts.createTestVault(t, token, "Validation Vault")

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts",
		`{"first_name":"","last_name":"NoFirst"}`, token)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %+v", resp.Error)
	}
}

func TestContactCreate_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cunauth@example.com")
	vault := ts.createTestVault(t, token, "Unauth Vault")

	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vault.ID+"/contacts",
		`{"first_name":"No","last_name":"Auth"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactGet_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cget@example.com")
	vault := ts.createTestVault(t, token, "Get Contact Vault")
	contact := ts.createTestContact(t, token, vault.ID, "GetMe")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts/"+contact.ID, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data contactData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse contact data: %v", err)
	}
	if data.FirstName != "GetMe" {
		t.Errorf("expected first_name=GetMe, got %s", data.FirstName)
	}
	if data.ID != contact.ID {
		t.Errorf("expected id=%s, got %s", contact.ID, data.ID)
	}
}

func TestContactGet_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cnotfound@example.com")
	vault := ts.createTestVault(t, token, "NotFound Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts/nonexistent-id", "", token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %+v", resp.Error)
	}
}

func TestContactUpdate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cupdate@example.com")
	vault := ts.createTestVault(t, token, "Update Contact Vault")
	contact := ts.createTestContact(t, token, vault.ID, "OldName")

	rec := ts.doRequest(http.MethodPut, "/api/vaults/"+vault.ID+"/contacts/"+contact.ID,
		`{"first_name":"NewName","last_name":"Updated"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data contactData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse contact data: %v", err)
	}
	if data.FirstName != "NewName" {
		t.Errorf("expected first_name=NewName, got %s", data.FirstName)
	}
	if data.LastName != "Updated" {
		t.Errorf("expected last_name=Updated, got %s", data.LastName)
	}
}

func TestContactDelete_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cdelete@example.com")
	vault := ts.createTestVault(t, token, "Delete Contact Vault")
	contact := ts.createTestContact(t, token, vault.ID, "DeleteMe")

	rec := ts.doRequest(http.MethodDelete, "/api/vaults/"+vault.ID+"/contacts/"+contact.ID, "", token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts/"+contact.ID, "", token)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestContactList_WithSearch(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "csearch@example.com")
	vault := ts.createTestVault(t, token, "Search Vault")
	ts.createTestContact(t, token, vault.ID, "Alice")
	ts.createTestContact(t, token, vault.ID, "Bob")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts?search=Alice", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var contacts []contactData
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1 contact matching 'Alice', got %d", len(contacts))
	}
	if len(contacts) > 0 && contacts[0].FirstName != "Alice" {
		t.Errorf("expected first_name=Alice, got %s", contacts[0].FirstName)
	}
}

func TestContactList_VaultAccessForbidden(t *testing.T) {
	ts := setupTestServer(t)
	token1, _ := ts.registerTestUser(t, "owner@example.com")
	vault := ts.createTestVault(t, token1, "Private Vault")

	token2, _ := ts.registerTestUser(t, "intruder@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts", "", token2)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactList_Pagination(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "cpage@example.com")
	vault := ts.createTestVault(t, token, "Pagination Vault")

	ts.createTestContact(t, token, vault.ID, "Contact1")
	ts.createTestContact(t, token, vault.ID, "Contact2")
	ts.createTestContact(t, token, vault.ID, "Contact3")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts?page=1&per_page=2", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var contacts []contactData
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts on page 1, got %d", len(contacts))
	}
	if resp.Meta == nil {
		t.Fatal("expected meta")
	}
	if resp.Meta.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Meta.Total)
	}
	if resp.Meta.TotalPages != 2 {
		t.Errorf("expected total_pages=2, got %d", resp.Meta.TotalPages)
	}

	rec2 := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/contacts?page=2&per_page=2", "", token)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	resp2 := parseResponse(t, rec2)
	var contacts2 []contactData
	if err := json.Unmarshal(resp2.Data, &contacts2); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts2) != 1 {
		t.Errorf("expected 1 contact on page 2, got %d", len(contacts2))
	}
}

// ==================== Notes ====================

func TestNoteCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "note-create@example.com")
	vault := ts.createTestVault(t, token, "Note Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/notes",
		`{"title":"Test Note","body":"Note body"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var itemData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &itemData); err != nil {
		t.Fatalf("failed to parse note data: %v", err)
	}
	if itemData["title"] != "Test Note" {
		t.Errorf("expected title=Test Note, got %v", itemData["title"])
	}
	if itemData["body"] != "Note body" {
		t.Errorf("expected body=Note body, got %v", itemData["body"])
	}
}

func TestNoteList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "note-list@example.com")
	vault := ts.createTestVault(t, token, "Note List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/notes"

	ts.doRequest(http.MethodPost, basePath, `{"title":"Note 1","body":"Body 1"}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"title":"Note 2","body":"Body 2"}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to parse note list: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 notes, got %d", len(items))
	}
}

func TestNoteUpdate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "note-update@example.com")
	vault := ts.createTestVault(t, token, "Note Update Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/notes"

	createRec := ts.doRequest(http.MethodPost, basePath, `{"title":"Old Title","body":"Old body"}`, token)
	createResp := parseResponse(t, createRec)
	var noteData map[string]interface{}
	json.Unmarshal(createResp.Data, &noteData)
	noteID := fmt.Sprintf("%v", noteData["id"])

	rec := ts.doRequest(http.MethodPut, basePath+"/"+noteID,
		`{"title":"New Title","body":"New body"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var updated map[string]interface{}
	json.Unmarshal(resp.Data, &updated)
	if updated["title"] != "New Title" {
		t.Errorf("expected title=New Title, got %v", updated["title"])
	}
	if updated["body"] != "New body" {
		t.Errorf("expected body=New body, got %v", updated["body"])
	}
}

func TestNoteDelete_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "note-delete@example.com")
	vault := ts.createTestVault(t, token, "Note Delete Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/notes"

	createRec := ts.doRequest(http.MethodPost, basePath, `{"title":"Delete Me","body":"Bye"}`, token)
	createResp := parseResponse(t, createRec)
	var noteData map[string]interface{}
	json.Unmarshal(createResp.Data, &noteData)
	noteID := fmt.Sprintf("%v", noteData["id"])

	rec := ts.doRequest(http.MethodDelete, basePath+"/"+noteID, "", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	listRec := ts.doRequest(http.MethodGet, basePath, "", token)
	listResp := parseResponse(t, listRec)
	var items []json.RawMessage
	json.Unmarshal(listResp.Data, &items)
	if len(items) != 0 {
		t.Errorf("expected 0 notes after delete, got %d", len(items))
	}
}

// ==================== Tasks ====================

func TestTaskCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "task-create@example.com")
	vault := ts.createTestVault(t, token, "Task Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/tasks",
		`{"label":"Test Task","description":"Task desc"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestTaskList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "task-list@example.com")
	vault := ts.createTestVault(t, token, "Task List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/tasks"

	ts.doRequest(http.MethodPost, basePath, `{"label":"Task 1","description":"Desc 1"}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"label":"Task 2","description":"Desc 2"}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to parse task list: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(items))
	}
}

func TestTaskToggle_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "task-toggle@example.com")
	vault := ts.createTestVault(t, token, "Task Toggle Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/tasks"

	createRec := ts.doRequest(http.MethodPost, basePath, `{"label":"Toggle Task","description":"Toggle desc"}`, token)
	createResp := parseResponse(t, createRec)
	var taskData map[string]interface{}
	json.Unmarshal(createResp.Data, &taskData)
	taskID := fmt.Sprintf("%v", taskData["id"])

	rec := ts.doRequest(http.MethodPut, basePath+"/"+taskID+"/toggle", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

// ==================== Reminders ====================

func TestReminderCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "reminder-create@example.com")
	vault := ts.createTestVault(t, token, "Reminder Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/reminders",
		`{"label":"Birthday","day":15,"month":6,"type":"one_time"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestReminderList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "reminder-list@example.com")
	vault := ts.createTestVault(t, token, "Reminder List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/reminders"

	ts.doRequest(http.MethodPost, basePath, `{"label":"Reminder 1","day":1,"month":1,"type":"one_time"}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"label":"Reminder 2","day":2,"month":2,"type":"one_time"}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to parse reminder list: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 reminders, got %d", len(items))
	}
}

// ==================== Calls ====================

func TestCallCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "call-create@example.com")
	vault := ts.createTestVault(t, token, "Call Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/calls",
		`{"called_at":"2024-01-15T10:00:00Z","description":"Quick chat","type":"audio","who_initiated":"me"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCallList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "call-list@example.com")
	vault := ts.createTestVault(t, token, "Call List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/calls"

	ts.doRequest(http.MethodPost, basePath, `{"called_at":"2024-01-15T10:00:00Z","description":"Call 1","type":"audio","who_initiated":"me"}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"called_at":"2024-01-16T10:00:00Z","description":"Call 2","type":"audio","who_initiated":"me"}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to parse call list: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 calls, got %d", len(items))
	}
}

// ==================== Goals ====================

func TestGoalCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "goal-create@example.com")
	vault := ts.createTestVault(t, token, "Goal Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/goals",
		`{"name":"Exercise daily"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestGoalGet_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "goal-get@example.com")
	vault := ts.createTestVault(t, token, "Goal Get Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/goals"

	createRec := ts.doRequest(http.MethodPost, basePath, `{"name":"Exercise daily"}`, token)
	createResp := parseResponse(t, createRec)
	var goalData map[string]interface{}
	json.Unmarshal(createResp.Data, &goalData)
	goalID := fmt.Sprintf("%v", goalData["id"])

	rec := ts.doRequest(http.MethodGet, basePath+"/"+goalID, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var fetched map[string]interface{}
	json.Unmarshal(resp.Data, &fetched)
	if fetched["name"] != "Exercise daily" {
		t.Errorf("expected name=Exercise daily, got %v", fetched["name"])
	}
}

// ==================== Loans ====================

func TestLoanCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "loan-create@example.com")
	vault := ts.createTestVault(t, token, "Loan Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/loans",
		`{"name":"Book loan","type":"debt","amount_lent":50}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestLoanToggle_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "loan-toggle@example.com")
	vault := ts.createTestVault(t, token, "Loan Toggle Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/loans"

	createRec := ts.doRequest(http.MethodPost, basePath, `{"name":"Book loan","type":"debt","amount_lent":50}`, token)
	createResp := parseResponse(t, createRec)
	var loanData map[string]interface{}
	json.Unmarshal(createResp.Data, &loanData)
	loanID := fmt.Sprintf("%v", loanData["id"])

	rec := ts.doRequest(http.MethodPut, basePath+"/"+loanID+"/toggle", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

// ==================== Pets ====================

func TestPetCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "pet-create@example.com")
	vault := ts.createTestVault(t, token, "Pet Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/pets",
		`{"name":"Buddy","pet_category_id":1}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestPetList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "pet-list@example.com")
	vault := ts.createTestVault(t, token, "Pet List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")
	basePath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/pets"

	ts.doRequest(http.MethodPost, basePath, `{"name":"Buddy","pet_category_id":1}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"name":"Max","pet_category_id":1}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var items []json.RawMessage
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("failed to parse pet list: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 pets, got %d", len(items))
	}
}

// ==================== Addresses ====================

func TestAddressCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "address-create@example.com")
	vault := ts.createTestVault(t, token, "Address Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/addresses",
		`{"street":"123 Main St","city":"Portland","country":"US","address_type_id":1}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

// ==================== Important Dates ====================

func TestImportantDateCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "date-create@example.com")
	vault := ts.createTestVault(t, token, "Date Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/dates",
		`{"label":"Birthday","day":15,"month":6}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

// ==================== Unauthorized ====================

func TestNoteCreate_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "note-unauth@example.com")
	vault := ts.createTestVault(t, token, "Unauth Note Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/notes",
		`{"title":"Test Note","body":"Note body"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Two-Factor Authentication ====================

func TestTwoFactor_Status(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "2fa-status@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/settings/2fa/status", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse 2fa status data: %v", err)
	}
	if data["enabled"] != false {
		t.Errorf("expected enabled=false for new user, got %v", data["enabled"])
	}
}

func TestTwoFactor_EnableFlow(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "2fa-enable@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/2fa/enable", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse 2fa enable data: %v", err)
	}
	if data["secret"] == nil || data["secret"] == "" {
		t.Error("expected non-empty secret")
	}
	if data["recovery_codes"] == nil {
		t.Error("expected recovery_codes in response")
	}
	codes, ok := data["recovery_codes"].([]interface{})
	if !ok || len(codes) == 0 {
		t.Error("expected non-empty recovery_codes array")
	}
}

func TestTwoFactor_Disable_NotEnabled(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "2fa-disable@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/2fa/disable",
		`{"code":"123456"}`, token)

	// 2FA not enabled yet, should return 400 (two_factor_not_set)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

// ==================== Search ====================

func TestSearch_EmptyQuery(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "search-empty@example.com")
	vault := ts.createTestVault(t, token, "Search Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/search?q=", "", token)

	// Empty query returns BadRequest
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestSearch_WithQuery(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "search-query@example.com")
	vault := ts.createTestVault(t, token, "Search Query Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/search?q=John", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestSearch_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "search-unauth@example.com")
	vault := ts.createTestVault(t, token, "Search Unauth Vault")

	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/search?q=test", "", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== vCard ====================

func TestVCard_ExportContact(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vcard-contact@example.com")
	vault := ts.createTestVault(t, token, "VCard Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Alice")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/vcard", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/vcard") {
		t.Errorf("expected Content-Type containing text/vcard, got %s", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "BEGIN:VCARD") {
		t.Error("expected vCard body to contain BEGIN:VCARD")
	}
}

func TestVCard_ExportVault(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "vcard-vault@example.com")
	vault := ts.createTestVault(t, token, "VCard Export Vault")
	ts.createTestContact(t, token, vault.ID, "Bob")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/export", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/vcard") {
		t.Errorf("expected Content-Type containing text/vcard, got %s", contentType)
	}
}

// ==================== Invitations ====================

func TestInvitation_List(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "invite-list@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/settings/invitations", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestInvitation_Create_InvalidEmail(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "invite-invalid@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/invitations",
		`{"email":"","permission":0}`, token)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

// ==================== Avatar ====================

func TestAvatar_GetInitials(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "avatar-initials@example.com")
	vault := ts.createTestVault(t, token, "Avatar Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Charlie")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/avatar", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "image/png") {
		t.Errorf("expected Content-Type containing image/png, got %s", contentType)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty image body")
	}
}

// ==================== 2FA Additional ====================

func TestTwoFactor_Confirm_InvalidCode(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "2fa-confirm-invalid@example.com")

	ts.doRequest(http.MethodPost, "/api/settings/2fa/enable", "", token)

	rec := ts.doRequest(http.MethodPost, "/api/settings/2fa/confirm",
		`{"code":"000000"}`, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestTwoFactor_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/settings/2fa/enable", "", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== File Upload ====================

func setupTestServerWithStorage(t *testing.T) *testServer {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		Server:   config.ServerConfig{Port: "8080", Host: "localhost"},
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"},
		JWT: config.JWTConfig{
			Secret:     "test-secret-key",
			ExpiryHrs:  24,
			RefreshHrs: 168,
		},
		App:     config.AppConfig{Name: "Monica Test", Env: "test", URL: "http://localhost:8080"},
		Storage: config.StorageConfig{UploadDir: t.TempDir(), MaxSize: 10 * 1024 * 1024},
	}
	e := echo.New()
	handlers.RegisterRoutes(e, db, cfg)
	return &testServer{e: e, db: db, cfg: cfg}
}

func TestFileUpload_Success(t *testing.T) {
	ts := setupTestServerWithStorage(t)

	token, _ := ts.registerTestUser(t, "file-upload@example.com")
	vault := ts.createTestVault(t, token, "File Upload Vault")

	pngData := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/files",
		token, "file", "test.png", "image/png", pngData)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestFileUpload_TooLarge(t *testing.T) {
	ts := setupTestServerWithStorage(t)

	token, _ := ts.registerTestUser(t, "file-large@example.com")
	vault := ts.createTestVault(t, token, "File Large Vault")

	largeData := make([]byte, 11*1024*1024)
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/files",
		token, "file", "huge.png", "image/png", largeData)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestFileUpload_InvalidType(t *testing.T) {
	ts := setupTestServerWithStorage(t)

	token, _ := ts.registerTestUser(t, "file-type@example.com")
	vault := ts.createTestVault(t, token, "File Type Vault")

	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/files",
		token, "file", "script.exe", "application/x-executable", []byte("MZ..."))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestFileDownload_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "file-notfound@example.com")
	vault := ts.createTestVault(t, token, "File NotFound Vault")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/files/9999/download", "", token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

// ==================== Invitations Additional ====================

func TestInvitationCreate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "invite-create@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/invitations",
		`{"email":"new-member@example.com","permission":200}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse invitation data: %v", err)
	}
	if data["email"] != "new-member@example.com" {
		t.Errorf("expected email=new-member@example.com, got %v", data["email"])
	}
}

func TestInvitationAccept_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/invitations/accept",
		`{"token":"invalid-uuid-token","first_name":"Test","password":"password123"}`, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}
