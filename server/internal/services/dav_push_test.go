package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

func setupDavPushTest(t *testing.T) (*DavPushService, *DavClientService, *VCardService, *ContactService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "dav-push-test@example.com",
		Password:  "password123",
	}, "en")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	vault, err := vaultSvc.CreateVault(resp.User.AccountID, resp.User.ID, dto.CreateVaultRequest{Name: "Test Vault"}, "en")
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	clientSvc := NewDavClientService(db, cfg.Secret)
	vcardSvc := NewVCardService(db)
	contactSvc := NewContactService(db)
	pushSvc := NewDavPushService(db, clientSvc, vcardSvc)

	return pushSvc, clientSvc, vcardSvc, contactSvc, vault.ID, resp.User.ID, resp.User.AccountID
}

func createPushSubscription(t *testing.T, clientSvc *DavClientService, vaultID, userID string, syncWay uint8) *dto.DavSubscriptionResponse {
	t.Helper()
	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
		SyncWay:  syncWay,
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}
	return sub
}

func TestDavPushService_PushContactChange_NewContact(t *testing.T) {
	pushSvc, clientSvc, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	sub := createPushSubscription(t, clientSvc, vaultID, userID, SyncWayPush)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Alice",
		LastName:  "Smith",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	var putCalled bool
	var putPath string
	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			putCalled = true
			putPath = path
			return &carddav.AddressObject{
				Path: path,
				ETag: "new-etag-1",
			}, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)

	if !putCalled {
		t.Error("expected PutAddressObject to be called")
	}

	expectedPath := "https://dav.example.com/contacts/" + contact.ID + ".vcf"
	if putPath != expectedPath {
		t.Errorf("expected PUT path %q, got %q", expectedPath, putPath)
	}

	var state models.ContactSubscriptionState
	if err := pushSvc.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
		t.Fatalf("ContactSubscriptionState not created: %v", err)
	}
	if state.DistantURI != expectedPath {
		t.Errorf("expected state DistantURI %q, got %q", expectedPath, state.DistantURI)
	}
	if state.DistantEtag != "new-etag-1" {
		t.Errorf("expected state DistantEtag %q, got %q", "new-etag-1", state.DistantEtag)
	}

	var logs []models.DavSyncLog
	pushSvc.db.Where("address_book_subscription_id = ? AND action = ?", sub.ID, "pushed").Find(&logs)
	if len(logs) != 1 {
		t.Errorf("expected 1 push log, got %d", len(logs))
	}
}

func TestDavPushService_PushContactChange_ExistingContact(t *testing.T) {
	pushSvc, clientSvc, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	sub := createPushSubscription(t, clientSvc, vaultID, userID, SyncWayBoth)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Bob",
		LastName:  "Jones",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	existingURI := "https://dav.example.com/contacts/existing-bob.vcf"
	pushSvc.db.Create(&models.ContactSubscriptionState{
		ContactID:                 contact.ID,
		AddressBookSubscriptionID: sub.ID,
		DistantURI:                existingURI,
		DistantEtag:               "old-etag",
	})

	var putPath string
	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			putPath = path
			return &carddav.AddressObject{
				Path: path,
				ETag: "updated-etag",
			}, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)

	if putPath != existingURI {
		t.Errorf("expected PUT to existing URI %q, got %q", existingURI, putPath)
	}

	var state models.ContactSubscriptionState
	pushSvc.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state)
	if state.DistantEtag != "updated-etag" {
		t.Errorf("expected updated etag %q, got %q", "updated-etag", state.DistantEtag)
	}
}

func TestDavPushService_PushContactChange_NoSubscriptions(t *testing.T) {
	pushSvc, _, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Charlie",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			t.Error("PutAddressObject should not be called when no subscriptions exist")
			return nil, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)
}

func TestDavPushService_PushContactChange_PullOnlySubscription(t *testing.T) {
	pushSvc, clientSvc, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	createPushSubscription(t, clientSvc, vaultID, userID, SyncWayPull)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Diana",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			t.Error("PutAddressObject should not be called for pull-only subscription")
			return nil, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)
}

func TestDavPushService_PushContactDelete(t *testing.T) {
	pushSvc, clientSvc, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	sub := createPushSubscription(t, clientSvc, vaultID, userID, SyncWayPush)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Eve",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	remoteURI := "https://dav.example.com/contacts/" + contact.ID + ".vcf"
	pushSvc.db.Create(&models.ContactSubscriptionState{
		ContactID:                 contact.ID,
		AddressBookSubscriptionID: sub.ID,
		DistantURI:                remoteURI,
		DistantEtag:               "etag-eve",
	})

	var removeCalled bool
	var removePath string
	mc := &mockCardDAVClient{
		removeAllFn: func(ctx context.Context, path string) error {
			removeCalled = true
			removePath = path
			return nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactDelete(contact.ID, vaultID)

	if !removeCalled {
		t.Error("expected RemoveAll to be called")
	}
	if removePath != remoteURI {
		t.Errorf("expected remove path %q, got %q", remoteURI, removePath)
	}

	var state models.ContactSubscriptionState
	err = pushSvc.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error
	if err == nil {
		t.Error("expected ContactSubscriptionState to be deleted")
	}

	var logs []models.DavSyncLog
	pushSvc.db.Where("address_book_subscription_id = ? AND action = ?", sub.ID, "push_deleted").Find(&logs)
	if len(logs) != 1 {
		t.Errorf("expected 1 push_deleted log, got %d", len(logs))
	}
}

func TestDavPushService_PushContactDelete_NoState(t *testing.T) {
	pushSvc, _, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Frank",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	mc := &mockCardDAVClient{
		removeAllFn: func(ctx context.Context, path string) error {
			t.Error("RemoveAll should not be called when no state exists")
			return nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactDelete(contact.ID, vaultID)
}

func TestDavPushService_PushUpdatesPullOrigin(t *testing.T) {
	pushSvc, clientSvc, _, _, vaultID, userID, _ := setupDavPushTest(t)

	sub := createPushSubscription(t, clientSvc, vaultID, userID, SyncWayBoth)
	addressBookPath := "/addressbooks/user/default/"
	if err := pushSvc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("address_book_path", addressBookPath).Error; err != nil {
		t.Fatalf("failed to set address book path: %v", err)
	}

	distantURI := addressBookPath + "pulled-contact.vcf"
	distantUUID := "remote-contact-uid"
	contact := models.Contact{
		VaultID:     vaultID,
		FirstName:   strPtrOrNil("Pulled"),
		LastName:    strPtrOrNil("Contact"),
		DistantUUID: &distantUUID,
		DistantURI:  &distantURI,
	}
	pushSvc.db.Create(&contact)

	pushSvc.db.Create(&models.ContactVaultUser{
		ContactID: contact.ID,
		UserID:    userID,
		VaultID:   vaultID,
	})

	var putPath string
	var pushedUID string
	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			putPath = path
			pushedUID = card.Value(vcard.FieldUID)
			return &carddav.AddressObject{Path: path, ETag: "updated-etag"}, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)

	if putPath != distantURI {
		t.Fatalf("expected PUT to original URI %q, got %q", distantURI, putPath)
	}
	if pushedUID != distantUUID {
		t.Fatalf("expected original vCard UID %q, got %q", distantUUID, pushedUID)
	}

	var state models.ContactSubscriptionState
	if err := pushSvc.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error; err != nil {
		t.Fatalf("expected subscription state to be created: %v", err)
	}
	if state.DistantURI != distantURI || state.DistantEtag != "updated-etag" {
		t.Fatalf("unexpected subscription state: %+v", state)
	}
}

func TestDavURIHasBase(t *testing.T) {
	tests := []struct {
		name       string
		distantURI string
		base       string
		want       bool
	}{
		{name: "relative address book path", distantURI: "/addressbooks/user/default/card.vcf", base: "/addressbooks/user/default/", want: true},
		{name: "absolute same host", distantURI: "https://dav.example.com/addressbooks/user/default/card.vcf", base: "https://dav.example.com/", want: true},
		{name: "different host", distantURI: "https://other.example.com/addressbooks/card.vcf", base: "https://dav.example.com/", want: false},
		{name: "path boundary", distantURI: "/addressbooks/user/default-other/card.vcf", base: "/addressbooks/user/default/", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := davURIHasBase(tt.distantURI, tt.base); got != tt.want {
				t.Fatalf("davURIHasBase(%q, %q) = %v, want %v", tt.distantURI, tt.base, got, tt.want)
			}
		})
	}
}

func TestDavPushService_PushDoesNotSkipDifferentOrigin(t *testing.T) {
	pushSvc, clientSvc, _, _, vaultID, userID, _ := setupDavPushTest(t)

	createPushSubscription(t, clientSvc, vaultID, userID, SyncWayPush)

	distantURI := "https://other-server.example.com/contacts/other-contact.vcf"
	contact := models.Contact{
		VaultID:    vaultID,
		FirstName:  strPtrOrNil("Other"),
		LastName:   strPtrOrNil("Origin"),
		DistantURI: &distantURI,
	}
	pushSvc.db.Create(&contact)

	pushSvc.db.Create(&models.ContactVaultUser{
		ContactID: contact.ID,
		UserID:    userID,
		VaultID:   vaultID,
	})

	var putCalled bool
	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			putCalled = true
			return &carddav.AddressObject{
				Path: path,
				ETag: "new-etag",
			}, nil
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)

	if !putCalled {
		t.Error("expected PutAddressObject to be called for contact from different origin")
	}
}

func TestDavPushService_PushContactChange_RemoteError(t *testing.T) {
	pushSvc, clientSvc, _, contactSvc, vaultID, userID, _ := setupDavPushTest(t)

	sub := createPushSubscription(t, clientSvc, vaultID, userID, SyncWayPush)

	contact, err := contactSvc.CreateContact(vaultID, userID, dto.CreateContactRequest{
		FirstName: "Grace",
	})
	if err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	mc := &mockCardDAVClient{
		putAddrObjFn: func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
			return nil, fmt.Errorf("remote server unavailable")
		},
	}
	pushSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	pushSvc.PushContactChange(contact.ID, vaultID)

	var state models.ContactSubscriptionState
	err = pushSvc.db.Where("contact_id = ? AND address_book_subscription_id = ?", contact.ID, sub.ID).First(&state).Error
	if err == nil {
		t.Error("expected no ContactSubscriptionState to be created on error")
	}

	var logs []models.DavSyncLog
	pushSvc.db.Where("address_book_subscription_id = ? AND action = ?", sub.ID, "error").Find(&logs)
	if len(logs) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logs))
	}
	if logs[0].ErrorMessage == nil || *logs[0].ErrorMessage == "" {
		t.Error("expected error message in log")
	}
}
