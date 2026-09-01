package dav

import (
	"errors"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// A phone syncing a contact back rewrites every address row, and vCards do
// not carry coordinates — so without the carry-over, any CardDAV write erased
// every pin the server had geocoded, even when the sync never touched the
// addresses. These tests pin the carry-over: same address, same pin; moved
// address, no stale pin.
func TestPutAddressObjectKeepsCoordinatesOfUnchangedAddresses(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	newCard := func(street, city, country string) vcard.Card {
		card := make(vcard.Card)
		card.SetValue(vcard.FieldVersion, "3.0")
		card.SetValue(vcard.FieldFormattedName, "Pinned Person")
		card.SetName(&vcard.Name{GivenName: "Pinned", FamilyName: "Person"})
		card.AddAddress(&vcard.Address{StreetAddress: street, Locality: city, Country: country})
		return card
	}

	if _, err := backend.PutAddressObject(ctx,
		"/dav/addressbooks/"+userID+"/"+vaultID+"/pinned-person.vcf",
		newCard("10 Downing Street", "London", "United Kingdom"), nil); err != nil {
		t.Fatalf("initial PutAddressObject failed: %v", err)
	}
	// A real client PUTs back to the path it was given, which carries the
	// contact's actual id.
	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Pinned").First(&contact).Error; err != nil {
		t.Fatalf("loading created contact: %v", err)
	}
	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contact.ID + ".vcf"

	loadAddress := func() models.Address {
		t.Helper()
		var pivot models.ContactAddress
		if err := db.Where("contact_id = ?", contact.ID).First(&pivot).Error; err != nil {
			t.Fatalf("loading address pivot: %v", err)
		}
		var address models.Address
		if err := db.First(&address, pivot.AddressID).Error; err != nil {
			t.Fatalf("loading address: %v", err)
		}
		return address
	}

	// The server geocoded the address at some point after the first sync.
	latitude, longitude := 51.5072, -0.1276
	if err := db.Model(&models.Address{}).Where("id = ?", loadAddress().ID).
		Updates(models.Address{Latitude: &latitude, Longitude: &longitude}).Error; err != nil {
		t.Fatalf("staging coordinates: %v", err)
	}

	// The phone syncs the same contact back — same address, different case and
	// spacing, which is how phones routinely normalise fields.
	if _, err := backend.PutAddressObject(ctx, path, newCard("10 Downing Street ", "LONDON", "United Kingdom"), nil); err != nil {
		t.Fatalf("second PutAddressObject failed: %v", err)
	}

	address := loadAddress()
	if address.Latitude == nil || *address.Latitude != 51.5072 || address.Longitude == nil || *address.Longitude != -0.1276 {
		t.Fatalf("a sync of the same address must keep its coordinates, got %v, %v", address.Latitude, address.Longitude)
	}

	// The phone moves the address: the old pin is wrong and must not survive.
	if _, err := backend.PutAddressObject(ctx, path, newCard("Stephansplatz 1", "Vienna", "Austria"), nil); err != nil {
		t.Fatalf("third PutAddressObject failed: %v", err)
	}
	address = loadAddress()
	if address.Latitude != nil || address.Longitude != nil {
		t.Fatalf("a moved address must not keep the old pin, got %v, %v", address.Latitude, address.Longitude)
	}
}

func TestPutAddressObjectRollsBackFailedFieldReplacement(t *testing.T) {
	backend, db, ctx, vaultID, userID := setupCardDAVTest(t)

	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")
	card.SetValue(vcard.FieldFormattedName, "Pinned Person")
	card.SetName(&vcard.Name{GivenName: "Pinned", FamilyName: "Person"})
	card.AddAddress(&vcard.Address{StreetAddress: "10 Downing Street", Locality: "London", Country: "United Kingdom"})
	if _, err := backend.PutAddressObject(ctx,
		"/dav/addressbooks/"+userID+"/"+vaultID+"/pinned-person.vcf", card, nil); err != nil {
		t.Fatalf("initial PutAddressObject failed: %v", err)
	}

	var contact models.Contact
	if err := db.Where("vault_id = ? AND first_name = ?", vaultID, "Pinned").First(&contact).Error; err != nil {
		t.Fatalf("loading created contact: %v", err)
	}
	var originalPivot models.ContactAddress
	if err := db.Where("contact_id = ?", contact.ID).First(&originalPivot).Error; err != nil {
		t.Fatalf("loading original address pivot: %v", err)
	}
	latitude, longitude := 51.5072, -0.1276
	if err := db.Model(&models.Address{}).Where("id = ?", originalPivot.AddressID).
		Updates(models.Address{Latitude: &latitude, Longitude: &longitude}).Error; err != nil {
		t.Fatalf("staging coordinates: %v", err)
	}

	injectedErr := errors.New("injected address create failure")
	callbackName := "test:fail_recreated_address"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "addresses" {
			tx.AddError(injectedErr)
		}
	}); err != nil {
		t.Fatalf("registering failure callback: %v", err)
	}
	defer func() {
		if err := db.Callback().Create().Remove(callbackName); err != nil {
			t.Errorf("removing failure callback: %v", err)
		}
	}()

	card.SetValue(vcard.FieldFormattedName, "Changed Person")
	card.SetName(&vcard.Name{GivenName: "Changed", FamilyName: "Person"})
	path := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contact.ID + ".vcf"
	if _, err := backend.PutAddressObject(ctx, path, card, nil); !errors.Is(err, injectedErr) {
		t.Fatalf("expected injected replacement error, got %v", err)
	}

	var persistedContact models.Contact
	if err := db.First(&persistedContact, "id = ?", contact.ID).Error; err != nil {
		t.Fatalf("reloading contact after rollback: %v", err)
	}
	if persistedContact.FirstName == nil || *persistedContact.FirstName != "Pinned" {
		t.Fatalf("contact update was not rolled back, got first name %v", persistedContact.FirstName)
	}
	var persistedPivot models.ContactAddress
	if err := db.Where("contact_id = ?", contact.ID).First(&persistedPivot).Error; err != nil {
		t.Fatalf("loading address pivot after rollback: %v", err)
	}
	if persistedPivot.AddressID != originalPivot.AddressID {
		t.Fatalf("address replacement was not rolled back, got address id %d, want %d", persistedPivot.AddressID, originalPivot.AddressID)
	}
	var persistedAddress models.Address
	if err := db.First(&persistedAddress, originalPivot.AddressID).Error; err != nil {
		t.Fatalf("loading original address after rollback: %v", err)
	}
	if persistedAddress.Latitude == nil || *persistedAddress.Latitude != latitude ||
		persistedAddress.Longitude == nil || *persistedAddress.Longitude != longitude {
		t.Fatalf("original coordinates were not restored by rollback, got %v, %v", persistedAddress.Latitude, persistedAddress.Longitude)
	}
}
