package dav

import (
	"context"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

func setupCardDAVTest(t *testing.T) (*CardDAVBackend, *gorm.DB, context.Context, string, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	authSvc := services.NewAuthService(db, cfg)
	vaultSvc := services.NewVaultService(db)

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

	backend := NewCardDAVBackend(db)
	ctx := WithUserID(context.Background(), resp.User.ID)
	ctx = WithAccountID(ctx, resp.User.AccountID)

	return backend, db, ctx, vault.ID, resp.User.ID
}

func createTestContact(t *testing.T, db *gorm.DB, vaultID, userID, firstName, lastName string) *models.Contact {
	t.Helper()
	now := time.Now()
	fn := firstName
	ln := lastName
	contact := models.Contact{
		VaultID:       vaultID,
		FirstName:     &fn,
		LastName:      &ln,
		LastUpdatedAt: &now,
	}
	if err := db.Create(&contact).Error; err != nil {
		t.Fatalf("CreateContact failed: %v", err)
	}

	cvu := models.ContactVaultUser{
		ContactID: contact.ID,
		UserID:    userID,
		VaultID:   vaultID,
	}
	if err := db.Create(&cvu).Error; err != nil {
		t.Fatalf("CreateContactVaultUser failed: %v", err)
	}

	return &contact
}

// Verify the CardDAVBackend implements the carddav.Backend interface at compile time.
var _ carddav.Backend = (*CardDAVBackend)(nil)

func TestListAddressBooks(t *testing.T) {
	backend, _, ctx, _, _ := setupCardDAVTest(t)

	books, err := backend.ListAddressBooks(ctx)
	if err != nil {
		t.Fatalf("ListAddressBooks failed: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("Expected 1 address book, got %d", len(books))
	}
	if books[0].Name != "Test Vault" {
		t.Errorf("Expected name 'Test Vault', got '%s'", books[0].Name)
	}
}

func TestGetAddressBook(t *testing.T) {
	backend, _, ctx, vaultID, userID := setupCardDAVTest(t)

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/"
	book, err := backend.GetAddressBook(ctx, path)
	if err != nil {
		t.Fatalf("GetAddressBook failed: %v", err)
	}
	if book.Name != "Test Vault" {
		t.Errorf("Expected name 'Test Vault', got '%s'", book.Name)
	}
}

func TestListAddressObjects(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	createTestContact(t, db, vaultID, userID, "Alice", "Smith")
	createTestContact(t, db, vaultID, userID, "Bob", "Jones")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/"
	objects, err := backend.ListAddressObjects(ctx, path, &carddav.AddressDataRequest{AllProp: true})
	if err != nil {
		t.Fatalf("ListAddressObjects failed: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("Expected 2 address objects, got %d", len(objects))
	}

	// Check vCard content
	foundAlice := false
	foundBob := false
	for _, obj := range objects {
		fn := obj.Card.Value(vcard.FieldFormattedName)
		if fn == "Alice Smith" {
			foundAlice = true
		}
		if fn == "Bob Jones" {
			foundBob = true
		}
		if obj.ETag == "" {
			t.Error("Expected non-empty ETag")
		}
		if obj.Path == "" {
			t.Error("Expected non-empty Path")
		}
	}
	if !foundAlice {
		t.Error("Expected to find Alice Smith in address objects")
	}
	if !foundBob {
		t.Error("Expected to find Bob Jones in address objects")
	}
}

func TestGetAddressObject(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Charlie", "Brown")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contact.ID + ".vcf"
	obj, err := backend.GetAddressObject(ctx, path, &carddav.AddressDataRequest{AllProp: true})
	if err != nil {
		t.Fatalf("GetAddressObject failed: %v", err)
	}
	fn := obj.Card.Value(vcard.FieldFormattedName)
	if fn != "Charlie Brown" {
		t.Errorf("Expected FN 'Charlie Brown', got '%s'", fn)
	}

	name := obj.Card.Name()
	if name == nil {
		t.Fatal("Expected N field to be set")
	}
	if name.GivenName != "Charlie" {
		t.Errorf("Expected given name 'Charlie', got '%s'", name.GivenName)
	}
	if name.FamilyName != "Brown" {
		t.Errorf("Expected family name 'Brown', got '%s'", name.FamilyName)
	}
}

func TestPutAddressObject(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetValue(vcard.FieldFormattedName, "Dave Wilson")
	card.SetName(&vcard.Name{
		GivenName:  "Dave",
		FamilyName: "Wilson",
	})
	card.SetValue(vcard.FieldNickname, "Davey")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/new-contact.vcf"
	obj, err := backend.PutAddressObject(ctx, path, card, nil)
	if err != nil {
		t.Fatalf("PutAddressObject failed: %v", err)
	}
	if obj.ETag == "" {
		t.Error("Expected non-empty ETag")
	}

	// Verify in DB
	var contacts []models.Contact
	if err := db.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		t.Fatalf("DB query failed: %v", err)
	}
	found := false
	for _, c := range contacts {
		if c.FirstName != nil && *c.FirstName == "Dave" {
			found = true
			if c.LastName == nil || *c.LastName != "Wilson" {
				t.Error("Expected last name 'Wilson'")
			}
			if c.Nickname == nil || *c.Nickname != "Davey" {
				t.Error("Expected nickname 'Davey'")
			}
		}
	}
	if !found {
		t.Error("Expected to find contact 'Dave' in database")
	}
}

func TestDeleteAddressObject(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Eve", "Delete")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contact.ID + ".vcf"

	if err := backend.DeleteAddressObject(ctx, path); err != nil {
		t.Fatalf("DeleteAddressObject failed: %v", err)
	}

	// Verify deleted
	var c models.Contact
	err := db.First(&c, "id = ?", contact.ID).Error
	if err == nil {
		t.Error("Expected contact to be deleted")
	}
}

func TestQueryAddressObjects(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	createTestContact(t, db, vaultID, userID, "Alice", "Smith")
	createTestContact(t, db, vaultID, userID, "Bob", "Jones")

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/"

	// Query for all
	objects, err := backend.QueryAddressObjects(ctx, path, &carddav.AddressBookQuery{})
	if err != nil {
		t.Fatalf("QueryAddressObjects failed: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("Expected 2 objects, got %d", len(objects))
	}

	// Query with filter
	objects, err = backend.QueryAddressObjects(ctx, path, &carddav.AddressBookQuery{
		PropFilters: []carddav.PropFilter{
			{
				Name: vcard.FieldFormattedName,
				TextMatches: []carddav.TextMatch{
					{Text: "Alice", MatchType: carddav.MatchContains},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("QueryAddressObjects with filter failed: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("Expected 1 filtered object, got %d", len(objects))
	}
}

func TestCurrentUserPrincipal(t *testing.T) {
	backend, _, ctx, _, userID := setupCardDAVTest(t)

	path, err := backend.CurrentUserPrincipal(ctx)
	if err != nil {
		t.Fatalf("CurrentUserPrincipal failed: %v", err)
	}
	expected := "/dav/principals/" + userID + "/"
	if path != expected {
		t.Errorf("Expected '%s', got '%s'", expected, path)
	}
}

func TestAddressBookHomeSetPath(t *testing.T) {
	backend, _, ctx, _, userID := setupCardDAVTest(t)

	path, err := backend.AddressBookHomeSetPath(ctx)
	if err != nil {
		t.Fatalf("AddressBookHomeSetPath failed: %v", err)
	}
	expected := "/dav/addressbooks/" + userID + "/"
	if path != expected {
		t.Errorf("Expected '%s', got '%s'", expected, path)
	}
}

func TestCreateAddressBookNotSupported(t *testing.T) {
	backend, _, ctx, _, _ := setupCardDAVTest(t)

	err := backend.CreateAddressBook(ctx, &carddav.AddressBook{})
	if err == nil {
		t.Error("Expected error for CreateAddressBook")
	}
}

func TestDeleteAddressBookNotSupported(t *testing.T) {
	backend, _, ctx, _, _ := setupCardDAVTest(t)

	err := backend.DeleteAddressBook(ctx, "/dav/addressbooks/x/y/")
	if err == nil {
		t.Error("Expected error for DeleteAddressBook")
	}
}
