package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
)

type mockCardDAVClient struct {
	findPrincipalFn func(ctx context.Context) (string, error)
	findHomeSetFn   func(ctx context.Context, principal string) (string, error)
	findAddrBooksFn func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error)
	syncCollFn      func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error)
	multiGetFn      func(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error)
	queryFn         func(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error)
	getAddrObjFn    func(ctx context.Context, path string) (*carddav.AddressObject, error)
	putAddrObjFn    func(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error)
	removeAllFn     func(ctx context.Context, path string) error
}

func (m *mockCardDAVClient) FindCurrentUserPrincipal(ctx context.Context) (string, error) {
	if m.findPrincipalFn != nil {
		return m.findPrincipalFn(ctx)
	}
	return "", nil
}

func (m *mockCardDAVClient) FindAddressBookHomeSet(ctx context.Context, principal string) (string, error) {
	if m.findHomeSetFn != nil {
		return m.findHomeSetFn(ctx, principal)
	}
	return "/addressbooks/user/", nil
}

func (m *mockCardDAVClient) FindAddressBooks(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
	if m.findAddrBooksFn != nil {
		return m.findAddrBooksFn(ctx, homeSet)
	}
	return []carddav.AddressBook{{Path: "/addressbooks/user/default/", Name: "Default"}}, nil
}

func (m *mockCardDAVClient) SyncCollection(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
	if m.syncCollFn != nil {
		return m.syncCollFn(ctx, path, query)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockCardDAVClient) MultiGetAddressBook(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
	if m.multiGetFn != nil {
		return m.multiGetFn(ctx, path, mg)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockCardDAVClient) QueryAddressBook(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, path, query)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockCardDAVClient) GetAddressObject(ctx context.Context, path string) (*carddav.AddressObject, error) {
	if m.getAddrObjFn != nil {
		return m.getAddrObjFn(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockCardDAVClient) PutAddressObject(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error) {
	if m.putAddrObjFn != nil {
		return m.putAddrObjFn(ctx, path, card)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockCardDAVClient) RemoveAll(ctx context.Context, path string) error {
	if m.removeAllFn != nil {
		return m.removeAllFn(ctx, path)
	}
	return fmt.Errorf("not implemented")
}

type mockCardDAVClientFactory struct {
	client CardDAVClient
	err    error
}

func (f *mockCardDAVClientFactory) NewClient(uri, username, password string) (CardDAVClient, error) {
	return f.client, f.err
}

func setupDavSyncTest(t *testing.T) (*DavSyncService, *DavClientService, *VCardService, string, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := NewAuthService(db, cfg)
	vaultSvc := NewVaultService(db)

	resp, err := authSvc.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "User",
		Email:     "dav-sync-test@example.com",
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
	syncSvc := NewDavSyncService(db, clientSvc, vcardSvc)

	return syncSvc, clientSvc, vcardSvc, vault.ID, resp.User.ID, resp.User.AccountID
}

func makeVCard(firstName, lastName, uid string) vcard.Card {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetName(&vcard.Name{
		GivenName:  firstName,
		FamilyName: lastName,
	})
	card.SetValue(vcard.FieldFormattedName, firstName+" "+lastName)
	if uid != "" {
		card.SetValue("UID", uid)
	}
	return card
}

func TestDavSyncService_TestConnection(t *testing.T) {
	syncSvc, _, _, _, _, _ := setupDavSyncTest(t)

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "/principals/user/", nil
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			return "/addressbooks/user/", nil
		},
		findAddrBooksFn: func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
			return []carddav.AddressBook{
				{Path: "/addressbooks/user/contacts/", Name: "Contacts"},
				{Path: "/addressbooks/user/work/", Name: "Work"},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.TestConnection(dto.TestDavConnectionRequest{
		URI:      "https://dav.example.com/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.AddressBooks) != 2 {
		t.Errorf("expected 2 address books, got %d", len(result.AddressBooks))
	}
	if result.AddressBooks[0].Name != "Contacts" {
		t.Errorf("expected first address book name 'Contacts', got %q", result.AddressBooks[0].Name)
	}
	if result.AddressBooks[0].Path != "/addressbooks/user/contacts/" {
		t.Errorf("expected first address book path '/addressbooks/user/contacts/', got %q", result.AddressBooks[0].Path)
	}
}

func TestDavSyncService_TestConnection_Failure(t *testing.T) {
	syncSvc, _, _, _, _, _ := setupDavSyncTest(t)

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "/principals/user/", nil
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			return "", fmt.Errorf("connection refused")
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.TestConnection(dto.TestDavConnectionRequest{
		URI:      "https://bad.example.com/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("TestConnection returned unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestDavSyncService_SyncSubscription_FullSync(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	objects := []carddav.AddressObject{
		{
			Path: "/contacts/alice.vcf",
			ETag: "etag-alice-1",
			Card: makeVCard("Alice", "Smith", "uid-alice"),
		},
		{
			Path: "/contacts/bob.vcf",
			ETag: "etag-bob-1",
			Card: makeVCard("Bob", "Jones", "uid-bob"),
		},
	}

	mc := &mockCardDAVClient{
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return nil, fmt.Errorf("sync not supported")
		},
		queryFn: func(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			return objects, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}
	if result.Created != 2 {
		t.Errorf("expected 2 created, got %d", result.Created)
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}

	var contacts []models.Contact
	syncSvc.db.Where("vault_id = ? AND distant_uri IS NOT NULL", vaultID).Find(&contacts)
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts in DB, got %d", len(contacts))
	}

	found := make(map[string]bool)
	for _, c := range contacts {
		found[ptrToStr(c.FirstName)] = true
	}
	if !found["Alice"] {
		t.Error("expected Alice contact")
	}
	if !found["Bob"] {
		t.Error("expected Bob contact")
	}
}

func TestDavSyncService_SyncSubscription_IncrementalSync(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	token := "sync-token-1"
	syncSvc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("distant_sync_token", token)

	existingContact := models.Contact{
		VaultID:     vaultID,
		FirstName:   strPtrOrNil("Old"),
		LastName:    strPtrOrNil("Name"),
		DistantURI:  strPtrOrNil("/contacts/existing.vcf"),
		DistantEtag: strPtrOrNil("old-etag"),
	}
	syncSvc.db.Create(&existingContact)

	mc := &mockCardDAVClient{
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			if query.SyncToken != token {
				t.Errorf("expected sync token %q, got %q", token, query.SyncToken)
			}
			return &carddav.SyncResponse{
				SyncToken: "sync-token-2",
				Updated: []carddav.AddressObject{
					{Path: "/contacts/existing.vcf", ETag: "new-etag"},
					{Path: "/contacts/new-contact.vcf", ETag: "etag-new"},
				},
			}, nil
		},
		multiGetFn: func(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			var result []carddav.AddressObject
			for _, p := range mg.Paths {
				switch p {
				case "/contacts/existing.vcf":
					result = append(result, carddav.AddressObject{
						Path: p,
						ETag: "new-etag",
						Card: makeVCard("Updated", "Name", "uid-existing"),
					})
				case "/contacts/new-contact.vcf":
					result = append(result, carddav.AddressObject{
						Path: p,
						ETag: "etag-new",
						Card: makeVCard("New", "Contact", "uid-new"),
					})
				}
			}
			return result, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("expected 1 updated, got %d", result.Updated)
	}

	var updatedContact models.Contact
	syncSvc.db.Where("distant_uri = ?", "/contacts/existing.vcf").First(&updatedContact)
	if ptrToStr(updatedContact.FirstName) != "Updated" {
		t.Errorf("expected updated first name 'Updated', got %q", ptrToStr(updatedContact.FirstName))
	}
	if ptrToStr(updatedContact.DistantEtag) != "new-etag" {
		t.Errorf("expected new etag, got %q", ptrToStr(updatedContact.DistantEtag))
	}
}

func TestDavSyncService_SyncSubscription_Deletion(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	token := "sync-token-1"
	syncSvc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Update("distant_sync_token", token)

	contact1 := models.Contact{
		VaultID:     vaultID,
		FirstName:   strPtrOrNil("ToDelete"),
		DistantURI:  strPtrOrNil("/contacts/delete-me.vcf"),
		DistantEtag: strPtrOrNil("etag1"),
	}
	syncSvc.db.Create(&contact1)

	contact2 := models.Contact{
		VaultID:     vaultID,
		FirstName:   strPtrOrNil("KeepMe"),
		DistantURI:  strPtrOrNil("/contacts/keep.vcf"),
		DistantEtag: strPtrOrNil("etag2"),
	}
	syncSvc.db.Create(&contact2)

	mc := &mockCardDAVClient{
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return &carddav.SyncResponse{
				SyncToken: "sync-token-2",
				Deleted:   []string{"/contacts/delete-me.vcf"},
			}, nil
		},
		multiGetFn: func(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			return nil, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}
	if result.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", result.Deleted)
	}

	var deletedContact models.Contact
	err = syncSvc.db.Unscoped().Where("id = ?", contact1.ID).First(&deletedContact).Error
	if err != nil {
		t.Fatalf("failed to find soft-deleted contact: %v", err)
	}
	if !deletedContact.DeletedAt.Valid {
		t.Error("expected contact to be soft-deleted")
	}

	var keptContact models.Contact
	err = syncSvc.db.Where("id = ?", contact2.ID).First(&keptContact).Error
	if err != nil {
		t.Fatalf("kept contact should still be findable: %v", err)
	}
}

func TestDavSyncService_GetSyncLogs(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	contactID := "test-contact-id"
	for i := 0; i < 5; i++ {
		syncSvc.logSyncAction(sub.ID, &contactID, fmt.Sprintf("/contacts/%d.vcf", i), "etag", "created", "")
	}

	logs, meta, err := syncSvc.GetSyncLogs(sub.ID, vaultID, 1, 3)
	if err != nil {
		t.Fatalf("GetSyncLogs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
	if meta.Total != 5 {
		t.Errorf("expected total 5, got %d", meta.Total)
	}
	if meta.TotalPages != 2 {
		t.Errorf("expected 2 pages, got %d", meta.TotalPages)
	}

	_, _, err = syncSvc.GetSyncLogs("nonexistent", vaultID, 1, 10)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestDavSyncService_SyncConflictLocalWins(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	token := "sync-token-1"
	pastTime := time.Now().Add(-2 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	syncSvc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Updates(map[string]interface{}{
		"distant_sync_token":   token,
		"last_synchronized_at": pastTime,
	})

	existingContact := models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil("LocalEdit"),
		LastName:      strPtrOrNil("User"),
		DistantURI:    strPtrOrNil("/contacts/conflict.vcf"),
		DistantEtag:   strPtrOrNil("old-etag"),
		LastUpdatedAt: &futureTime,
	}
	syncSvc.db.Create(&existingContact)

	mc := &mockCardDAVClient{
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return &carddav.SyncResponse{
				SyncToken: "sync-token-2",
				Updated: []carddav.AddressObject{
					{Path: "/contacts/conflict.vcf", ETag: "new-remote-etag"},
				},
			}, nil
		},
		multiGetFn: func(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			return []carddav.AddressObject{
				{
					Path: "/contacts/conflict.vcf",
					ETag: "new-remote-etag",
					Card: makeVCard("RemoteOverwrite", "Name", "uid-conflict"),
				},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (conflict), got %d", result.Skipped)
	}
	if result.Updated != 0 {
		t.Errorf("expected 0 updated, got %d", result.Updated)
	}

	var contact models.Contact
	syncSvc.db.Where("distant_uri = ?", "/contacts/conflict.vcf").First(&contact)
	if ptrToStr(contact.FirstName) != "LocalEdit" {
		t.Errorf("expected local name preserved 'LocalEdit', got %q", ptrToStr(contact.FirstName))
	}
	if ptrToStr(contact.DistantEtag) != "old-etag" {
		t.Errorf("expected old etag preserved, got %q", ptrToStr(contact.DistantEtag))
	}

	var logs []models.DavSyncLog
	syncSvc.db.Where("address_book_subscription_id = ? AND action = ?", sub.ID, "conflict_local_wins").Find(&logs)
	if len(logs) != 1 {
		t.Errorf("expected 1 conflict_local_wins log, got %d", len(logs))
	}
}

func TestDavSyncService_SyncNoConflictWhenNotLocallyModified(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/contacts/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	token := "sync-token-1"
	syncTime := time.Now().Add(1 * time.Hour)
	beforeSync := time.Now().Add(-2 * time.Hour)

	syncSvc.db.Model(&models.AddressBookSubscription{}).Where("id = ?", sub.ID).Updates(map[string]interface{}{
		"distant_sync_token":   token,
		"last_synchronized_at": syncTime,
	})

	existingContact := models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil("Old"),
		LastName:      strPtrOrNil("Name"),
		DistantURI:    strPtrOrNil("/contacts/no-conflict.vcf"),
		DistantEtag:   strPtrOrNil("old-etag"),
		LastUpdatedAt: &beforeSync,
	}
	syncSvc.db.Create(&existingContact)

	mc := &mockCardDAVClient{
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return &carddav.SyncResponse{
				SyncToken: "sync-token-2",
				Updated: []carddav.AddressObject{
					{Path: "/contacts/no-conflict.vcf", ETag: "new-etag"},
				},
			}, nil
		},
		multiGetFn: func(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error) {
			return []carddav.AddressObject{
				{
					Path: "/contacts/no-conflict.vcf",
					ETag: "new-etag",
					Card: makeVCard("Updated", "Remote", "uid-no-conflict"),
				},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}

	if result.Updated != 1 {
		t.Errorf("expected 1 updated, got %d", result.Updated)
	}

	var contact models.Contact
	syncSvc.db.Where("distant_uri = ?", "/contacts/no-conflict.vcf").First(&contact)
	if ptrToStr(contact.FirstName) != "Updated" {
		t.Errorf("expected remote update applied 'Updated', got %q", ptrToStr(contact.FirstName))
	}
}

func TestDavSyncService_TestConnection_WithPrincipal(t *testing.T) {
	syncSvc, _, _, _, _, _ := setupDavSyncTest(t)

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "/principals/user/", nil
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			if principal != "/principals/user/" {
				t.Errorf("expected principal '/principals/user/', got %q", principal)
			}
			return "/addressbooks/user/", nil
		},
		findAddrBooksFn: func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
			return []carddav.AddressBook{
				{Path: "/addressbooks/user/contacts/", Name: "Contacts"},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.TestConnection(dto.TestDavConnectionRequest{
		URI:      "https://dav.example.com/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.AddressBooks) != 1 {
		t.Errorf("expected 1 address book, got %d", len(result.AddressBooks))
	}
}

func TestDavSyncService_TestConnection_PrincipalFallback(t *testing.T) {
	syncSvc, _, _, _, _, _ := setupDavSyncTest(t)

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "", fmt.Errorf("current-user-principal not supported")
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			if principal != "" {
				t.Errorf("expected empty principal on fallback, got %q", principal)
			}
			return "/addressbooks/user/", nil
		},
		findAddrBooksFn: func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
			return []carddav.AddressBook{
				{Path: "/addressbooks/user/contacts/", Name: "Contacts"},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.TestConnection(dto.TestDavConnectionRequest{
		URI:      "https://simple.example.com/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success on fallback, got error: %s", result.Error)
	}
	if len(result.AddressBooks) != 1 {
		t.Errorf("expected 1 address book, got %d", len(result.AddressBooks))
	}
}

func TestDavSyncService_TestConnection_SabreDAVPath(t *testing.T) {
	syncSvc, _, _, _, _, _ := setupDavSyncTest(t)

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "/dav.php/principals/user/", nil
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			if principal != "/dav.php/principals/user/" {
				t.Errorf("expected SabreDAV principal '/dav.php/principals/user/', got %q", principal)
			}
			return "/dav.php/addressbooks/user/", nil
		},
		findAddrBooksFn: func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
			if homeSet != "/dav.php/addressbooks/user/" {
				t.Errorf("expected SabreDAV homeSet '/dav.php/addressbooks/user/', got %q", homeSet)
			}
			return []carddav.AddressBook{
				{Path: "/dav.php/addressbooks/user/default/", Name: "Default"},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.TestConnection(dto.TestDavConnectionRequest{
		URI:      "https://baikal.example.com/dav.php/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.AddressBooks) != 1 {
		t.Errorf("expected 1 address book, got %d", len(result.AddressBooks))
	}
	if result.AddressBooks[0].Name != "Default" {
		t.Errorf("expected 'Default', got %q", result.AddressBooks[0].Name)
	}
	if result.AddressBooks[0].Path != "/dav.php/addressbooks/user/default/" {
		t.Errorf("expected path '/dav.php/addressbooks/user/default/', got %q", result.AddressBooks[0].Path)
	}
}

func TestDavSyncService_SyncSubscription_DiscoverPath(t *testing.T) {
	syncSvc, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)

	// Create subscription with URI only — no AddressBookPath
	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI:      "https://dav.example.com/",
		Username: "user",
		Password: "pwd",
	})
	if err != nil {
		t.Fatalf("Create subscription failed: %v", err)
	}

	// Verify AddressBookPath is initially empty
	var initialSub models.AddressBookSubscription
	syncSvc.db.First(&initialSub, "id = ?", sub.ID)
	if initialSub.AddressBookPath != "" {
		t.Fatalf("expected empty AddressBookPath initially, got %q", initialSub.AddressBookPath)
	}

	discoveredPath := "/addressbooks/user/main/"
	var queryPathReceived string

	mc := &mockCardDAVClient{
		findPrincipalFn: func(ctx context.Context) (string, error) {
			return "/principals/user/", nil
		},
		findHomeSetFn: func(ctx context.Context, principal string) (string, error) {
			return "/addressbooks/user/", nil
		},
		findAddrBooksFn: func(ctx context.Context, homeSet string) ([]carddav.AddressBook, error) {
			return []carddav.AddressBook{
				{Path: discoveredPath, Name: "Main"},
			}, nil
		},
		syncCollFn: func(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error) {
			return nil, fmt.Errorf("sync not supported")
		},
		queryFn: func(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
			queryPathReceived = path
			return []carddav.AddressObject{
				{
					Path: "/addressbooks/user/main/alice.vcf",
					ETag: "etag-alice",
					Card: makeVCard("Alice", "Discover", "uid-alice-disc"),
				},
			}, nil
		},
	}
	syncSvc.SetClientFactory(&mockCardDAVClientFactory{client: mc})

	result, err := syncSvc.SyncSubscription(context.Background(), sub.ID, vaultID)
	if err != nil {
		t.Fatalf("SyncSubscription failed: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}

	// Verify QueryAddressBook received the DISCOVERED path, not the URI
	if queryPathReceived != discoveredPath {
		t.Errorf("expected QueryAddressBook to use discovered path %q, got %q", discoveredPath, queryPathReceived)
	}

	// Verify AddressBookPath was saved to DB
	var updatedSub models.AddressBookSubscription
	syncSvc.db.First(&updatedSub, "id = ?", sub.ID)
	if updatedSub.AddressBookPath != discoveredPath {
		t.Errorf("expected AddressBookPath cached as %q, got %q", discoveredPath, updatedSub.AddressBookPath)
	}
}
