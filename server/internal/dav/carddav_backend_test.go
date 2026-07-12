package dav

import (
	"context"
	"strings"
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

func mustDecodeAddressObjectCard(t *testing.T, obj carddav.AddressObject) vcard.Card {
	t.Helper()
	return obj.Card
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
	bob := createTestContact(t, db, vaultID, userID, "Bob", "Jones")

	var vault models.Vault
	if err := db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("load vault: %v", err)
	}
	var phoneType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", vault.AccountID, "phone").First(&phoneType).Error; err != nil {
		t.Fatalf("load phone type: %v", err)
	}
	var socialType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", vault.AccountID, "social").First(&socialType).Error; err != nil {
		t.Fatalf("load social type: %v", err)
	}
	var birthdateType models.ContactImportantDateType
	if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&birthdateType).Error; err != nil {
		t.Fatalf("load birthdate type: %v", err)
	}
	phoneValue := "+1-555-2222"
	socialValue := "https://twitter.com/bob_jones"
	photoURL := "https://example.com/bob.jpg"
	if err := db.Create(&models.ContactInformation{ContactID: bob.ID, TypeID: phoneType.ID, Data: phoneValue}).Error; err != nil {
		t.Fatalf("create phone info: %v", err)
	}
	if err := db.Create(&models.ContactInformation{ContactID: bob.ID, TypeID: socialType.ID, Data: socialValue}).Error; err != nil {
		t.Fatalf("create social info: %v", err)
	}
	if err := db.Create(&models.ContactImportantDate{
		ContactID:                  bob.ID,
		ContactImportantDateTypeID: &birthdateType.ID,
		Label:                      "Birthdate",
		Day:                        intPtr(2),
		Month:                      intPtr(3),
		Year:                       intPtr(1988),
	}).Error; err != nil {
		t.Fatalf("create birthday: %v", err)
	}
	if err := db.Create(&models.File{VaultID: vaultID, UUID: "bob-photo", OriginalURL: &photoURL, MimeType: "image/jpeg", Name: "bob.jpg", Type: "image", Size: 2345}).Error; err != nil {
		t.Fatalf("create photo: %v", err)
	}
	var storedPhoto models.File
	if err := db.Where("uuid = ?", "bob-photo").First(&storedPhoto).Error; err != nil {
		t.Fatalf("reload photo: %v", err)
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", bob.ID).Updates(map[string]any{
		"file_id":      storedPhoto.ID,
		"job_position": "Architect",
		"description":  "Long-time friend",
	}).Error; err != nil {
		t.Fatalf("update bob contact: %v", err)
	}

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
		card := mustDecodeAddressObjectCard(t, obj)
		if card.Value(vcard.FieldVersion) != "3.0" {
			t.Fatalf("expected VERSION 3.0, got %q", card.Value(vcard.FieldVersion))
		}
		if card.Value(vcard.FieldKind) != "" {
			t.Fatalf("expected KIND to be absent, got %q", card.Value(vcard.FieldKind))
		}
		if card.Value(vcard.FieldUID) == "" {
			t.Fatal("expected UID to be set")
		}
		fn := card.Value(vcard.FieldFormattedName)
		if fn == "Alice Smith" {
			foundAlice = true
		}
		if fn == "Bob Jones" {
			foundBob = true
			if card.Value(vcard.FieldTitle) != "Architect" {
				t.Fatalf("expected TITLE Architect, got %q", card.Value(vcard.FieldTitle))
			}
			if card.Value(vcard.FieldNote) != "Long-time friend" {
				t.Fatalf("expected NOTE Long-time friend, got %q", card.Value(vcard.FieldNote))
			}
			if card.Value(vcard.FieldBirthday) != "1988-03-02" {
				t.Fatalf("expected BDAY 1988-03-02, got %q", card.Value(vcard.FieldBirthday))
			}
			if card.Value(vcard.FieldTelephone) != phoneValue {
				t.Fatalf("expected TEL %q, got %q", phoneValue, card.Value(vcard.FieldTelephone))
			}
			if card.Value(vcard.FieldIMPP) != socialValue {
				t.Fatalf("expected IMPP %q, got %q", socialValue, card.Value(vcard.FieldIMPP))
			}
			if card.Value(vcard.FieldURL) != socialValue {
				t.Fatalf("expected URL %q, got %q", socialValue, card.Value(vcard.FieldURL))
			}
			if got := card["X-SOCIALPROFILE"]; len(got) != 1 || got[0].Value != socialValue {
				t.Fatalf("expected X-SOCIALPROFILE %q, got %#v", socialValue, got)
			}
			if card.Value(vcard.FieldPhoto) != photoURL {
				t.Fatalf("expected PHOTO %q, got %q", photoURL, card.Value(vcard.FieldPhoto))
			}
		}
		if obj.ETag == "" {
			t.Error("Expected non-empty ETag")
		}
		if !strings.HasPrefix(obj.ETag, "v3-") {
			t.Errorf("Expected v3 ETag marker, got %q", obj.ETag)
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

func TestListAddressObjects_ExcludesArchivedContacts(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	activeContact := createTestContact(t, db, vaultID, userID, "Active", "Person")
	archivedContact := createTestContact(t, db, vaultID, userID, "Archived", "Person")
	if err := db.Model(&models.Contact{}).Where("id = ?", archivedContact.ID).Update("listed", false).Error; err != nil {
		t.Fatalf("archive contact: %v", err)
	}

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/"

	// Given an active contact and an archived contact in the same vault.
	// When CardDAV lists the address book.
	objects, err := backend.ListAddressObjects(ctx, path, &carddav.AddressDataRequest{AllProp: true})
	if err != nil {
		t.Fatalf("ListAddressObjects failed: %v", err)
	}

	// Then only the active contact should be synced.
	if len(objects) != 1 {
		t.Fatalf("expected 1 address object after excluding archived contacts, got %d", len(objects))
	}
	card := mustDecodeAddressObjectCard(t, objects[0])
	if got := card.Value(vcard.FieldUID); got != activeContact.ID {
		t.Fatalf("expected active contact UID %q, got %q", activeContact.ID, got)
	}
	if got := card.Value(vcard.FieldFormattedName); got != "Active Person" {
		t.Fatalf("expected active contact FN, got %q", got)
	}
}

func TestGetAddressObject(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	contact := createTestContact(t, db, vaultID, userID, "Charlie", "Brown")
	var vault models.Vault
	if err := db.First(&vault, "id = ?", vaultID).Error; err != nil {
		t.Fatalf("load vault: %v", err)
	}
	var emailType models.ContactInformationType
	if err := db.Where("account_id = ? AND type = ?", vault.AccountID, "email").First(&emailType).Error; err != nil {
		t.Fatalf("load email type: %v", err)
	}
	if err := db.Create(&models.ContactInformation{ContactID: contact.ID, TypeID: emailType.ID, Data: "charlie@example.com"}).Error; err != nil {
		t.Fatalf("create email: %v", err)
	}
	if err := db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"job_position": "Engineer",
		"description":  "Enjoys hiking",
	}).Error; err != nil {
		t.Fatalf("update contact: %v", err)
	}

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contact.ID + ".vcf"
	obj, err := backend.GetAddressObject(ctx, path, &carddav.AddressDataRequest{AllProp: true})
	if err != nil {
		t.Fatalf("GetAddressObject failed: %v", err)
	}
	card := mustDecodeAddressObjectCard(t, *obj)
	if got := card.Value(vcard.FieldVersion); got != "3.0" {
		t.Fatalf("expected VERSION 3.0, got %q", got)
	}
	if got := card.Value(vcard.FieldKind); got != "" {
		t.Fatalf("expected KIND to be absent, got %q", got)
	}
	if got := card.Value(vcard.FieldUID); got != contact.ID {
		t.Fatalf("expected UID %q, got %q", contact.ID, got)
	}
	if got := card.Value(vcard.FieldTitle); got != "Engineer" {
		t.Fatalf("expected TITLE Engineer, got %q", got)
	}
	if got := card.Value(vcard.FieldNote); got != "Enjoys hiking" {
		t.Fatalf("expected NOTE Enjoys hiking, got %q", got)
	}
	if got := card.Value(vcard.FieldEmail); got != "charlie@example.com" {
		t.Fatalf("expected EMAIL charlie@example.com, got %q", got)
	}
	fn := card.Value(vcard.FieldFormattedName)
	if fn != "Charlie Brown" {
		t.Errorf("Expected FN 'Charlie Brown', got '%s'", fn)
	}

	name := card.Name()
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

func TestGetAddressObject_ReturnsNotFoundForArchivedContact(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	archivedContact := createTestContact(t, db, vaultID, userID, "Archived", "Person")
	if err := db.Model(&models.Contact{}).Where("id = ?", archivedContact.ID).Update("listed", false).Error; err != nil {
		t.Fatalf("archive contact: %v", err)
	}

	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + archivedContact.ID + ".vcf"

	// Given an archived contact path.
	// When CardDAV fetches the address object directly.
	obj, err := backend.GetAddressObject(ctx, path, &carddav.AddressDataRequest{AllProp: true})

	// Then archived contacts should not be exposed through direct DAV fetches.
	if err == nil {
		t.Fatalf("expected archived contact fetch to fail, got object %+v", obj)
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

func TestPutAddressObjectRejectsExistingContactFromDifferentVault(t *testing.T) {
	backend, db, ctx, sourceVaultID, userID := setupCardDAVTest(t)
	vaultSvc := services.NewVaultService(db)
	targetVault, err := vaultSvc.CreateVault(AccountIDFromContext(ctx), userID, dto.CreateVaultRequest{Name: "Target Vault"}, "en")
	if err != nil {
		t.Fatalf("Create target vault failed: %v", err)
	}

	contact := createTestContact(t, db, sourceVaultID, userID, "Alice", "Original")

	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "4.0")
	card.SetValue(vcard.FieldFormattedName, "Mallory Overwrite")
	card.SetName(&vcard.Name{
		GivenName:  "Mallory",
		FamilyName: "Overwrite",
	})

	path := "/dav/addressbooks/" + userID + "/" + targetVault.ID + "/" + contact.ID + ".vcf"
	_, err = backend.PutAddressObject(ctx, path, card, nil)
	if err == nil {
		t.Error("expected cross-vault contact update to be rejected")
	}

	var stored models.Contact
	if err := db.First(&stored, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("reload source contact: %v", err)
	}
	if stored.FirstName == nil || *stored.FirstName != "Alice" {
		t.Fatalf("expected source contact first name to remain Alice, got %v", stored.FirstName)
	}
	if stored.LastName == nil || *stored.LastName != "Original" {
		t.Fatalf("expected source contact last name to remain Original, got %v", stored.LastName)
	}

	var targetContactCount int64
	if err := db.Model(&models.Contact{}).
		Where("vault_id = ? AND first_name = ?", targetVault.ID, "Mallory").
		Count(&targetContactCount).Error; err != nil {
		t.Fatalf("count target contacts: %v", err)
	}
	if targetContactCount != 0 {
		t.Fatalf("expected no Mallory contact in target vault, got %d", targetContactCount)
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

func intPtr(value int) *int {
	return &value
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
