package services

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrVCardUnsafeReplacement = errors.New("vCard update would discard local data")

// ReplaceContactVCardFields reconciles the DAV projection record by record.
// Whole-group comparisons made harmless phone parameter reordering, adding a
// birthday beside an unrelated date, or adding an address block all DAV updates.
// Both subscription pulls and CardDAV PUT must run this in their transaction.
func ReplaceContactVCardFields(db *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	var current models.Contact
	if err := preloadContactVCardRelations(db).First(&current, "id = ? AND vault_id = ?", contactID, vaultID).Error; err != nil {
		return err
	}
	if err := reconcileVCardBirthday(db, card, &current); err != nil {
		return err
	}
	if err := reconcileVCardInformation(db, card, &current, accountID); err != nil {
		return err
	}
	return reconcileVCardAddresses(db, card, &current)
}

func reconcileVCardBirthday(db *gorm.DB, card vcard.Card, current *models.Contact) error {
	birthday := contactBirthdayImportantDate(current.ImportantDates)
	incoming := strings.TrimSpace(card.Value(vcard.FieldBirthday))
	// Partial dates have no BDAY projection, so omission cannot mean deletion.
	if birthday != nil && (birthday.Month == nil || birthday.Day == nil) && incoming == "" {
		return nil
	}
	if birthday == nil {
		if incoming == "" {
			return nil
		}
		year, month, day := parseBirthdayString(incoming)
		if month == 0 || day == 0 || !isValidReminderMonthDay(month, day) || (year > 0 && !isValidReminderDay(year, month, day)) {
			return ErrVCardInvalidData
		}
		return ImportContactVCardFields(db, vcard.Card{vcard.FieldBirthday: card[vcard.FieldBirthday]}, current.ID, current.VaultID, "")
	}
	year, month, day := parseBirthdayString(incoming)
	if incoming != "" && (month == 0 || day == 0 || !isValidReminderMonthDay(month, day) || (year > 0 && !isValidReminderDay(year, month, day))) {
		return ErrVCardInvalidData
	}
	if incoming != "" && birthday.Month != nil && birthday.Day != nil && *birthday.Month == month && *birthday.Day == day && ((birthday.Year == nil && year == 0) || (birthday.Year != nil && *birthday.Year == year)) {
		return nil
	}
	// The Gregorian projection cannot edit a lunar source or a separately synced
	// calendar event. Only that birthday is protected, never unrelated dates.
	if (birthday.CalendarType != "" && birthday.CalendarType != "gregorian") || birthday.IsAgeBased || birthday.DistantURI != nil || birthday.Vcalendar != nil {
		return fmt.Errorf("%w: edit this birthday's calendar data in Bonds", ErrVCardUnsafeReplacement)
	}
	if incoming == "" {
		return NewImportantDateService(db).delete(birthday.ID, current.ID, current.VaultID)
	}
	birthday.Year = nil
	birthday.Month, birthday.Day = &month, &day
	birthday.DatePrecision = importantDatePrecisionMonthDay
	birthday.IsYearUnknown = year == 0
	if year != 0 {
		birthday.Year = &year
		birthday.DatePrecision = importantDatePrecisionFull
	}
	if err := db.Omit("ContactImportantDateType").Save(birthday).Error; err != nil {
		return err
	}
	// Keep reminder identities, recipients and delivery history; pending schedules
	// must follow the edited birthday instead of continuing to notify on its old day.
	var reminders []models.ContactReminder
	if err := db.Where("contact_id = ? AND important_date_id = ?", current.ID, birthday.ID).Find(&reminders).Error; err != nil {
		return err
	}
	for i := range reminders {
		reminder := &reminders[i]
		reminder.Year, reminder.Month, reminder.Day = birthday.Year, birthday.Month, birthday.Day
		if err := db.Save(reminder).Error; err != nil {
			return err
		}
		if err := reschedulePendingReminder(db, reminder); err != nil {
			return err
		}
	}
	return nil
}

func reconcileVCardInformation(db *gorm.DB, card vcard.Card, current *models.Contact, accountID string) error {
	for _, group := range []struct{ field, kind string }{{vcard.FieldTelephone, "phone"}, {vcard.FieldEmail, "email"}} {
		var infoType models.ContactInformationType
		if err := db.Where("account_id = ? AND type = ?", accountID, group.kind).First(&infoType).Error; err != nil {
			return err
		}
		retained := map[uint]bool{}
		for _, field := range card[group.field] {
			if field.Value == "" {
				continue
			}
			kind, preferred := vcardInformationMetadata(field, group.kind)
			var match *models.ContactInformation
			// Semantic matching keeps IDs and user kind aliases across v3/v4 wire
			// normalization. An email edit must never recreate unrelated phone rows.
			for i := range current.ContactInformations {
				info := &current.ContactInformations[i]
				if retained[info.ID] || ptrToStr(info.ContactInformationType.Type) != group.kind || info.Data != field.Value {
					continue
				}
				if match == nil {
					match = info
				}
				if normalizedInformationKind(info, group.kind) == kind && info.Pref == preferred {
					match = info
					break
				}
			}
			if match != nil {
				retained[match.ID] = true
				if normalizedInformationKind(match, group.kind) == kind && match.Pref == preferred {
					continue
				}
				if err := db.Model(&models.ContactInformation{}).Where("id = ?", match.ID).Updates(map[string]any{"kind": strPtrOrNil(kind), "pref": preferred}).Error; err != nil {
					return err
				}
			} else {
				info := models.ContactInformation{ContactID: current.ID, TypeID: infoType.ID, Data: field.Value, Kind: strPtrOrNil(kind), Pref: preferred}
				if err := db.Create(&info).Error; err != nil {
					return err
				}
				if !preferred {
					if err := db.Model(&info).Update("pref", false).Error; err != nil {
						return err
					}
				}
			}
		}
		for _, info := range current.ContactInformations {
			if ptrToStr(info.ContactInformationType.Type) == group.kind && !retained[info.ID] {
				if err := db.Delete(&info).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func normalizedInformationKind(info *models.ContactInformation, group string) string {
	mapper := phoneKindToVCardTypes
	if group == "email" {
		mapper = emailKindToVCardTypes
	}
	kind, _ := vcardInformationMetadata(&vcard.Field{Params: contactInformationParams(info, mapper)}, group)
	return kind
}

func vcardInformationMetadata(field *vcard.Field, group string) (string, bool) {
	types := map[string]bool{}
	preferred := field.Params.Get(vcard.ParamPreferred) == "1"
	for _, value := range field.Params[vcard.ParamType] {
		for _, token := range strings.Split(value, ",") {
			types[strings.ToLower(strings.TrimSpace(token))] = true
		}
	}
	preferred = preferred || types["pref"]
	delete(types, "pref")
	delete(types, "internet")
	delete(types, "voice")
	if types["cell"] {
		delete(types, "cell")
		types["mobile"] = true
	}
	// Kind is a user-facing label. Standard composite phone kinds use the same
	// spelling as the Bonds editor and exporter.
	if group == "phone" && types["fax"] {
		if types["home"] {
			return "home fax", preferred
		}
		if types["work"] {
			return "work fax", preferred
		}
	}
	tokens := make([]string, 0, len(types))
	for token := range types {
		tokens = append(tokens, token)
	}
	slices.Sort(tokens)
	return strings.Join(tokens, ","), preferred
}

func reconcileVCardAddresses(db *gorm.DB, card vcard.Card, current *models.Contact) error {
	var pivots []models.ContactAddress
	if err := db.Where("contact_id = ?", current.ID).Find(&pivots).Error; err != nil {
		return err
	}
	retained := map[uint]bool{}
	pending := make(vcard.Card)
	for _, incoming := range card.Addresses() {
		found := false
		for _, address := range current.Addresses {
			if retained[address.ID] {
				continue
			}
			if sameVCardAddress(&address, incoming) {
				retained[address.ID], found = true, true
				break
			}
		}
		if !found {
			pending.AddAddress(incoming)
		}
	}
	for _, pivot := range pivots {
		if retained[pivot.AddressID] {
			continue
		}
		var address models.Address
		if err := db.First(&address, pivot.AddressID).Error; err != nil {
			return err
		}
		// Reject only the removed record whose unrepresented detail would be lost.
		// Unchanged history/shared rows do not prohibit adding or editing other ADRs.
		if pivot.IsPastAddress || pivot.DateFrom != nil || pivot.DateTo != nil || address.AddressTypeID != nil || address.Line2 != nil {
			return fmt.Errorf("%w: edit this address's details or history in Bonds", ErrVCardUnsafeReplacement)
		}
		if err := db.Delete(&pivot).Error; err != nil {
			return err
		}
		var links int64
		if err := db.Model(&models.ContactAddress{}).Where("address_id = ?", address.ID).Count(&links).Error; err != nil {
			return err
		}
		// Removing a shared link must never remove another contact's address.
		if links == 0 {
			if err := db.Delete(&address).Error; err != nil {
				return err
			}
		}
	}
	return ImportContactVCardFields(db, pending, current.ID, current.VaultID, "")
}

func sameVCardAddress(address *models.Address, incoming *vcard.Address) bool {
	// Client casing/whitespace normalization must retain geocoded coordinates.
	for _, pair := range [][2]string{{ptrToStr(address.Line1), incoming.StreetAddress}, {ptrToStr(address.City), incoming.Locality}, {ptrToStr(address.Province), incoming.Region}, {ptrToStr(address.PostalCode), incoming.PostalCode}, {ptrToStr(address.Country), incoming.Country}} {
		if !strings.EqualFold(strings.Join(strings.Fields(pair[0]), " "), strings.Join(strings.Fields(pair[1]), " ")) {
			return false
		}
	}
	return true
}
