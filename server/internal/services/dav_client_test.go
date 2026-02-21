package services

import (
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupDavClientTest(t *testing.T) (*DavClientService, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "dav-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	svc := NewDavClientService(db, cfg.Secret)
	return svc, vault.ID, resp.User.ID
}

func TestDavClientService_EncryptDecryptPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewDavClientService(db, "test-secret-key")

	original := "my-app-password-123!@#"
	encrypted, err := svc.encryptPassword(original)
	if err != nil {
		t.Fatalf("encryptPassword failed: %v", err)
	}
	if encrypted == original {
		t.Error("encrypted should differ from original")
	}

	decrypted, err := svc.decryptPassword(encrypted)
	if err != nil {
		t.Fatalf("decryptPassword failed: %v", err)
	}
	if decrypted != original {
		t.Errorf("expected %q, got %q", original, decrypted)
	}

	svc2 := NewDavClientService(db, "different-secret")
	_, err = svc2.decryptPassword(encrypted)
	if err == nil {
		t.Error("expected decryption to fail with different key")
	}
}

func TestDavClientService_Create(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	resp, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:       "https://dav.example.com/addressbooks/user/contacts/",
		Username:  "user@example.com",
		Password:  "app-password",
		SyncWay:   2,
		Frequency: 180,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.VaultID != vaultID {
		t.Errorf("expected vault_id %s, got %s", vaultID, resp.VaultID)
	}
	if resp.URI != "https://dav.example.com/addressbooks/user/contacts/" {
		t.Errorf("unexpected URI: %s", resp.URI)
	}
	if resp.Username != "user@example.com" {
		t.Errorf("unexpected username: %s", resp.Username)
	}
	if resp.SyncWay != 2 {
		t.Errorf("expected sync_way 2, got %d", resp.SyncWay)
	}
	if resp.Frequency != 180 {
		t.Errorf("expected frequency 180, got %d", resp.Frequency)
	}

	var sub models.AddressBookSubscription
	svc.db.First(&sub, "id = ?", resp.ID)
	if sub.Password == "app-password" {
		t.Error("password should be encrypted in DB")
	}
}

func TestDavClientService_CreateDefaults(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	resp, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/addressbooks/",
		Username: "user@example.com",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.SyncWay != 2 {
		t.Errorf("expected default sync_way 2, got %d", resp.SyncWay)
	}
	if resp.Frequency != 180 {
		t.Errorf("expected default frequency 180, got %d", resp.Frequency)
	}
}

func TestDavClientService_List(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	for i := 0; i < 3; i++ {
		_, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
			URI:      "https://dav.example.com/ab/" + string(rune('a'+i)) + "/",
			Username: "user",
			Password: "pwd",
		})
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	list, err := svc.List(vaultID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 subscriptions, got %d", len(list))
	}
}

func TestDavClientService_Get(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	created, _ := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})

	got, err := svc.Get(created.ID, vaultID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, got.ID)
	}

	_, err = svc.Get("nonexistent", vaultID)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestDavClientService_Update(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	created, _ := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})

	active := false
	updated, err := svc.Update(created.ID, vaultID, dto.UpdateDavSubscriptionRequest{
		URI:      "https://dav.example.com/new-contacts/",
		Username: "newuser",
		Active:   &active,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.URI != "https://dav.example.com/new-contacts/" {
		t.Errorf("unexpected URI: %s", updated.URI)
	}
	if updated.Username != "newuser" {
		t.Errorf("unexpected username: %s", updated.Username)
	}
	if updated.Active != false {
		t.Error("expected active=false")
	}
}

func TestDavClientService_UpdatePasswordEmpty(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	created, _ := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "original-pwd",
	})

	var subBefore models.AddressBookSubscription
	svc.db.First(&subBefore, "id = ?", created.ID)

	_, err := svc.Update(created.ID, vaultID, dto.UpdateDavSubscriptionRequest{
		Username: "updated-user",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	var subAfter models.AddressBookSubscription
	svc.db.First(&subAfter, "id = ?", created.ID)
	if subAfter.Password != subBefore.Password {
		t.Error("password should not have changed when empty password provided")
	}

	_, password, err := svc.GetDecryptedPassword(created.ID, vaultID)
	if err != nil {
		t.Fatalf("GetDecryptedPassword failed: %v", err)
	}
	if password != "original-pwd" {
		t.Errorf("expected original password, got %q", password)
	}
}

func TestDavClientService_Delete(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	created, _ := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})

	err := svc.Delete(created.ID, vaultID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.Get(created.ID, vaultID)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound after delete, got %v", err)
	}

	err = svc.Delete("nonexistent", vaultID)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestDavClientService_ListDue(t *testing.T) {
	svc, vaultID, userID := setupDavClientTest(t)

	_, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:       "https://dav.example.com/never-synced/",
		Username:  "user",
		Password:  "pwd",
		Frequency: 60,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	resp2, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:       "https://dav.example.com/recently-synced/",
		Username:  "user",
		Password:  "pwd",
		Frequency: 60,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	now := time.Now()
	svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", resp2.ID).
		Update("last_synchronized_at", now)

	resp3, err := svc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:       "https://dav.example.com/old-synced/",
		Username:  "user",
		Password:  "pwd",
		Frequency: 1,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	oldTime := time.Now().Add(-2 * time.Hour)
	svc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", resp3.ID).
		Update("last_synchronized_at", oldTime)

	due, err := svc.ListDue()
	if err != nil {
		t.Fatalf("ListDue failed: %v", err)
	}

	dueIDs := make(map[string]bool)
	for _, s := range due {
		dueIDs[s.ID] = true
	}

	if !dueIDs[resp3.ID] {
		t.Error("expected old-synced subscription to be due")
	}
	if dueIDs[resp2.ID] {
		t.Error("expected recently-synced subscription to NOT be due")
	}
	if len(due) < 2 {
		t.Errorf("expected at least 2 due subscriptions (never-synced + old-synced), got %d", len(due))
	}
}
