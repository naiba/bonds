package services

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrVCardUnsafeReplacement = errors.New("vCard update would discard local data")

// ReplaceContactVCardFields preserves unchanged groups and replaces only groups
// whose stored information can be represented safely by the incoming vCard.
// Both subscription pulls and local CardDAV PUTs must use this path: a resumed
// subscription can contain merged records even if none were linked at merge time.
// The caller must include this operation in its contact update transaction.
func ReplaceContactVCardFields(db *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	var current models.Contact
	if err := preloadContactVCardRelations(db).First(&current, "id = ?", contactID).Error; err != nil {
		return err
	}
	previous := BuildContactVCard(&current)
	if card.Value(vcard.FieldVersion) == "3.0" {
		previous = BuildContactCardDAVV3(&current)
	}
	keepInformation := reflect.DeepEqual(previous[vcard.FieldTelephone], card[vcard.FieldTelephone]) && reflect.DeepEqual(previous[vcard.FieldEmail], card[vcard.FieldEmail])
	keepAddresses := reflect.DeepEqual(previous.Addresses(), card.Addresses())
	// A vCard cannot represent extra important dates, reminder links, address
	// history or all typed contact facts. An unchanged group must retain its rows;
	// destructive edits to richer groups require the full Bonds editor.
	updates := make(vcard.Card, len(card))
	for key, fields := range card {
		updates[key] = fields
	}
	if len(current.ImportantDates) > 0 {
		if previous.Value(vcard.FieldBirthday) != card.Value(vcard.FieldBirthday) {
			return fmt.Errorf("%w: edit important dates in Bonds to preserve dates and reminders", ErrVCardUnsafeReplacement)
		}
		delete(updates, vcard.FieldBirthday)
	}
	if keepInformation {
		delete(updates, vcard.FieldTelephone)
		delete(updates, vcard.FieldEmail)
	} else {
		for _, info := range current.ContactInformations {
			kind := ""
			if info.ContactInformationType.Type != nil {
				kind = *info.ContactInformationType.Type
			}
			if (kind == "phone" || kind == "email") && (info.Kind != nil || !info.Pref) {
				return fmt.Errorf("%w: edit typed contact information in Bonds to preserve its metadata", ErrVCardUnsafeReplacement)
			}
		}
		types := db.Model(&models.ContactInformationType{}).Select("id").Where("type IN ?", []string{"phone", "email"})
		if err := db.Where("contact_id = ? AND type_id IN (?)", contactID, types).Delete(&models.ContactInformation{}).Error; err != nil {
			return err
		}
	}
	if keepAddresses {
		delete(updates, vcard.FieldAddress)
	}

	// vCards do not carry coordinates, so the delete-and-recreate below would
	// erase every pin the server has geocoded — on every sync from a phone,
	// even one that never touched the addresses. Remember the coordinates by
	// what the address says, and put them back on recreated rows that still
	// say the same thing.
	coordinates := map[string][2]*float64{}
	var pivots []models.ContactAddress
	if err := db.Where("contact_id = ?", contactID).Find(&pivots).Error; err != nil {
		return err
	}
	if len(pivots) > 0 && !keepAddresses {
		addressIDs := make([]uint, len(pivots))
		for i, p := range pivots {
			if p.IsPastAddress || p.DateFrom != nil || p.DateTo != nil {
				return fmt.Errorf("%w: edit address history in Bonds", ErrVCardUnsafeReplacement)
			}
			var otherLinks int64
			if err := db.Model(&models.ContactAddress{}).Where("address_id = ? AND contact_id <> ?", p.AddressID, contactID).Count(&otherLinks).Error; err != nil {
				return err
			}
			if otherLinks > 0 {
				return fmt.Errorf("%w: edit shared addresses in Bonds", ErrVCardUnsafeReplacement)
			}
			addressIDs[i] = p.AddressID
		}
		var previous []models.Address
		if err := db.Where("id IN ?", addressIDs).Find(&previous).Error; err != nil {
			return err
		}
		for i := range previous {
			address := &previous[i]
			if address.AddressTypeID != nil || address.Line2 != nil {
				return fmt.Errorf("%w: edit detailed addresses in Bonds", ErrVCardUnsafeReplacement)
			}
			if address.Latitude == nil || address.Longitude == nil {
				continue
			}
			coordinates[addressCoordinateKey(address)] = [2]*float64{address.Latitude, address.Longitude}
		}
		if err := db.Where("contact_id = ?", contactID).Delete(&models.ContactAddress{}).Error; err != nil {
			return err
		}
		if err := db.Where("id IN ?", addressIDs).Delete(&models.Address{}).Error; err != nil {
			return err
		}
	}

	if err := importVCardFields(db, updates, contactID, vaultID, accountID); err != nil {
		return err
	}

	if len(coordinates) > 0 {
		var recreatedPivots []models.ContactAddress
		if err := db.Where("contact_id = ?", contactID).Find(&recreatedPivots).Error; err != nil {
			return err
		}
		if len(recreatedPivots) == 0 {
			return nil
		}
		recreatedIDs := make([]uint, len(recreatedPivots))
		for i, p := range recreatedPivots {
			recreatedIDs[i] = p.AddressID
		}
		var recreated []models.Address
		if err := db.Where("id IN ?", recreatedIDs).Find(&recreated).Error; err != nil {
			return err
		}
		for i := range recreated {
			address := &recreated[i]
			if address.Latitude != nil && address.Longitude != nil {
				continue
			}
			pair, known := coordinates[addressCoordinateKey(address)]
			if !known {
				continue
			}
			if err := db.Model(address).
				Select("latitude", "longitude").
				Updates(models.Address{Latitude: pair[0], Longitude: pair[1]}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// addressCoordinateKey identifies an address by what it says, so coordinates
// can survive a delete-and-recreate when the recreated row still describes
// the same place. Case and whitespace are ignored: they would not change what
// a geocoder was asked, so they do not make it a different address.
func addressCoordinateKey(address *models.Address) string {
	parts := make([]string, 0, 5)
	for _, part := range []*string{address.Line1, address.City, address.Province, address.PostalCode, address.Country} {
		if part == nil {
			parts = append(parts, "")
			continue
		}
		parts = append(parts, strings.ToLower(strings.Join(strings.Fields(*part), "")))
	}
	return strings.Join(parts, "|")
}
