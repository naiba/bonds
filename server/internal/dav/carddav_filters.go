package dav

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
)

// Filter before encoding photos: presence rules need only already-loaded contact
// fields, and must not retain or read image payloads for excluded contacts.
func matchesCardDAVFilter(contact *models.Contact, filter *models.CardDAVFilter) bool {
	if !contact.Listed {
		return false
	}
	if filter == nil || len(filter.Fields) == 0 {
		return true
	}
	if filter.Match != "all" && filter.Match != "any" {
		return false
	}
	for _, field := range filter.Fields {
		present := cardDAVFieldPresent(contact, field)
		if filter.Match == "all" && !present {
			return false
		}
		if filter.Match == "any" && present {
			return true
		}
	}
	return filter.Match == "all"
}

func cardDAVFieldPresent(contact *models.Contact, field string) bool {
	nonblank := func(value *string) bool { return value != nil && strings.TrimSpace(*value) != "" }
	switch field {
	case "first_name":
		return nonblank(contact.FirstName)
	case "last_name":
		return nonblank(contact.LastName)
	case "nickname":
		return nonblank(contact.Nickname)
	case "job_title":
		return nonblank(contact.JobPosition)
	case "organization":
		return contact.Company != nil && strings.TrimSpace(contact.Company.Name) != ""
	case "phone", "email":
		for _, info := range contact.ContactInformations {
			if ptrToStr(info.ContactInformationType.Type) == field && strings.TrimSpace(info.Data) != "" {
				return true
			}
		}
	case "address":
		for _, address := range contact.Addresses {
			// Use only address components actually exported by BuildContactVCard.
			if nonblank(address.Line1) || nonblank(address.City) || nonblank(address.Province) || nonblank(address.PostalCode) || nonblank(address.Country) {
				return true
			}
		}
	}
	return false
}

func (b *CardDAVBackend) contactMatchesVaultFilter(contact *models.Contact) (bool, error) {
	var vault models.Vault
	if err := b.db.First(&vault, "id = ?", contact.VaultID).Error; err != nil {
		return false, err
	}
	return matchesCardDAVFilter(contact, vault.CardDAVFilter), nil
}
