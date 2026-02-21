package services

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func ptrStr(s string) *string { return &s }

func setupOAuthProviderTest(t *testing.T) (*OAuthProviderService, *SystemSettingService, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	settings := NewSystemSettingService(db)
	svc := NewOAuthProviderService(db)
	svc.SetSystemSettings(settings)
	return svc, settings, db
}

func TestOAuthProviderCreate(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	resp, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type:         "github",
		Name:         "my-github",
		ClientID:     "cid-123",
		ClientSecret: "csecret-456",
		DisplayName:  "My GitHub",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == 0 {
		t.Error("Expected non-zero ID")
	}
	if resp.Type != "github" {
		t.Errorf("Expected type 'github', got '%s'", resp.Type)
	}
	if resp.Name != "my-github" {
		t.Errorf("Expected name 'my-github', got '%s'", resp.Name)
	}
	if resp.ClientID != "cid-123" {
		t.Errorf("Expected client_id 'cid-123', got '%s'", resp.ClientID)
	}
	if !resp.HasSecret {
		t.Error("Expected has_secret to be true")
	}
	if !resp.Enabled {
		t.Error("Expected enabled to be true by default")
	}
	if resp.DisplayName != "My GitHub" {
		t.Errorf("Expected display_name 'My GitHub', got '%s'", resp.DisplayName)
	}
}

func TestOAuthProviderCreateDuplicateName(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	_, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "dup", ClientID: "a", ClientSecret: "b",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.Create(dto.CreateOAuthProviderRequest{
		Type: "google", Name: "dup", ClientID: "c", ClientSecret: "d",
	})
	if err != ErrOAuthProviderNameExists {
		t.Errorf("Expected ErrOAuthProviderNameExists, got %v", err)
	}
}

func TestOAuthProviderList(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	_, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "gh", ClientID: "a", ClientSecret: "b",
	})
	if err != nil {
		t.Fatalf("Create 1 failed: %v", err)
	}
	_, err = svc.Create(dto.CreateOAuthProviderRequest{
		Type: "google", Name: "goo", ClientID: "c", ClientSecret: "d",
	})
	if err != nil {
		t.Fatalf("Create 2 failed: %v", err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(list))
	}
}

func TestOAuthProviderUpdate(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	created, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "upd", ClientID: "old-id", ClientSecret: "old-secret",
		DisplayName: "Old",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, dto.UpdateOAuthProviderRequest{
		ClientID:    ptrStr("new-id"),
		DisplayName: ptrStr("New Display"),
		Enabled:     ptrBool(false),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.ClientID != "new-id" {
		t.Errorf("Expected client_id 'new-id', got '%s'", updated.ClientID)
	}
	if updated.DisplayName != "New Display" {
		t.Errorf("Expected display_name 'New Display', got '%s'", updated.DisplayName)
	}
	if updated.Enabled {
		t.Error("Expected enabled to be false after update")
	}
	if !updated.HasSecret {
		t.Error("Expected has_secret to remain true (secret not changed)")
	}
}

func TestOAuthProviderUpdateNotFound(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)

	_, err := svc.Update(99999, dto.UpdateOAuthProviderRequest{
		ClientID: ptrStr("x"),
	})
	if err != ErrOAuthProviderNotFoundByID {
		t.Errorf("Expected ErrOAuthProviderNotFoundByID, got %v", err)
	}
}

func TestOAuthProviderDelete(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	created, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "del", ClientID: "a", ClientSecret: "b",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 providers after delete, got %d", len(list))
	}
}

func TestOAuthProviderDeleteNotFound(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)

	err := svc.Delete(99999)
	if err != ErrOAuthProviderNotFoundByID {
		t.Errorf("Expected ErrOAuthProviderNotFoundByID, got %v", err)
	}
}

func TestOAuthProviderReloadProviders(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	_, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "reload-gh", ClientID: "cid", ClientSecret: "csec",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	providers := goth.GetProviders()
	if _, ok := providers["github"]; !ok {
		t.Error("Expected 'github' in goth providers after create (goth uses type as provider name)")
	}
}

func TestOAuthProviderReloadDisabled(t *testing.T) {
	svc, _, _ := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	_, err := svc.Create(dto.CreateOAuthProviderRequest{
		Type: "github", Name: "disabled-gh", ClientID: "cid", ClientSecret: "csec",
		Enabled: ptrBool(false),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	providers := goth.GetProviders()
	if _, ok := providers["disabled-gh"]; ok {
		t.Error("Expected 'disabled-gh' NOT in goth providers when disabled")
	}
}

func TestSplitScopes(t *testing.T) {
	defaults := []string{"email", "profile"}

	result := splitScopes("", defaults...)
	if len(result) != 2 || result[0] != "email" || result[1] != "profile" {
		t.Errorf("Empty string: expected defaults %v, got %v", defaults, result)
	}

	result = splitScopes("read:user, repo", defaults...)
	if len(result) != 2 || result[0] != "read:user" || result[1] != "repo" {
		t.Errorf("Comma-separated: expected [read:user repo], got %v", result)
	}

	result = splitScopes(" , , ", defaults...)
	if len(result) != 2 || result[0] != "email" || result[1] != "profile" {
		t.Errorf("Whitespace-only: expected defaults %v, got %v", defaults, result)
	}

	result = splitScopes("single")
	if len(result) != 1 || result[0] != "single" {
		t.Errorf("Single scope: expected [single], got %v", result)
	}
}

func TestCreateGothProvider(t *testing.T) {
	appURL := "http://localhost:8080"

	tests := []struct {
		name     string
		provider models.OAuthProvider
		wantName string
		wantErr  bool
	}{
		{
			name: "github",
			provider: models.OAuthProvider{
				Type: "github", Name: "gh1", ClientID: "cid", ClientSecret: "sec",
			},
			wantName: "github",
		},
		{
			name: "google",
			provider: models.OAuthProvider{
				Type: "google", Name: "goo1", ClientID: "cid", ClientSecret: "sec",
			},
			wantName: "google",
		},
		{
			name: "gitlab",
			provider: models.OAuthProvider{
				Type: "gitlab", Name: "gl1", ClientID: "cid", ClientSecret: "sec",
			},
			wantName: "gitlab",
		},
		{
			name: "discord",
			provider: models.OAuthProvider{
				Type: "discord", Name: "disc1", ClientID: "cid", ClientSecret: "sec",
			},
			wantName: "discord",
		},
		{
			name: "unknown type",
			provider: models.OAuthProvider{
				Type: "twitter", Name: "tw1", ClientID: "cid", ClientSecret: "sec",
			},
			wantErr: true,
		},
		{
			name: "oidc missing discovery url",
			provider: models.OAuthProvider{
				Type: "oidc", Name: "oidc1", ClientID: "cid", ClientSecret: "sec",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gp, err := createGothProvider(tc.provider, appURL)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if gp.Name() != tc.wantName {
				t.Errorf("Expected provider name '%s', got '%s'", tc.wantName, gp.Name())
			}
		})
	}
}

func TestMigrateOAuthSettingsToProviders(t *testing.T) {
	_, settings, db := setupOAuthProviderTest(t)
	defer goth.ClearProviders()

	if err := settings.Set("oauth_github_key", "gh-key-123"); err != nil {
		t.Fatalf("Set github key: %v", err)
	}
	if err := settings.Set("oauth_github_secret", "gh-secret-456"); err != nil {
		t.Fatalf("Set github secret: %v", err)
	}
	if err := settings.Set("oauth_google_key", "goo-key-789"); err != nil {
		t.Fatalf("Set google key: %v", err)
	}
	if err := settings.Set("oauth_google_secret", "goo-secret-012"); err != nil {
		t.Fatalf("Set google secret: %v", err)
	}
	if err := settings.Set("oidc_client_id", "oidc-id"); err != nil {
		t.Fatalf("Set oidc id: %v", err)
	}
	if err := settings.Set("oidc_client_secret", "oidc-secret"); err != nil {
		t.Fatalf("Set oidc secret: %v", err)
	}
	if err := settings.Set("oidc_discovery_url", "https://example.com/.well-known"); err != nil {
		t.Fatalf("Set oidc discovery: %v", err)
	}
	if err := settings.Set("oidc_name", "My SSO"); err != nil {
		t.Fatalf("Set oidc name: %v", err)
	}

	MigrateOAuthSettingsToProviders(db, settings)

	var providers []models.OAuthProvider
	if err := db.Order("id ASC").Find(&providers).Error; err != nil {
		t.Fatalf("Find providers: %v", err)
	}
	if len(providers) != 3 {
		t.Fatalf("Expected 3 migrated providers, got %d", len(providers))
	}

	gh := providers[0]
	if gh.Type != "github" || gh.ClientID != "gh-key-123" || gh.ClientSecret != "gh-secret-456" {
		t.Errorf("GitHub provider mismatch: %+v", gh)
	}
	goo := providers[1]
	if goo.Type != "google" || goo.ClientID != "goo-key-789" || goo.ClientSecret != "goo-secret-012" {
		t.Errorf("Google provider mismatch: %+v", goo)
	}
	oidc := providers[2]
	if oidc.Type != "oidc" || oidc.ClientID != "oidc-id" || oidc.DisplayName != "My SSO" {
		t.Errorf("OIDC provider mismatch: %+v", oidc)
	}
	if oidc.DiscoveryURL != "https://example.com/.well-known" {
		t.Errorf("OIDC discovery URL mismatch: %s", oidc.DiscoveryURL)
	}

	legacyKeys := []string{
		"oauth_github_key", "oauth_github_secret",
		"oauth_google_key", "oauth_google_secret",
		"oidc_client_id", "oidc_client_secret", "oidc_discovery_url", "oidc_name",
	}
	for _, key := range legacyKeys {
		_, err := settings.Get(key)
		if err != ErrSystemSettingNotFound {
			t.Errorf("Expected legacy setting '%s' to be deleted, got err=%v", key, err)
		}
	}
}

func TestMigrateOAuthSettingsSkipsIfProvidersExist(t *testing.T) {
	_, settings, db := setupOAuthProviderTest(t)

	db.Create(&models.OAuthProvider{
		Type: "github", Name: "existing", ClientID: "x", ClientSecret: "y", Enabled: true,
	})

	if err := settings.Set("oauth_github_key", "should-not-migrate"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := settings.Set("oauth_github_secret", "should-not-migrate"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	MigrateOAuthSettingsToProviders(db, settings)

	var count int64
	db.Model(&models.OAuthProvider{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 provider (migration skipped), got %d", count)
	}

	val, err := settings.Get("oauth_github_key")
	if err != nil {
		t.Errorf("Expected legacy setting to still exist, got err=%v", err)
	}
	if val != "should-not-migrate" {
		t.Errorf("Expected legacy setting value 'should-not-migrate', got '%s'", val)
	}
}
