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
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
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
	ID             string `json:"id"`
	VaultID        string `json:"vault_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	MiddleName     string `json:"middle_name"`
	Nickname       string `json:"nickname"`
	MaidenName     string `json:"maiden_name"`
	Prefix         string `json:"prefix"`
	Suffix         string `json:"suffix"`
	GenderID       *uint  `json:"gender_id"`
	PronounID      *uint  `json:"pronoun_id"`
	TemplateID     *uint  `json:"template_id"`
	CompanyID      *uint  `json:"company_id"`
	ReligionID     *uint  `json:"religion_id"`
	FileID         *uint  `json:"file_id"`
	JobPosition    string `json:"job_position"`
	Listed         bool   `json:"listed"`
	ShowQuickFacts bool   `json:"show_quick_facts"`
	IsArchived     bool   `json:"is_archived"`
	IsFavorite     bool   `json:"is_favorite"`
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

	settingSvc := services.NewSystemSettingService(db)
	if err := services.SeedSettingsFromEnv(settingSvc, cfg); err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	e := echo.New()
	handlers.RegisterRoutes(e, db, cfg, "test")

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
		`{"first_name":"Jane","last_name":"Smith","middle_name":"Marie","nickname":"Janey","maiden_name":"Jones","prefix":"Ms.","suffix":"PhD"}`, token)

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
	if data.MiddleName != "Marie" {
		t.Errorf("expected middle_name=Marie, got %s", data.MiddleName)
	}
	if data.Nickname != "Janey" {
		t.Errorf("expected nickname=Janey, got %s", data.Nickname)
	}
	if data.MaidenName != "Jones" {
		t.Errorf("expected maiden_name=Jones, got %s", data.MaidenName)
	}
	if data.Prefix != "Ms." {
		t.Errorf("expected prefix=Ms., got %s", data.Prefix)
	}
	if data.Suffix != "PhD" {
		t.Errorf("expected suffix=PhD, got %s", data.Suffix)
	}
	if data.VaultID != vault.ID {
		t.Errorf("expected vault_id=%s, got %s", vault.ID, data.VaultID)
	}
	if data.ID == "" {
		t.Error("expected non-empty contact ID")
	}
	if !data.Listed {
		t.Error("expected listed=true by default")
	}
	if data.IsArchived {
		t.Error("expected is_archived=false by default")
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
		`{"first_name":"NewName","last_name":"Updated","middle_name":"Mid","maiden_name":"OldFamily","prefix":"Dr.","suffix":"Sr."}`, token)

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
	if data.MiddleName != "Mid" {
		t.Errorf("expected middle_name=Mid, got %s", data.MiddleName)
	}
	if data.MaidenName != "OldFamily" {
		t.Errorf("expected maiden_name=OldFamily, got %s", data.MaidenName)
	}
	if data.Prefix != "Dr." {
		t.Errorf("expected prefix=Dr., got %s", data.Prefix)
	}
	if data.Suffix != "Sr." {
		t.Errorf("expected suffix=Sr., got %s", data.Suffix)
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

func TestImportantDateCreateLunar_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "date-lunar@example.com")
	vault := ts.createTestVault(t, token, "Lunar Date Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/dates",
		`{"label":"Lunar Birthday","calendar_type":"lunar","original_day":15,"original_month":1,"original_year":2025}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse date data: %v", err)
	}
	if data["calendar_type"] != "lunar" {
		t.Errorf("expected calendar_type='lunar', got %v", data["calendar_type"])
	}
	if data["original_day"] == nil {
		t.Error("expected original_day to be set")
	}
	if data["original_month"] == nil {
		t.Error("expected original_month to be set")
	}
}

func TestReminderCreateLunar_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "reminder-lunar@example.com")
	vault := ts.createTestVault(t, token, "Lunar Reminder Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/reminders",
		`{"label":"Lunar Bday","type":"recurring_year","calendar_type":"lunar","original_day":15,"original_month":1,"original_year":2025}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse reminder data: %v", err)
	}
	if data["calendar_type"] != "lunar" {
		t.Errorf("expected calendar_type='lunar', got %v", data["calendar_type"])
	}
	if data["original_day"] == nil {
		t.Error("expected original_day to be set")
	}
	if data["day"] == nil {
		t.Error("expected converted gregorian day to be set")
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
	settingSvc := services.NewSystemSettingService(db)
	if err := services.SeedSettingsFromEnv(settingSvc, cfg); err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}
	e := echo.New()
	handlers.RegisterRoutes(e, db, cfg, "test")
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

func TestContactPhotoUpload_CreatesFeedEntry(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "photo-feed@example.com")
	vault := ts.createTestVault(t, token, "Photo Feed Vault")
	contact := ts.createTestContact(t, token, vault.ID, "FeedPhoto")

	pngData := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos",
		token, "file", "trip.png", "image/png", pngData)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload photo failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/feed", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get feed failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "file_uploaded") {
		t.Errorf("expected feed to contain 'file_uploaded' action, got: %s", body)
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

// ==================== Contact Photo Delete ====================

func TestContactPhotoList_IncludesAvatar(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "photo-list-avatar@example.com")
	vault := ts.createTestVault(t, token, "Photo List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "PhotoListAvatar")

	// 1) Upload a regular photo via POST
	pngData := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos",
		token, "file", "photo.png", "image/png", pngData)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload photo failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 2) Upload avatar via PUT (multipart)
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	fakeAvatar := make([]byte, 100)
	copy(fakeAvatar, pngHeader)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="file"; filename="avatar.png"`)
	mh.Set("Content-Type", "image/png")
	part, err := mw.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	part.Write(fakeAvatar)
	mw.Close()
	avatarPath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/avatar"
	req := httptest.NewRequest(http.MethodPut, avatarPath, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	ts.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT avatar: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3) List photos — should include both the regular photo AND the avatar
	rec = ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list photos: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var photos []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &photos); err != nil {
		t.Fatalf("failed to parse photos: %v", err)
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos (1 photo + 1 avatar), got %d", len(photos))
	}

	// Verify we have both types
	typeSet := map[string]bool{}
	for _, p := range photos {
		if ft, ok := p["type"].(string); ok {
			typeSet[ft] = true
		}
	}
	if !typeSet["photo"] {
		t.Error("expected a photo-type file in list")
	}
	if !typeSet["avatar"] {
		t.Error("expected an avatar-type file in list")
	}

	// 4) Delete the avatar via photo delete endpoint — should also unset contact.FileID
	avatarID := ""
	for _, p := range photos {
		if ft, ok := p["type"].(string); ok && ft == "avatar" {
			avatarID = fmt.Sprintf("%.0f", p["id"].(float64))
		}
	}
	if avatarID == "" {
		t.Fatal("could not find avatar ID in photos list")
	}
	rec = ts.doRequest(http.MethodDelete,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos/"+avatarID, "", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete avatar photo: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5) List photos again — should only have the regular photo
	rec = ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list photos after delete: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var photosAfter []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &photosAfter); err != nil {
		t.Fatalf("failed to parse photos after delete: %v", err)
	}
	if len(photosAfter) != 1 {
		t.Fatalf("expected 1 photo after avatar delete, got %d", len(photosAfter))
	}
	if ft, ok := photosAfter[0]["type"].(string); !ok || ft != "photo" {
		t.Errorf("expected remaining photo to be type 'photo', got '%v'", photosAfter[0]["type"])
	}

	// 6) Verify avatar is unset on the contact
	var contactAfter models.Contact
	if err := ts.db.First(&contactAfter, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("query contact: %v", err)
	}
	if contactAfter.FileID != nil {
		t.Error("expected contact.FileID to be nil after avatar deletion")
	}
}

func TestContactPhotoDelete_Success(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "photo-del@example.com")
	vault := ts.createTestVault(t, token, "Photo Del Vault")
	contact := ts.createTestContact(t, token, vault.ID, "PhotoDel")

	pngData := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos",
		token, "file", "test.png", "image/png", pngData)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload photo failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var fileData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &fileData); err != nil {
		t.Fatalf("failed to parse file data: %v", err)
	}
	photoID := fmt.Sprintf("%.0f", fileData["id"].(float64))

	rec = ts.doRequest(http.MethodDelete,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos/"+photoID, "", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactPhotoDelete_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "photo-del-nf@example.com")
	vault := ts.createTestVault(t, token, "Photo Del NF Vault")
	contact := ts.createTestContact(t, token, vault.ID, "PhotoDelNF")

	rec := ts.doRequest(http.MethodDelete,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/photos/9999", "", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Contact Document List & Delete ====================

func TestContactDocumentList(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "doc-list@example.com")
	vault := ts.createTestVault(t, token, "Doc List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "DocList")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/documents", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestContactDocumentDelete_Success(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "doc-del@example.com")
	vault := ts.createTestVault(t, token, "Doc Del Vault")
	contact := ts.createTestContact(t, token, vault.ID, "DocDel")

	pdfData := []byte("%PDF-1.4 test content")
	rec := ts.doMultipartUpload(t,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/documents",
		token, "file", "test.pdf", "application/pdf", pdfData)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload document failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var fileData map[string]interface{}
	if err := json.Unmarshal(resp.Data, &fileData); err != nil {
		t.Fatalf("failed to parse file data: %v", err)
	}
	docID := fmt.Sprintf("%.0f", fileData["id"].(float64))

	rec = ts.doRequest(http.MethodDelete,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/documents/"+docID, "", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactDocumentDelete_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "doc-del-nf@example.com")
	vault := ts.createTestVault(t, token, "Doc Del NF Vault")
	contact := ts.createTestContact(t, token, vault.ID, "DocDelNF")

	rec := ts.doRequest(http.MethodDelete,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/documents/9999", "", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Contact Label Update ====================

func TestContactLabelUpdate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "label-update@example.com")
	vault := ts.createTestVault(t, token, "Label Update Vault")
	contact := ts.createTestContact(t, token, vault.ID, "LabelUpdate")

	labelRec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/settings/labels",
		`{"name":"Label1","bg_color":"bg-blue-200","text_color":"text-blue-700"}`, token)
	if labelRec.Code != http.StatusCreated {
		t.Fatalf("create label failed: status=%d body=%s", labelRec.Code, labelRec.Body.String())
	}
	labelResp := parseResponse(t, labelRec)
	var label1 map[string]interface{}
	if err := json.Unmarshal(labelResp.Data, &label1); err != nil {
		t.Fatalf("failed to parse label data: %v", err)
	}
	label1ID := fmt.Sprintf("%.0f", label1["id"].(float64))

	labelRec2 := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/settings/labels",
		`{"name":"Label2","bg_color":"bg-red-200","text_color":"text-red-700"}`, token)
	if labelRec2.Code != http.StatusCreated {
		t.Fatalf("create label2 failed: status=%d body=%s", labelRec2.Code, labelRec2.Body.String())
	}
	labelResp2 := parseResponse(t, labelRec2)
	var label2 map[string]interface{}
	if err := json.Unmarshal(labelResp2.Data, &label2); err != nil {
		t.Fatalf("failed to parse label2 data: %v", err)
	}
	label2ID := fmt.Sprintf("%.0f", label2["id"].(float64))

	addRec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/labels",
		`{"label_id":`+label1ID+`}`, token)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add label failed: status=%d body=%s", addRec.Code, addRec.Body.String())
	}
	addResp := parseResponse(t, addRec)
	var addedLabel map[string]interface{}
	if err := json.Unmarshal(addResp.Data, &addedLabel); err != nil {
		t.Fatalf("failed to parse added label data: %v", err)
	}
	pivotID := fmt.Sprintf("%.0f", addedLabel["id"].(float64))

	updateRec := ts.doRequest(http.MethodPut,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/labels/"+pivotID,
		`{"label_id":`+label2ID+`}`, token)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	updateResp := parseResponse(t, updateRec)
	if !updateResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestContactLabelUpdate_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "label-upd-nf@example.com")
	vault := ts.createTestVault(t, token, "Label Upd NF Vault")
	contact := ts.createTestContact(t, token, vault.ID, "LabelUpdNF")

	rec := ts.doRequest(http.MethodPut,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/labels/9999",
		`{"label_id":1}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Contact Companies List ====================

func TestContactCompaniesList(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "companies-list@example.com")
	vault := ts.createTestVault(t, token, "Companies List Vault")
	contact := ts.createTestContact(t, token, vault.ID, "CompList")

	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/companies/list", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

// ==================== Contact List By Label ====================

func TestContactListByLabel_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")
	vault := ts.createTestVault(t, token, "Test Vault")
	contact := ts.createTestContact(t, token, vault.ID, "John")

	ts.db.Exec("INSERT INTO labels (vault_id, name, slug, bg_color, text_color) VALUES (?, 'Family', 'family', 'bg-zinc-200', 'text-zinc-700')", vault.ID)
	var labelID uint
	ts.db.Raw("SELECT id FROM labels WHERE vault_id = ? AND name = 'Family'", vault.ID).Scan(&labelID)
	ts.db.Exec("INSERT INTO contact_label (label_id, contact_id) VALUES (?, ?)", labelID, contact.ID)

	rec := ts.doRequest(http.MethodGet, fmt.Sprintf("/api/vaults/%s/contacts/labels/%d", vault.ID, labelID), "", token)
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
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].ID != contact.ID {
		t.Errorf("expected contact ID %s, got %s", contact.ID, contacts[0].ID)
	}
}

func TestContactListByLabel_NoResults(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")
	vault := ts.createTestVault(t, token, "Test Vault")

	ts.db.Exec("INSERT INTO labels (vault_id, name, slug, bg_color, text_color) VALUES (?, 'Empty', 'empty', 'bg-zinc-200', 'text-zinc-700')", vault.ID)
	var labelID uint
	ts.db.Raw("SELECT id FROM labels WHERE vault_id = ? AND name = 'Empty'", vault.ID).Scan(&labelID)

	rec := ts.doRequest(http.MethodGet, fmt.Sprintf("/api/vaults/%s/contacts/labels/%d", vault.ID, labelID), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var contacts []contactData
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		t.Fatalf("failed to parse contacts: %v", err)
	}
	if len(contacts) != 0 {
		t.Fatalf("expected 0 contacts, got %d", len(contacts))
	}
}

// ==================== RelationshipType Sub-resource CRUD ====================

func TestRelationshipType_Create(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var groupTypeID uint
	ts.db.Raw("SELECT id FROM relationship_group_types LIMIT 1").Scan(&groupTypeID)
	if groupTypeID == 0 {
		t.Fatal("no seeded relationship group type found")
	}

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types", groupTypeID),
		`{"name":"Mentor","name_reverse_relationship":"Mentee"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestRelationshipType_Update(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var groupTypeID uint
	ts.db.Raw("SELECT id FROM relationship_group_types LIMIT 1").Scan(&groupTypeID)

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types", groupTypeID),
		`{"name":"Mentor","name_reverse_relationship":"Mentee"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	rec = ts.doRequest(http.MethodPut,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types/%d", groupTypeID, created.ID),
		`{"name":"Coach","name_reverse_relationship":"Coachee"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRelationshipType_Delete(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var groupTypeID uint
	ts.db.Raw("SELECT id FROM relationship_group_types LIMIT 1").Scan(&groupTypeID)

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types", groupTypeID),
		`{"name":"Mentor","name_reverse_relationship":"Mentee"}`, token)
	resp := parseResponse(t, rec)
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	rec = ts.doRequest(http.MethodDelete,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types/%d", groupTypeID, created.ID),
		"", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRelationshipType_DeleteUndeletable(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var groupTypeID, typeID uint
	ts.db.Raw("SELECT relationship_group_type_id, id FROM relationship_types WHERE can_be_deleted = false LIMIT 1").Row().Scan(&groupTypeID, &typeID)
	if typeID == 0 {
		t.Skip("no undeletable relationship type found in seed data")
	}

	rec := ts.doRequest(http.MethodDelete,
		fmt.Sprintf("/api/settings/personalize/relationship-types/%d/types/%d", groupTypeID, typeID),
		"", token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRelationshipType_GroupNotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	rec := ts.doRequest(http.MethodPost,
		"/api/settings/personalize/relationship-types/99999/types",
		`{"name":"Mentor","name_reverse_relationship":"Mentee"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ==================== CallReason Sub-resource CRUD ====================

func TestCallReason_Create(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var callReasonTypeID uint
	ts.db.Raw("SELECT id FROM call_reason_types LIMIT 1").Scan(&callReasonTypeID)
	if callReasonTypeID == 0 {
		t.Fatal("no seeded call reason type found")
	}

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/call-reasons/%d/reasons", callReasonTypeID),
		`{"label":"Follow-up call"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestCallReason_Update(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var callReasonTypeID uint
	ts.db.Raw("SELECT id FROM call_reason_types LIMIT 1").Scan(&callReasonTypeID)

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/call-reasons/%d/reasons", callReasonTypeID),
		`{"label":"Follow-up call"}`, token)
	resp := parseResponse(t, rec)
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	rec = ts.doRequest(http.MethodPut,
		fmt.Sprintf("/api/settings/personalize/call-reasons/%d/reasons/%d", callReasonTypeID, created.ID),
		`{"label":"Check-in call"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCallReason_Delete(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	var callReasonTypeID uint
	ts.db.Raw("SELECT id FROM call_reason_types LIMIT 1").Scan(&callReasonTypeID)

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/settings/personalize/call-reasons/%d/reasons", callReasonTypeID),
		`{"label":"Follow-up call"}`, token)
	resp := parseResponse(t, rec)
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &created)

	rec = ts.doRequest(http.MethodDelete,
		fmt.Sprintf("/api/settings/personalize/call-reasons/%d/reasons/%d", callReasonTypeID, created.ID),
		"", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCallReason_TypeNotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "test@example.com")

	rec := ts.doRequest(http.MethodPost,
		"/api/settings/personalize/call-reasons/99999/reasons",
		`{"label":"Test"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContactQuickSearch_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "search-test@example.com")
	vault := ts.createTestVault(t, token, "Search Vault")
	ts.createTestContact(t, token, vault.ID, "Alice")
	ts.createTestContact(t, token, vault.ID, "Bob")
	ts.createTestContact(t, token, vault.ID, "Charlie")

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/search/contacts", vault.ID),
		`{"search_term":"Alice"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var results []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp.Data, &results); err != nil {
		t.Fatalf("failed to parse search results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID == "" {
		t.Error("expected non-empty ID")
	}

	rec = ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/search/contacts", vault.ID),
		`{"search_term":"Doe"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	json.Unmarshal(resp.Data, &results)
	if len(results) != 3 {
		t.Fatalf("expected 3 results for 'Doe' (all have last_name Doe), got %d", len(results))
	}
}

func TestContactQuickSearch_EmptyTerm(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "search-empty@example.com")
	vault := ts.createTestVault(t, token, "Search Vault 2")

	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/search/contacts", vault.ID),
		`{"search_term":""}`, token)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty search_term, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTaskListCompleted_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "task-completed@example.com")
	vault := ts.createTestVault(t, token, "Task Vault")
	contact := ts.createTestContact(t, token, vault.ID, "TaskContact")

	basePath := fmt.Sprintf("/api/vaults/%s/contacts/%s/tasks", vault.ID, contact.ID)

	rec := ts.doRequest(http.MethodPost, basePath, `{"label":"Task 1"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task 1 failed: %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var task1 struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &task1)

	rec = ts.doRequest(http.MethodPost, basePath, `{"label":"Task 2"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task 2 failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPost, basePath, `{"label":"Task 3"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task 3 failed: %d %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var task3 struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &task3)

	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("%s/%d/toggle", basePath, task1.ID), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle task 1 failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("%s/%d/toggle", basePath, task3.ID), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("toggle task 3 failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, basePath+"/completed", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list completed tasks failed: %d %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var completed []struct {
		ID        uint   `json:"id"`
		Label     string `json:"label"`
		Completed bool   `json:"completed"`
	}
	if err := json.Unmarshal(resp.Data, &completed); err != nil {
		t.Fatalf("failed to parse completed tasks: %v", err)
	}
	if len(completed) != 2 {
		t.Fatalf("expected 2 completed tasks, got %d", len(completed))
	}
	for _, task := range completed {
		if !task.Completed {
			t.Errorf("expected task %d to be completed", task.ID)
		}
	}

	rec = ts.doRequest(http.MethodGet, basePath, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list all tasks failed: %d %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var allTasks []struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &allTasks)
	if len(allTasks) != 3 {
		t.Fatalf("expected 3 total tasks, got %d", len(allTasks))
	}
}

func TestCurrencyToggle_Success(t *testing.T) {
	ts := setupTestServer(t)
	if err := models.SeedCurrencies(ts.db); err != nil {
		t.Fatalf("seed currencies: %v", err)
	}
	token, _ := ts.registerTestUser(t, "cur-toggle@test.com")

	rec := ts.doRequest(http.MethodGet, "/api/currencies", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var currencies []struct {
		ID   uint   `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(resp.Data, &currencies)
	if len(currencies) == 0 {
		t.Fatal("expected currencies to exist after seed")
	}
	currID := currencies[0].ID

	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("/api/settings/personalize/currencies/%d/toggle", currID), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on first toggle, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("/api/settings/personalize/currencies/%d/toggle", currID), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on second toggle, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCurrencyEnableAll_Success(t *testing.T) {
	ts := setupTestServer(t)
	if err := models.SeedCurrencies(ts.db); err != nil {
		t.Fatalf("seed currencies: %v", err)
	}
	token, _ := ts.registerTestUser(t, "cur-enable@test.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/personalize/currencies/enable-all", "", token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCurrencyDisableAll_Success(t *testing.T) {
	ts := setupTestServer(t)
	if err := models.SeedCurrencies(ts.db); err != nil {
		t.Fatalf("seed currencies: %v", err)
	}
	token, _ := ts.registerTestUser(t, "cur-disable@test.com")

	rec := ts.doRequest(http.MethodDelete, "/api/settings/personalize/currencies/disable-all", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostGet_ViewCount(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "post-view@test.com")
	vault := ts.createTestVault(t, token, "View Vault")
	journalID := ts.createTestJournal(t, token, vault.ID, "View Journal")
	postID := ts.createTestPost(t, token, vault.ID, journalID, "View Post")

	path := fmt.Sprintf("/api/vaults/%s/journals/%d/posts/%d", vault.ID, journalID, postID)

	rec := ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var post struct {
		ViewCount int `json:"view_count"`
	}
	json.Unmarshal(resp.Data, &post)
	if post.ViewCount < 1 {
		t.Fatalf("expected view_count >= 1, got %d", post.ViewCount)
	}
	firstCount := post.ViewCount

	rec = ts.doRequest(http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	json.Unmarshal(resp.Data, &post)
	if post.ViewCount != firstCount+1 {
		t.Fatalf("expected view_count %d, got %d", firstCount+1, post.ViewCount)
	}
}

func TestPostUpdate_WithContacts(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "post-contacts@test.com")
	vault := ts.createTestVault(t, token, "PC Vault")
	journalID := ts.createTestJournal(t, token, vault.ID, "PC Journal")
	postID := ts.createTestPost(t, token, vault.ID, journalID, "PC Post")
	contact1 := ts.createTestContact(t, token, vault.ID, "Alice")
	contact2 := ts.createTestContact(t, token, vault.ID, "Bob")

	path := fmt.Sprintf("/api/vaults/%s/journals/%d/posts/%d", vault.ID, journalID, postID)

	body := fmt.Sprintf(`{"title":"Updated","written_at":"2024-01-01T00:00:00Z","contact_ids":["%s","%s"]}`, contact1.ID, contact2.ID)
	rec := ts.doRequest(http.MethodPut, path, body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var postResp struct {
		Contacts []struct {
			ID string `json:"id"`
		} `json:"contacts"`
	}
	json.Unmarshal(resp.Data, &postResp)
	if len(postResp.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(postResp.Contacts))
	}

	body = fmt.Sprintf(`{"title":"Updated Again","written_at":"2024-01-01T00:00:00Z","contact_ids":["%s"]}`, contact1.ID)
	rec = ts.doRequest(http.MethodPut, path, body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	json.Unmarshal(resp.Data, &postResp)
	if len(postResp.Contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(postResp.Contacts))
	}
}

func TestNotificationCreate_HasToken(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "notif-create@test.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/notifications",
		`{"type":"email","label":"Test","content":"test@example.com"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var channel struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &channel)
	if channel.ID == 0 {
		t.Fatal("expected channel ID > 0")
	}

	var dbChannel models.UserNotificationChannel
	if err := ts.db.Where("id = ? AND user_id = ?", channel.ID, auth.User.ID).First(&dbChannel).Error; err != nil {
		t.Fatalf("failed to query channel from DB: %v", err)
	}
	if dbChannel.VerificationToken == nil || *dbChannel.VerificationToken == "" {
		t.Fatal("expected VerificationToken to be set, got nil/empty")
	}
}

func TestNotificationVerify_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "notif-verify@test.com")
	ts.createTestVault(t, token, "Notif Vault")

	rec := ts.doRequest(http.MethodPost, "/api/settings/notifications",
		`{"type":"email","label":"Verify Me","content":"verify@example.com"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var channel struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &channel)

	var dbChannel models.UserNotificationChannel
	if err := ts.db.Where("id = ? AND user_id = ?", channel.ID, auth.User.ID).First(&dbChannel).Error; err != nil {
		t.Fatalf("failed to query channel: %v", err)
	}
	if dbChannel.VerificationToken == nil {
		t.Fatal("expected VerificationToken to be set")
	}
	verifyToken := *dbChannel.VerificationToken

	rec = ts.doRequest(http.MethodGet, fmt.Sprintf("/api/settings/notifications/%d/verify/%s", channel.ID, verifyToken), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := ts.db.Where("id = ?", channel.ID).First(&dbChannel).Error; err != nil {
		t.Fatalf("failed to reload channel: %v", err)
	}
	if dbChannel.VerifiedAt == nil {
		t.Fatal("expected VerifiedAt to be set after verification")
	}
	if !dbChannel.Active {
		t.Fatal("expected channel to be active after verification")
	}
}

func TestNotificationUpdate_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "notif-update@test.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/notifications",
		`{"type":"shoutrrr","label":"My Bot","content":"telegram://token@telegram?channels=123"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var channel struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &channel)

	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("/api/settings/notifications/%d", channel.ID),
		`{"label":"Updated Bot","content":"telegram://token@telegram?channels=456"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var updated struct {
		Label   string `json:"label"`
		Content string `json:"content"`
	}
	json.Unmarshal(resp.Data, &updated)
	if updated.Label != "Updated Bot" {
		t.Errorf("expected label 'Updated Bot', got '%s'", updated.Label)
	}
	if updated.Content != "telegram://token@telegram?channels=456" {
		t.Errorf("expected updated content, got '%s'", updated.Content)
	}
}

func TestNotificationUpdate_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "notif-update-nf@test.com")

	rec := ts.doRequest(http.MethodPut, "/api/settings/notifications/99999",
		`{"label":"X","content":"x@example.com"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotificationUpdate_EmailResetsVerification(t *testing.T) {
	ts := setupTestServer(t)
	token, auth := ts.registerTestUser(t, "notif-update-reverify@test.com")
	ts.createTestVault(t, token, "V")

	rec := ts.doRequest(http.MethodPost, "/api/settings/notifications",
		`{"type":"email","label":"Email","content":"original@example.com"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var channel struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &channel)

	var dbChannel models.UserNotificationChannel
	ts.db.Where("id = ? AND user_id = ?", channel.ID, auth.User.ID).First(&dbChannel)
	verifyToken := *dbChannel.VerificationToken
	ts.doRequest(http.MethodGet, fmt.Sprintf("/api/settings/notifications/%d/verify/%s", channel.ID, verifyToken), "", token)

	ts.db.First(&dbChannel, channel.ID)
	if dbChannel.VerifiedAt == nil {
		t.Fatal("expected channel to be verified before update")
	}

	rec = ts.doRequest(http.MethodPut, fmt.Sprintf("/api/settings/notifications/%d", channel.ID),
		`{"label":"Email","content":"changed@example.com"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var updated struct {
		Active     bool    `json:"active"`
		VerifiedAt *string `json:"verified_at"`
	}
	json.Unmarshal(resp.Data, &updated)
	if updated.VerifiedAt != nil {
		t.Error("expected verified_at to be null after content change")
	}
	if updated.Active {
		t.Error("expected channel to be inactive after content change")
	}
}

func (ts *testServer) createTestJournal(t *testing.T, token, vaultID, name string) uint {
	t.Helper()
	body := `{"name":"` + name + `"}`
	rec := ts.doRequest(http.MethodPost, "/api/vaults/"+vaultID+"/journals", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create journal failed: %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &data)
	return data.ID
}

func (ts *testServer) createTestPost(t *testing.T, token, vaultID string, journalID uint, title string) uint {
	t.Helper()
	body := fmt.Sprintf(`{"title":"%s","written_at":"2024-01-01T00:00:00Z"}`, title)
	path := fmt.Sprintf("/api/vaults/%s/journals/%d/posts", vaultID, journalID)
	rec := ts.doRequest(http.MethodPost, path, body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create post failed: %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &data)
	return data.ID
}

func TestPostTagList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "pt-handler@test.com")
	vault := ts.createTestVault(t, token, "PT Vault")
	journalID := ts.createTestJournal(t, token, vault.ID, "Test Journal")
	postID := ts.createTestPost(t, token, vault.ID, journalID, "Test Post")

	basePath := fmt.Sprintf("/api/vaults/%s/journals/%d/posts/%d/tags", vault.ID, journalID, postID)

	ts.doRequest(http.MethodPost, basePath, `{"name":"alpha"}`, token)
	ts.doRequest(http.MethodPost, basePath, `{"name":"beta"}`, token)

	rec := ts.doRequest(http.MethodGet, basePath, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var tags []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(resp.Data, &tags)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
}

func TestPostMetricList_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "pm-handler@test.com")
	vault := ts.createTestVault(t, token, "PM Vault")
	journalID := ts.createTestJournal(t, token, vault.ID, "Test Journal")
	postID := ts.createTestPost(t, token, vault.ID, journalID, "Test Post")

	metricPath := fmt.Sprintf("/api/vaults/%s/journals/%d/metrics", vault.ID, journalID)
	rec := ts.doRequest(http.MethodPost, metricPath, `{"label":"Mood"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create journal metric failed: %d %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var jm struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(resp.Data, &jm)

	pmPath := fmt.Sprintf("/api/vaults/%s/journals/%d/posts/%d/metrics", vault.ID, journalID, postID)
	ts.doRequest(http.MethodPost, pmPath, fmt.Sprintf(`{"journal_metric_id":%d,"value":7}`, jm.ID), token)

	rec = ts.doRequest(http.MethodGet, pmPath, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var metrics []struct {
		ID    uint `json:"id"`
		Value int  `json:"value"`
	}
	json.Unmarshal(resp.Data, &metrics)
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Value != 7 {
		t.Errorf("expected value 7, got %d", metrics[0].Value)
	}
}

func TestContactTabs_Success(t *testing.T) {
	ts := setupTestServer(t)
	_, auth := ts.registerTestUser(t, "tabs@example.com")
	vault := ts.createTestVault(t, auth.Token, "Tabs Vault")
	contact := ts.createTestContact(t, auth.Token, vault.ID, "Jane")

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/contacts/%s/tabs", vault.ID, contact.ID),
		"", auth.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatalf("expected success=true")
	}

	var tabs struct {
		TemplateID   uint   `json:"template_id"`
		TemplateName string `json:"template_name"`
		Pages        []struct {
			ID      uint   `json:"id"`
			Slug    string `json:"slug"`
			Modules []struct {
				ID   uint   `json:"id"`
				Type string `json:"type"`
			} `json:"modules"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(resp.Data, &tabs); err != nil {
		t.Fatalf("failed to parse tabs: %v", err)
	}
	if tabs.TemplateName != "Default template" {
		t.Errorf("expected 'Default template', got '%s'", tabs.TemplateName)
	}
	if len(tabs.Pages) != 5 {
		t.Fatalf("expected 5 pages, got %d", len(tabs.Pages))
	}
	if tabs.Pages[0].Slug != "contact" {
		t.Errorf("expected first page slug 'contact', got '%s'", tabs.Pages[0].Slug)
	}
	if len(tabs.Pages[0].Modules) != 10 {
		t.Errorf("expected 10 modules on contact page, got %d", len(tabs.Pages[0].Modules))
	}
}

func TestContactTabs_ContactNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, auth := ts.registerTestUser(t, "tabs404@example.com")
	vault := ts.createTestVault(t, auth.Token, "Tabs404 Vault")

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/contacts/%s/tabs", vault.ID, "nonexistent-id"),
		"", auth.Token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

type davSubscriptionData struct {
	ID                 string  `json:"id"`
	VaultID            string  `json:"vault_id"`
	URI                string  `json:"uri"`
	Username           string  `json:"username"`
	Active             bool    `json:"active"`
	SyncWay            uint8   `json:"sync_way"`
	Frequency          int     `json:"frequency"`
	LastSynchronizedAt *string `json:"last_synchronized_at"`
}

func (ts *testServer) createTestDavSubscription(t *testing.T, token, vaultID string) davSubscriptionData {
	t.Helper()
	body := `{"uri":"https://dav.example.com/contacts/","username":"testuser","password":"testpass","sync_way":2,"frequency":180}`
	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vaultID),
		body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create dav subscription failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data davSubscriptionData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse dav subscription data: %v", err)
	}
	return data
}

func TestDavSubscription_Create_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-create@test.com")
	vault := ts.createTestVault(t, token, "DAV Create Vault")

	body := `{"uri":"https://dav.example.com/contacts/","username":"testuser","password":"testpass","sync_way":2,"frequency":180}`
	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var sub davSubscriptionData
	if err := json.Unmarshal(resp.Data, &sub); err != nil {
		t.Fatalf("failed to parse subscription: %v", err)
	}
	if sub.ID == "" {
		t.Error("expected non-empty subscription ID")
	}
	if sub.VaultID != vault.ID {
		t.Errorf("expected vault_id=%s, got %s", vault.ID, sub.VaultID)
	}
	if sub.URI != "https://dav.example.com/contacts/" {
		t.Errorf("expected uri=https://dav.example.com/contacts/, got %s", sub.URI)
	}
	if sub.Username != "testuser" {
		t.Errorf("expected username=testuser, got %s", sub.Username)
	}
	if sub.SyncWay != 2 {
		t.Errorf("expected sync_way=2, got %d", sub.SyncWay)
	}
	if sub.Frequency != 180 {
		t.Errorf("expected frequency=180, got %d", sub.Frequency)
	}
	if !sub.Active {
		t.Error("expected active=true")
	}
}

func TestDavSubscription_Create_MissingFields(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-create-bad@test.com")
	vault := ts.createTestVault(t, token, "DAV Create Bad Vault")

	body := `{"uri":"https://dav.example.com/contacts/"}`
	rec := ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		body, token)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestDavSubscription_List_Empty(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-list-empty@test.com")
	vault := ts.createTestVault(t, token, "DAV List Empty Vault")

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		"", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var subs []davSubscriptionData
	if err := json.Unmarshal(resp.Data, &subs); err != nil {
		t.Fatalf("failed to parse subscriptions: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(subs))
	}
}

func TestDavSubscription_List_WithData(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-list-data@test.com")
	vault := ts.createTestVault(t, token, "DAV List Data Vault")

	ts.createTestDavSubscription(t, token, vault.ID)

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		"", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	var subs []davSubscriptionData
	if err := json.Unmarshal(resp.Data, &subs); err != nil {
		t.Fatalf("failed to parse subscriptions: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}
	if subs[0].URI != "https://dav.example.com/contacts/" {
		t.Errorf("expected uri=https://dav.example.com/contacts/, got %s", subs[0].URI)
	}
}

func TestDavSubscription_Get_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-get@test.com")
	vault := ts.createTestVault(t, token, "DAV Get Vault")

	created := ts.createTestDavSubscription(t, token, vault.ID)

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, created.ID),
		"", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var sub davSubscriptionData
	if err := json.Unmarshal(resp.Data, &sub); err != nil {
		t.Fatalf("failed to parse subscription: %v", err)
	}
	if sub.ID != created.ID {
		t.Errorf("expected id=%s, got %s", created.ID, sub.ID)
	}
	if sub.Username != "testuser" {
		t.Errorf("expected username=testuser, got %s", sub.Username)
	}
}

func TestDavSubscription_Get_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-get404@test.com")
	vault := ts.createTestVault(t, token, "DAV Get404 Vault")

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, "nonexistent-id"),
		"", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDavSubscription_Update_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-update@test.com")
	vault := ts.createTestVault(t, token, "DAV Update Vault")

	created := ts.createTestDavSubscription(t, token, vault.ID)

	body := `{"uri":"https://dav2.example.com/contacts/","username":"newuser","sync_way":3,"frequency":60}`
	rec := ts.doRequest(http.MethodPut,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, created.ID),
		body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var sub davSubscriptionData
	if err := json.Unmarshal(resp.Data, &sub); err != nil {
		t.Fatalf("failed to parse subscription: %v", err)
	}
	if sub.URI != "https://dav2.example.com/contacts/" {
		t.Errorf("expected uri=https://dav2.example.com/contacts/, got %s", sub.URI)
	}
	if sub.Username != "newuser" {
		t.Errorf("expected username=newuser, got %s", sub.Username)
	}
	if sub.SyncWay != 3 {
		t.Errorf("expected sync_way=3, got %d", sub.SyncWay)
	}
	if sub.Frequency != 60 {
		t.Errorf("expected frequency=60, got %d", sub.Frequency)
	}
}

func TestDavSubscription_Delete_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-delete@test.com")
	vault := ts.createTestVault(t, token, "DAV Delete Vault")

	created := ts.createTestDavSubscription(t, token, vault.ID)

	rec := ts.doRequest(http.MethodDelete,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, created.ID),
		"", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, created.ID),
		"", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDavSubscription_Delete_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-delete404@test.com")
	vault := ts.createTestVault(t, token, "DAV Delete404 Vault")

	rec := ts.doRequest(http.MethodDelete,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s", vault.ID, "nonexistent-id"),
		"", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDavSubscription_GetLogs_Empty(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-logs@test.com")
	vault := ts.createTestVault(t, token, "DAV Logs Vault")

	created := ts.createTestDavSubscription(t, token, vault.ID)

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions/%s/logs", vault.ID, created.ID),
		"", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var logs []json.RawMessage
	if err := json.Unmarshal(resp.Data, &logs); err != nil {
		t.Fatalf("failed to parse logs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
	if resp.Meta == nil {
		t.Fatal("expected meta in response")
	}
	if resp.Meta.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Meta.Total)
	}
}

func TestDavSubscription_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "dav-unauth@test.com")
	vault := ts.createTestVault(t, token, "DAV Unauth Vault")

	rec := ts.doRequest(http.MethodGet,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		"", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPost,
		fmt.Sprintf("/api/vaults/%s/dav/subscriptions", vault.ID),
		`{"uri":"https://dav.example.com/","username":"u","password":"p"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAvatarUploadAndGet(t *testing.T) {
	ts := setupTestServerWithStorage(t)
	token, _ := ts.registerTestUser(t, "avatar-test@example.com")
	vault := ts.createTestVault(t, token, "Avatar Vault")
	contact := ts.createTestContact(t, token, vault.ID, "AvatarTest")

	// 1) GET avatar before upload -> should return generated initials PNG
	rec := ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/avatar", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET avatar before upload: expected 200, got %d", rec.Code)
	}
	contentType := rec.Header().Get("Content-Type")
	t.Logf("Before upload Content-Type: %s, body size: %d", contentType, rec.Body.Len())

	// 2) Upload avatar via PUT (multipart)
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	fakeImage := make([]byte, 100)
	copy(fakeImage, pngHeader)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="file"; filename="avatar.png"`)
	mh.Set("Content-Type", "image/png")
	part, err := mw.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	part.Write(fakeImage)
	mw.Close()
	avatarPath := "/api/vaults/" + vault.ID + "/contacts/" + contact.ID + "/avatar"
	req := httptest.NewRequest(http.MethodPut, avatarPath, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	ts.e.ServeHTTP(rec, req)
	t.Logf("Upload response: status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT avatar: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3) GET avatar after upload -> should return the uploaded image
	rec = ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/avatar", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET avatar after upload: expected 200, got %d", rec.Code)
	}
	contentType = rec.Header().Get("Content-Type")
	t.Logf("After upload Content-Type: %s, body size: %d", contentType, rec.Body.Len())
	bodyBytes := rec.Body.Bytes()
	if len(bodyBytes) != 100 {
		t.Errorf("expected body size 100 (uploaded file), got %d — avatar not returned!", len(bodyBytes))
	}
}

func TestReportOverviewHandler(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "report-overview@example.com")
	vault := ts.createTestVault(t, token, "Report Vault")
	c1 := ts.createTestContact(t, token, vault.ID, "Alice")
	ts.createTestContact(t, token, vault.ID, "Bob")

	// Add address
	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+c1.ID+"/addresses",
		`{"line_1":"123 Main St","city":"Springfield","country":"US","address_type_id":1}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create address: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Add important date
	rec = ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+c1.ID+"/dates",
		`{"label":"Birthday","day":15,"month":6,"year":1990}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create date: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get overview
	rec = ts.doRequest(http.MethodGet,
		"/api/vaults/"+vault.ID+"/reports/overview", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("overview: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var overview struct {
		TotalContacts       int `json:"total_contacts"`
		TotalAddresses      int `json:"total_addresses"`
		TotalImportantDates int `json:"total_important_dates"`
		TotalMoodEntries    int `json:"total_mood_entries"`
	}
	if err := json.Unmarshal(resp.Data, &overview); err != nil {
		t.Fatalf("failed to parse overview: %v", err)
	}
	if overview.TotalContacts != 2 {
		t.Errorf("total_contacts: expected 2, got %d", overview.TotalContacts)
	}
	if overview.TotalAddresses != 1 {
		t.Errorf("total_addresses: expected 1, got %d", overview.TotalAddresses)
	}
	if overview.TotalImportantDates < 1 {
		t.Errorf("total_important_dates: expected >= 1, got %d", overview.TotalImportantDates)
	}
}

func TestInstanceInfo(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/instance/info", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var info struct {
		Version             string   `json:"version"`
		RegistrationEnabled bool     `json:"registration_enabled"`
		PasswordAuthEnabled bool     `json:"password_auth_enabled"`
		OAuthProviders      []string `json:"oauth_providers"`
		WebAuthnEnabled     bool     `json:"webauthn_enabled"`
		AppName             string   `json:"app_name"`
	}
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		t.Fatalf("failed to parse info: %v", err)
	}
	if info.Version != "test" {
		t.Errorf("expected version 'test', got '%s'", info.Version)
	}
	if !info.RegistrationEnabled {
		t.Error("expected registration_enabled=true by default")
	}
	if !info.PasswordAuthEnabled {
		t.Error("expected password_auth_enabled=true by default")
	}
	if info.AppName != "Monica Test" {
		t.Errorf("expected app_name 'Monica Test', got '%s'", info.AppName)
	}
}

func TestAdminListUsers_Success(t *testing.T) {
	ts := setupTestServer(t)

	token, _ := ts.registerTestUser(t, "admin-h-list@example.com")
	ts.registerTestUser(t, "admin-h-list2@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/users", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var users []json.RawMessage
	if err := json.Unmarshal(resp.Data, &users); err != nil {
		t.Fatalf("failed to parse users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestAdminListUsers_NonAdminForbidden(t *testing.T) {
	ts := setupTestServer(t)

	ts.registerTestUser(t, "admin-h-first@example.com")
	token2, _ := ts.registerTestUser(t, "admin-h-nonadmin@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/users", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminListUsers_Unauthenticated(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodGet, "/api/admin/users", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminToggleUser_Handler(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "admin-h-toggle@example.com")
	_, target := ts.registerTestUser(t, "admin-h-toggle-target@example.com")

	rec := ts.doRequest(http.MethodPut,
		"/api/admin/users/"+target.User.ID+"/toggle",
		`{"disabled":true}`, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var user models.User
	ts.db.First(&user, "id = ?", target.User.ID)
	if !user.Disabled {
		t.Error("expected user to be disabled")
	}
}

func TestAdminToggleUser_CannotDisableSelf(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, adminData := ts.registerTestUser(t, "admin-h-self@example.com")

	rec := ts.doRequest(http.MethodPut,
		"/api/admin/users/"+adminData.User.ID+"/toggle",
		`{"disabled":true}`, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminSetAdmin_Handler(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "admin-h-setadmin@example.com")
	_, target := ts.registerTestUser(t, "admin-h-setadmin-target@example.com")

	rec := ts.doRequest(http.MethodPut,
		"/api/admin/users/"+target.User.ID+"/admin",
		`{"is_instance_administrator":true}`, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var user models.User
	ts.db.First(&user, "id = ?", target.User.ID)
	if !user.IsInstanceAdministrator {
		t.Error("expected user to be instance admin")
	}
}

func TestAdminDeleteUser_Handler(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "admin-h-delete@example.com")
	_, target := ts.registerTestUser(t, "admin-h-del-target@example.com")

	rec := ts.doRequest(http.MethodDelete,
		"/api/admin/users/"+target.User.ID, "", adminToken)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	var userCount int64
	ts.db.Model(&models.User{}).Where("id = ?", target.User.ID).Count(&userCount)
	if userCount != 0 {
		t.Error("expected user to be deleted")
	}
}

func TestAdminDeleteUser_CannotDeleteSelf(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, adminData := ts.registerTestUser(t, "admin-h-delself@example.com")

	rec := ts.doRequest(http.MethodDelete,
		"/api/admin/users/"+adminData.User.ID, "", adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminSettings_GetAndUpdate(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "admin-h-settings@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/settings", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var getResp struct {
		Settings []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(resp.Data, &getResp); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}
	initialCount := len(getResp.Settings)
	if initialCount != 21 {
		t.Errorf("expected 21 settings initially (seeded from env), got %d", initialCount)
	}

	rec = ts.doRequest(http.MethodPut, "/api/admin/settings",
		`{"settings":[{"key":"app.name","value":"TestApp"},{"key":"test.custom.setting","value":"test"}]}`,
		adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update settings: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	if err := json.Unmarshal(resp.Data, &getResp); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}
	if len(getResp.Settings) != 22 {
		t.Errorf("expected 22 settings after update, got %d", len(getResp.Settings))
	}
}

func TestAdminSettings_NonAdminForbidden(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "admin-h-settings-first@example.com")
	token2, _ := ts.registerTestUser(t, "admin-h-settings-nonadmin@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/settings", "", token2)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginDisabledUser_Handler(t *testing.T) {
	ts := setupTestServer(t)
	_, data := ts.registerTestUser(t, "disabled-h@example.com")

	ts.db.Model(&models.User{}).Where("id = ?", data.User.ID).Update("disabled", true)

	rec := ts.doRequest(http.MethodPost, "/api/auth/login",
		`{"email":"disabled-h@example.com","password":"password123"}`, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestRegisterFirstUserIsInstanceAdmin_Handler(t *testing.T) {
	ts := setupTestServer(t)

	rec := ts.doRequest(http.MethodPost, "/api/auth/register",
		`{"first_name":"First","last_name":"User","email":"first-h@example.com","password":"password123"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var firstUser models.User
	ts.db.Where("email = ?", "first-h@example.com").First(&firstUser)
	if !firstUser.IsInstanceAdministrator {
		t.Error("expected first user to be instance administrator")
	}

	rec = ts.doRequest(http.MethodPost, "/api/auth/register",
		`{"first_name":"Second","last_name":"User","email":"second-h@example.com","password":"password123"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var secondUser models.User
	ts.db.Where("email = ?", "second-h@example.com").First(&secondUser)
	if secondUser.IsInstanceAdministrator {
		t.Error("expected second user NOT to be instance administrator")
	}
}

func TestAdminOAuthProviders_CRUD(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "admin-oauth-crud@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/oauth-providers", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var list []json.RawMessage
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("parse list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 providers initially, got %d", len(list))
	}

	rec = ts.doRequest(http.MethodPost, "/api/admin/oauth-providers",
		`{"type":"github","name":"github","client_id":"test-key","client_secret":"test-secret","display_name":"GitHub"}`,
		adminToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var created struct {
		ID          uint   `json:"id"`
		Type        string `json:"type"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Enabled     bool   `json:"enabled"`
		HasSecret   bool   `json:"has_secret"`
	}
	if err := json.Unmarshal(resp.Data, &created); err != nil {
		t.Fatalf("parse created: %v", err)
	}
	if created.Type != "github" {
		t.Errorf("expected type 'github', got '%s'", created.Type)
	}
	if !created.Enabled {
		t.Error("expected enabled=true by default")
	}
	if !created.HasSecret {
		t.Error("expected has_secret=true")
	}

	rec = ts.doRequest(http.MethodGet, "/api/admin/oauth-providers", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list after create: expected 200, got %d", rec.Code)
	}
	resp = parseResponse(t, rec)
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("parse list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 provider, got %d", len(list))
	}

	updateURL := fmt.Sprintf("/api/admin/oauth-providers/%d", created.ID)
	rec = ts.doRequest(http.MethodPut, updateURL,
		`{"display_name":"GitHub Updated","enabled":false}`, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseResponse(t, rec)
	var updated struct {
		DisplayName string `json:"display_name"`
		Enabled     bool   `json:"enabled"`
	}
	if err := json.Unmarshal(resp.Data, &updated); err != nil {
		t.Fatalf("parse updated: %v", err)
	}
	if updated.DisplayName != "GitHub Updated" {
		t.Errorf("expected display_name 'GitHub Updated', got '%s'", updated.DisplayName)
	}
	if updated.Enabled {
		t.Error("expected enabled=false after update")
	}

	rec = ts.doRequest(http.MethodDelete, updateURL, "", adminToken)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/admin/oauth-providers", "", adminToken)
	resp = parseResponse(t, rec)
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("parse list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 providers after delete, got %d", len(list))
	}
}

func TestAdminOAuthProviders_NonAdminForbidden(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "admin-oauth-first@example.com")
	nonAdminToken, _ := ts.registerTestUser(t, "admin-oauth-nonadmin@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/oauth-providers", "", nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodPost, "/api/admin/oauth-providers",
		`{"type":"github","name":"github","client_id":"k","client_secret":"s"}`, nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for create, got %d", rec.Code)
	}
}

func TestBackups_RequiresInstanceAdmin(t *testing.T) {
	ts := setupTestServer(t)
	ts.cfg.Backup.Dir = t.TempDir()
	adminToken, _ := ts.registerTestUser(t, "backup-admin@example.com")
	nonAdminToken, _ := ts.registerTestUser(t, "backup-nonadmin@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/admin/backups", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("instance admin: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/admin/backups", "", nonAdminToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/admin/backups/config", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("backup config: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDisabledUser_JWTRejected(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "disabled-jwt@example.com")

	rec := ts.doRequest(http.MethodGet, "/api/vaults", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("before disable: expected 200, got %d", rec.Code)
	}

	ts.db.Model(&models.User{}).Where("email = ?", "disabled-jwt@example.com").Update("disabled", true)

	rec = ts.doRequest(http.MethodGet, "/api/vaults", "", token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("after disable: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp.Success {
		t.Error("expected success=false for disabled user")
	}
}

func enableEmailVerification(ts *testServer) {
	settings := services.NewSystemSettingService(ts.db)
	settings.Set("auth.require_email_verification", "true")
	settings.Set("smtp.host", "test-smtp")
}

func TestVerifyEmail_Handler(t *testing.T) {
	ts := setupTestServer(t)
	enableEmailVerification(ts)

	_, _ = ts.registerTestUser(t, "first@example.com")

	body := `{"first_name":"Second","last_name":"User","email":"second@example.com","password":"password123"}`
	rec := ts.doRequest(http.MethodPost, "/api/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register second user: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var user models.User
	ts.db.First(&user, "email = ?", "second@example.com")
	if user.EmailVerificationToken == nil {
		t.Fatal("expected email_verification_token to be set")
	}

	verifyBody := fmt.Sprintf(`{"token":"%s"}`, *user.EmailVerificationToken)
	rec = ts.doRequest(http.MethodPost, "/api/auth/verify-email", verifyBody, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify email: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestVerifyEmailInvalidToken_Handler(t *testing.T) {
	ts := setupTestServer(t)

	body := `{"token":"invalid-token-12345"}`
	rec := ts.doRequest(http.MethodPost, "/api/auth/verify-email", body, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResendVerification_Handler(t *testing.T) {
	ts := setupTestServer(t)
	enableEmailVerification(ts)

	_, _ = ts.registerTestUser(t, "first@example.com")

	body := `{"first_name":"Second","last_name":"User","email":"second@example.com","password":"password123"}`
	rec := ts.doRequest(http.MethodPost, "/api/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register second user: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var auth authData
	json.Unmarshal(resp.Data, &auth)

	rec = ts.doRequest(http.MethodPost, "/api/auth/resend-verification", "", auth.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("resend verification: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnverifiedUserBlockedFromProtectedEndpoints(t *testing.T) {
	ts := setupTestServer(t)
	enableEmailVerification(ts)

	_, _ = ts.registerTestUser(t, "first@example.com")

	body := `{"first_name":"Second","last_name":"User","email":"second@example.com","password":"password123"}`
	rec := ts.doRequest(http.MethodPost, "/api/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register second user: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var auth authData
	json.Unmarshal(resp.Data, &auth)
	secondToken := auth.Token

	rec = ts.doRequest(http.MethodGet, "/api/vaults", "", secondToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unverified user: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var user models.User
	ts.db.First(&user, "email = ?", "second@example.com")
	if user.EmailVerificationToken == nil {
		t.Fatal("expected token to be set")
	}
	verifyBody := fmt.Sprintf(`{"token":"%s"}`, *user.EmailVerificationToken)
	rec = ts.doRequest(http.MethodPost, "/api/auth/verify-email", verifyBody, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify email: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/vaults", "", secondToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("verified user: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
