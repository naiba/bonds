package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/naiba/bonds/internal/models"
)

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type mcpBearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t mcpBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Authorization", "Bearer "+t.token)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}

func parseMCPResponse(t *testing.T, recBody string) mcpResponse {
	t.Helper()
	var resp mcpResponse
	if err := json.Unmarshal([]byte(recBody), &resp); err != nil {
		t.Fatalf("failed to parse MCP response: %v\nbody: %s", err, recBody)
	}
	return resp
}

func mcpToolCall(id int, name string, args string) string {
	if args == "" {
		args = `{}`
	}
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, id, name, args)
}

func decodeSDKStructuredContent(t *testing.T, content interface{}, target interface{}) {
	t.Helper()
	payload, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal SDK structured content: %v", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("failed to parse SDK structured content: %v\ncontent: %s", err, payload)
	}
}

func TestMCPRequiresAuth(t *testing.T) {
	ts := setupTestServer(t)
	rec := ts.doRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMCPInitializeAndToolsList(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-tools@example.com")

	rec := ts.doRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 initialize, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	if resp.Error != nil {
		t.Fatalf("initialize returned error: %+v", resp.Error)
	}

	rec = ts.doRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 tools/list, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseMCPResponse(t, rec.Body.String())
	var payload struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		t.Fatalf("failed to parse tools/list result: %v", err)
	}
	want := map[string]bool{"execute_action": false, "search_bonds": false, "fetch_resource": false, "discover_capabilities": false}
	for _, tool := range payload.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("expected tool %s in tools/list; tools=%+v", name, payload.Tools)
		}
	}
}

func TestMCPGoSDKClientCanWriteAndRead(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-sdk@example.com")
	vault := ts.createTestVault(t, token, "MCP SDK Vault")

	httpServer := httptest.NewServer(ts.e)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "bonds-mcp-sdk-test", Version: "test"}, nil)
	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: mcpBearerTransport{token: token}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("SDK client failed to connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	if got := session.InitializeResult().ProtocolVersion; got != "2025-06-18" {
		t.Fatalf("expected negotiated protocol version 2025-06-18, got %q", got)
	}

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("SDK tools/list failed: %v", err)
	}
	wantTools := map[string]bool{"execute_action": false, "search_bonds": false, "fetch_resource": false}
	for _, tool := range tools.Tools {
		if _, ok := wantTools[tool.Name]; ok {
			wantTools[tool.Name] = true
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Fatalf("expected SDK tools/list to include %s", name)
		}
	}

	createResult, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "execute_action",
		Arguments: map[string]interface{}{
			"action_id":   "post_vaults_by_vault_id_contacts",
			"path_params": map[string]string{"vault_id": vault.ID},
			"body": map[string]interface{}{
				"first_name": "SDK",
				"last_name":  "Client",
			},
		},
	})
	if err != nil {
		t.Fatalf("SDK execute_action call failed: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("SDK execute_action returned tool error: %+v", createResult.Content)
	}
	var createContent struct {
		Status int `json:"status"`
		Data   struct {
			Success bool        `json:"success"`
			Data    contactData `json:"data"`
		} `json:"data"`
	}
	decodeSDKStructuredContent(t, createResult.StructuredContent, &createContent)
	if createContent.Status != http.StatusCreated {
		t.Fatalf("expected execute_action status 201, got %d", createContent.Status)
	}
	createdContact := createContent.Data.Data
	if !createContent.Data.Success || createdContact.ID == "" {
		t.Fatalf("expected created contact data, got %+v", createContent)
	}
	if createdContact.FirstName != "SDK" || createdContact.LastName != "Client" {
		t.Fatalf("unexpected created contact name: %+v", createdContact)
	}

	searchResult, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "search_bonds",
		Arguments: map[string]interface{}{
			"vault_id": vault.ID,
			"query":    "SDK",
		},
	})
	if err != nil {
		t.Fatalf("SDK search_bonds call failed: %v", err)
	}
	if searchResult.IsError {
		t.Fatalf("SDK search_bonds returned tool error: %+v", searchResult.Content)
	}
	var searchContent struct {
		Results []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"results"`
	}
	decodeSDKStructuredContent(t, searchResult.StructuredContent, &searchContent)
	foundContact := false
	for _, result := range searchContent.Results {
		if result.Type == "contact" && result.ID == createdContact.ID {
			foundContact = true
		}
	}
	if !foundContact {
		t.Fatalf("expected search_bonds to find created contact %s; results=%+v", createdContact.ID, searchContent.Results)
	}

	resource, err := session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "bonds://contact/" + createdContact.ID})
	if err != nil {
		t.Fatalf("SDK resources/read failed: %v", err)
	}
	if len(resource.Contents) != 1 {
		t.Fatalf("expected one resource content, got %d", len(resource.Contents))
	}
	var fetched struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.Unmarshal([]byte(resource.Contents[0].Text), &fetched); err != nil {
		t.Fatalf("failed to parse fetched resource: %v\ntext: %s", err, resource.Contents[0].Text)
	}
	if fetched.ID != createdContact.ID || fetched.FirstName != "SDK" || fetched.LastName != "Client" {
		t.Fatalf("unexpected fetched resource: %+v", fetched)
	}
}

func TestMCPDiscoverCapabilitiesIncludesAPIRoutes(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-actions@example.com")

	body := mcpToolCall(1, "discover_capabilities", `{"filter":"contacts","limit":100}`)
	rec := ts.doRequest(http.MethodPost, "/mcp", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	var toolResult struct {
		StructuredContent struct {
			Actions []struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			} `json:"actions"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse tool result: %v", err)
	}
	found := false
	for _, action := range toolResult.StructuredContent.Actions {
		if action.ID == "post_vaults_by_vault_id_contacts" && action.Path == "/api/vaults/:vault_id/contacts" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected contacts create action, actions=%+v", toolResult.StructuredContent.Actions)
	}
}

func TestMCPExecuteActionUsesExistingPermissions(t *testing.T) {
	ts := setupTestServer(t)
	managerToken, managerAuth := ts.registerTestUser(t, "mcp-manager@example.com")
	vault := ts.createTestVault(t, managerToken, "MCP Vault")
	var birthdateType models.ContactImportantDateType
	if err := ts.db.Where("vault_id = ? AND internal_type = ?", vault.ID, "birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("find birthdate type: %v", err)
	}

	body := mcpToolCall(1, "execute_action", fmt.Sprintf(`{"action_id":"post_vaults_by_vault_id_contacts","path_params":{"vault_id":%q},"body":{"first_name":"Alice","last_name":"Agent","important_dates":[{"label":"Birthdate","date_precision":"full","year":1990,"month":6,"day":15,"contact_important_date_type_id":%d}]}}`, vault.ID, birthdateType.ID))
	rec := ts.doRequest(http.MethodPost, "/mcp", body, managerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	var toolResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if toolResult.IsError {
		t.Fatalf("manager execute_action unexpectedly failed: %s", rec.Body.String())
	}
	var createdContact models.Contact
	if err := ts.db.Where("vault_id = ? AND first_name = ?", vault.ID, "Alice").First(&createdContact).Error; err != nil {
		t.Fatalf("load contact created through MCP: %v", err)
	}
	var birthdateCount int64
	if err := ts.db.Model(&models.ContactImportantDate{}).
		Where("contact_id = ? AND contact_important_date_type_id = ?", createdContact.ID, birthdateType.ID).
		Count(&birthdateCount).Error; err != nil {
		t.Fatalf("count MCP-created birthdates: %v", err)
	}
	if birthdateCount != 1 {
		t.Fatalf("expected MCP to create one birthdate, got %d", birthdateCount)
	}

	viewer := createSecondUser(t, ts, managerAuth.User.AccountID, "mcp-viewer@example.com", false)
	addUserToVault(t, ts, viewer.ID, vault.ID, models.PermissionViewer)
	viewerToken := generateJWT(viewer.ID, viewer.AccountID, viewer.Email, false, false)
	rec = ts.doRequest(http.MethodPost, "/mcp", body, viewerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected MCP 200 with tool error, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseMCPResponse(t, rec.Body.String())
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse viewer result: %v", err)
	}
	if !toolResult.IsError {
		t.Fatalf("viewer execute_action should fail through existing permission middleware: %s", rec.Body.String())
	}
}

func TestMCPSearchAndFetchResource(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-search@example.com")
	vault := ts.createTestVault(t, token, "Search Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Searchable")

	searchBody := mcpToolCall(1, "search_bonds", fmt.Sprintf(`{"vault_id":%q,"query":"Searchable"}`, vault.ID))
	rec := ts.doRequest(http.MethodPost, "/mcp", searchBody, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 search, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	var searchResult struct {
		IsError           bool `json:"isError"`
		StructuredContent struct {
			Capabilities struct {
				SemanticVectorSearch bool `json:"semantic_vector_search"`
			} `json:"capabilities"`
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(resp.Result, &searchResult); err != nil {
		t.Fatalf("failed to parse search result: %v", err)
	}
	if searchResult.IsError {
		t.Fatalf("search_bonds failed: %s", rec.Body.String())
	}
	if searchResult.StructuredContent.Capabilities.SemanticVectorSearch {
		t.Fatal("MCP search must not enable vector search")
	}
	found := false
	for _, result := range searchResult.StructuredContent.Results {
		if result.ID == contact.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected contact in search results: %s", rec.Body.String())
	}

	fetchBody := mcpToolCall(2, "fetch_resource", fmt.Sprintf(`{"uri":"bonds://contact/%s"}`, contact.ID))
	rec = ts.doRequest(http.MethodPost, "/mcp", fetchBody, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 fetch, got %d: %s", rec.Code, rec.Body.String())
	}
	resp = parseMCPResponse(t, rec.Body.String())
	var fetchResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resp.Result, &fetchResult); err != nil {
		t.Fatalf("failed to parse fetch result: %v", err)
	}
	if fetchResult.IsError {
		t.Fatalf("fetch_resource failed: %s", rec.Body.String())
	}
}

func TestMCPAcceptsPersonalAccessToken(t *testing.T) {
	ts := setupTestServer(t)
	jwtToken, _ := ts.registerTestUser(t, "mcp-pat@example.com")

	rec := ts.doRequest(http.MethodPost, "/api/settings/tokens", `{"name":"MCP Agent"}`, jwtToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create PAT: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	apiResp := parseResponse(t, rec)
	var created struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(apiResp.Data, &created); err != nil {
		t.Fatalf("failed to parse PAT response: %v", err)
	}

	body := mcpToolCall(1, "get_current_context", `{}`)
	rec = ts.doRequest(http.MethodPost, "/mcp", body, created.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with PAT, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	if resp.Error != nil {
		t.Fatalf("MCP returned error with PAT: %+v", resp.Error)
	}
	var toolResult struct {
		IsError           bool `json:"isError"`
		StructuredContent struct {
			User struct {
				AuthType string `json:"auth_type"`
			} `json:"user"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse PAT MCP result: %v", err)
	}
	if toolResult.IsError {
		t.Fatalf("get_current_context failed with PAT: %s", rec.Body.String())
	}
	if toolResult.StructuredContent.User.AuthType != "pat" {
		t.Fatalf("expected PAT auth_type, got %q: %s", toolResult.StructuredContent.User.AuthType, rec.Body.String())
	}
}

func TestMCPFetchResourceRejectsUnlistedContact(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-unlisted@example.com")
	vault := ts.createTestVault(t, token, "Unlisted Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Unlisted")
	if err := ts.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Update("listed", false).Error; err != nil {
		t.Fatalf("archive contact: %v", err)
	}

	fetchBody := mcpToolCall(1, "fetch_resource", fmt.Sprintf(`{"uri":"bonds://contact/%s"}`, contact.ID))
	rec := ts.doRequest(http.MethodPost, "/mcp", fetchBody, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected MCP 200 with tool error, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	var toolResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse unlisted fetch result: %v", err)
	}
	if !toolResult.IsError {
		t.Fatalf("unlisted contact fetch should fail: %s", rec.Body.String())
	}
}

func TestMCPFetchResourceRejectsTaskOnlyAssignedToUnlistedContact(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := ts.registerTestUser(t, "mcp-unlisted-task@example.com")
	vault := ts.createTestVault(t, token, "Unlisted Task Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Unlisted")
	if err := ts.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Update("listed", false).Error; err != nil {
		t.Fatalf("archive contact: %v", err)
	}
	hiddenTask := models.ContactTask{VaultID: vault.ID, Label: "Unlisted-only task", AuthorName: "tester"}
	if err := ts.db.Create(&hiddenTask).Error; err != nil {
		t.Fatalf("failed to create hidden task: %v", err)
	}
	if err := ts.db.Create(&models.TaskContact{ContactTaskID: hiddenTask.ID, ContactID: contact.ID}).Error; err != nil {
		t.Fatalf("failed to assign hidden task: %v", err)
	}

	fetchBody := mcpToolCall(1, "fetch_resource", fmt.Sprintf(`{"uri":"bonds://task/%d"}`, hiddenTask.ID))
	rec := ts.doRequest(http.MethodPost, "/mcp", fetchBody, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected MCP 200 with tool error, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseMCPResponse(t, rec.Body.String())
	var toolResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to parse hidden task fetch result: %v", err)
	}
	if !toolResult.IsError {
		t.Fatalf("unlisted-only task fetch should fail: %s", rec.Body.String())
	}
}
