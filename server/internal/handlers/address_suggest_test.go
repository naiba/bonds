package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/models"
)

type suggestionsData struct {
	Enabled     bool `json:"enabled"`
	Suggestions []struct {
		Label string `json:"label"`
	} `json:"suggestions"`
	Attribution []struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	} `json:"attribution"`
}

func suggestProbe(t *testing.T, ts *testServer, vaultID, token string) suggestionsData {
	t.Helper()
	// An empty q probes availability without the service ever contacting the
	// provider, so these tests cannot leak a request to a real geocoder.
	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vaultID+"/addresses/suggest?q=", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("suggest probe: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data suggestionsData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to parse suggestions: %v", err)
	}
	return data
}

// The admin's precision choice must reach the running service through the
// production wiring in RegisterRoutes, not only through test setters: at
// locality precision no street may leave the server, so lookup reports itself
// unavailable however capable the provider is.
func TestAddressSuggestHonoursThePersistedPrecisionSetting(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "locationiq", APIKey: "test-key", Precision: "locality"}
	})
	token, _ := ts.registerTestUser(t, "suggest-precision@example.com")
	vault := ts.createTestVault(t, token, "Precision Vault")

	data := suggestProbe(t, ts, vault.ID, token)
	if data.Enabled {
		t.Fatal("locality precision must withdraw address lookup, but the route reports it enabled")
	}
	if len(data.Attribution) != 2 {
		t.Fatalf("forward-geocoded coordinates still require provider credits, got %v", data.Attribution)
	}
}

func TestAddressSuggestEnabledAtExactPrecisionWithCapableProvider(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "locationiq", APIKey: "test-key", Precision: "exact"}
	})
	token, _ := ts.registerTestUser(t, "suggest-exact@example.com")
	vault := ts.createTestVault(t, token, "Exact Vault")

	data := suggestProbe(t, ts, vault.ID, token)
	if !data.Enabled {
		t.Fatal("exact precision with LocationIQ should offer lookup")
	}
	if len(data.Attribution) != 2 {
		t.Fatalf("expected OpenStreetMap and LocationIQ credits, got %v", data.Attribution)
	}
	if data.Attribution[0].URL != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("first credit must be OpenStreetMap's copyright page, got %v", data.Attribution[0])
	}
}

func TestAdminGeocodingSettingsHotReloadTheRunningAddressService(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "locationiq", APIKey: "old-key", Precision: "exact"}
	})
	adminToken, _ := ts.registerTestUser(t, "suggest-hot-reload@example.com")
	vault := ts.createTestVault(t, adminToken, "Hot Reload Vault")

	if data := suggestProbe(t, ts, vault.ID, adminToken); !data.Enabled {
		t.Fatal("precondition: exact LocationIQ lookup should be enabled")
	}

	// Saving a provider credential through the structured endpoint must replace
	// the active runtime without exposing the key through generic settings.
	rec := ts.doRequest(http.MethodPut, "/api/admin/geocoding/providers/locationiq",
		`{"config":{"api_key":"new-key"}}`,
		adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update LocationIQ provider: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Saving privacy precision must affect the already-running service; a
	// restart-only setting would leave this probe enabled.
	rec = ts.doRequest(http.MethodPut, "/api/admin/geocoding",
		`{"active_provider":"locationiq","precision":"locality"}`,
		adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update geocoding precision: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := suggestProbe(t, ts, vault.ID, adminToken)
	if data.Enabled {
		t.Fatal("the running service kept exact precision after the admin update")
	}
	if len(data.Attribution) != 2 {
		t.Fatalf("the active LocationIQ provider should retain both credits, got %v", data.Attribution)
	}

	// Removing the provider must atomically remove its runtime instance too.
	rec = ts.doRequest(http.MethodPut, "/api/admin/geocoding",
		`{"active_provider":"","precision":"exact"}`,
		adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove geocoding provider: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data = suggestProbe(t, ts, vault.ID, adminToken)
	if data.Enabled || len(data.Attribution) != 0 {
		t.Fatalf("removed provider is still active: %+v", data)
	}
}

func TestAdminGeocodingProviderConfigurationsAreStructured(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "locationiq", APIKey: "bootstrap-key", Precision: "exact"}
	})
	adminToken, _ := ts.registerTestUser(t, "geocoding-structured@example.com")
	vault := ts.createTestVault(t, adminToken, "Structured Geocoding Vault")

	rec := ts.doRequest(http.MethodGet, "/api/admin/geocoding", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("get geocoding: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data struct {
		ActiveProvider string `json:"active_provider"`
		Providers      []struct {
			ID         string            `json:"id"`
			Configured bool              `json:"configured"`
			Config     map[string]string `json:"config"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("parse geocoding response: %v", err)
	}
	if data.ActiveProvider != "locationiq" || len(data.Providers) != 4 {
		t.Fatalf("unexpected provider catalog: %+v", data)
	}
	for _, provider := range data.Providers {
		if provider.ID == "locationiq" && (!provider.Configured || provider.Config["api_key"] != "***") {
			t.Fatalf("LocationIQ config should be configured and redacted: %+v", provider)
		}
		if provider.ID == "photon" && provider.Config["base_url"] != "https://photon.komoot.io" {
			t.Fatalf("Photon should expose the public default: %+v", provider)
		}
	}

	rec = ts.doRequest(http.MethodPut, "/api/admin/geocoding/providers/geoapify",
		`{"config":{"api_key":"geoapify-secret"}}`, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("save Geoapify: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = ts.doRequest(http.MethodPut, "/api/admin/geocoding",
		`{"active_provider":"geoapify","precision":"exact"}`, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate Geoapify: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if probe := suggestProbe(t, ts, vault.ID, adminToken); !probe.Enabled || len(probe.Attribution) != 2 || probe.Attribution[1].Label != "Powered by Geoapify" {
		t.Fatalf("Geoapify runtime was not activated: %+v", probe)
	}

	rec = ts.doRequest(http.MethodDelete, "/api/admin/geocoding/providers/geoapify", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete Geoapify: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if probe := suggestProbe(t, ts, vault.ID, adminToken); probe.Enabled || len(probe.Attribution) != 0 {
		t.Fatalf("deleted active provider remained available: %+v", probe)
	}
}

func TestAdminGeocodingRejectsInvalidProviderConfigurations(t *testing.T) {
	ts := setupTestServer(t)
	adminToken, _ := ts.registerTestUser(t, "geocoding-invalid@example.com")

	for _, tc := range []struct {
		path string
		body string
	}{
		{"/api/admin/geocoding/providers/photon", `{"config":{"base_url":"file:///tmp/photon"}}`},
		{"/api/admin/geocoding/providers/unknown", `{"config":{}}`},
		{"/api/admin/geocoding", `{"active_provider":"geoapify","precision":"exact"}`},
	} {
		rec := ts.doRequest(http.MethodPut, tc.path, tc.body, adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %s: expected 400, got %d: %s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestAdminGeocodingRequiresInstanceAdministrator(t *testing.T) {
	ts := setupTestServer(t)
	ts.registerTestUser(t, "geocoding-admin@example.com")
	nonAdminToken, _ := ts.registerTestUser(t, "geocoding-non-admin@example.com")

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/admin/geocoding", ""},
		{http.MethodPut, "/api/admin/geocoding/providers/photon", `{"config":{"base_url":"https://photon.example.com"}}`},
	} {
		rec := ts.doRequest(tc.method, tc.path, tc.body, nonAdminToken)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d: %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestAddressSuggestDisabledForPublicNominatim(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "nominatim", Precision: "exact"}
	})
	token, _ := ts.registerTestUser(t, "suggest-nominatim@example.com")
	vault := ts.createTestVault(t, token, "Nominatim Vault")

	data := suggestProbe(t, ts, vault.ID, token)
	if data.Enabled {
		t.Fatal("the public Nominatim instance forbids autocomplete; the route must report lookup unavailable")
	}
	if len(data.Attribution) != 1 || data.Attribution[0].URL != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("Nominatim forward-geocoded coordinates still require OSM attribution, got %v", data.Attribution)
	}
}

func TestMapReportIncludesAttributionForDisplayedCoordinates(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "nominatim", Precision: "exact"}
	})
	token, _ := ts.registerTestUser(t, "map-attribution@example.com")
	vault := ts.createTestVault(t, token, "Attributed Map Vault")
	contact := ts.createTestContact(t, token, vault.ID, "Mapped Contact")

	rec := ts.doRequest(http.MethodPost,
		"/api/vaults/"+vault.ID+"/contacts/"+contact.ID+"/addresses",
		`{"line_1":"Synthetic Street","city":"London","country":"United Kingdom","latitude":51.5,"longitude":-0.12}`,
		token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create mapped address: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/reports/map", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("map report: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	var data struct {
		GeocodedCount int `json:"geocoded_count"`
		Attribution   []struct {
			URL string `json:"url"`
		} `json:"attribution"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("parse map report: %v", err)
	}
	if data.GeocodedCount != 1 {
		t.Fatalf("expected one displayed coordinate, got %d", data.GeocodedCount)
	}
	if len(data.Attribution) != 1 || data.Attribution[0].URL != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("displayed Nominatim coordinates need linked OSM attribution, got %v", data.Attribution)
	}
}

// Every lookup spends a request against the instance's geocoding quota, so the
// route is held to the same permission as saving the address it would fill in.
func TestAddressSuggestRequiresEditorPermission(t *testing.T) {
	ts := setupTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Geocoding = config.GeocodingConfig{Provider: "locationiq", APIKey: "test-key", Precision: "exact"}
	})
	token, auth := ts.registerTestUser(t, "suggest-owner@example.com")
	vault := ts.createTestVault(t, token, "Permission Vault")

	viewer := createSecondUser(t, ts, auth.User.AccountID, "suggest-viewer@example.com", false)
	addUserToVault(t, ts, viewer.ID, vault.ID, models.PermissionViewer)
	viewerToken := generateJWT(viewer.ID, viewer.AccountID, viewer.Email, false, false)
	rec := ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/addresses/suggest?q=", "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer suggest: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	editor := createSecondUser(t, ts, auth.User.AccountID, "suggest-editor@example.com", false)
	addUserToVault(t, ts, editor.ID, vault.ID, models.PermissionEditor)
	editorToken := generateJWT(editor.ID, editor.AccountID, editor.Email, false, false)
	rec = ts.doRequest(http.MethodGet, "/api/vaults/"+vault.ID+"/addresses/suggest?q=", "", editorToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("editor suggest: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
