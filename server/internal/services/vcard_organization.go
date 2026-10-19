package services

import (
	"errors"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// reconcileVCardOrganization runs inside the caller's contact transaction.
// ORG must populate the same jobs that normal UI writes and vCard exports use.
// Departments are not company names; split components before interpreting text.
func reconcileVCardOrganization(tx *gorm.DB, card vcard.Card, contactID, vaultID string) error {
	organizations := card[vcard.FieldOrganization]
	if len(organizations) == 0 && len(card[vcard.FieldTitle]) == 0 {
		return nil
	}
	if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
		return err
	}
	contact, err := loadContactEmployment(tx, contactID, vaultID)
	if err != nil {
		return err
	}
	if len(organizations) == 0 {
		// A title-only edit targets the first projected job, not a stale profile copy.
		employment := ContactVCardEmployments(contact)
		if len(employment) > 0 && employment[0].JobID != 0 {
			job := models.ContactCompany{ID: employment[0].JobID, ContactID: contactID, JobPosition: strPtrOrNil(card.Value(vcard.FieldTitle))}
			return updateContactRecord(tx, &job, contactID, vaultID, ErrContactJobNotFound, "job_position")
		}
		return nil
	}
	// Resolve the entire input before writing. A greedy per-field match lets an
	// inserted/edited first role consume an unchanged later role's identity.
	type incomingEmployment struct {
		name, title string
		jobID       uint
	}
	var incoming []incomingEmployment
	for _, field := range organizations {
		components := field.Organization()
		if len(components) == 0 || strings.TrimSpace(components[0]) == "" {
			continue
		}
		title := ""
		if fieldTitle := vcardOrganizationTitle(card, field.Group); fieldTitle != nil {
			title = strings.TrimSpace(fieldTitle.Value)
		}
		// Missing TITLE is an empty title, not permission to keep an old title.
		incoming = append(incoming, incomingEmployment{name: strings.TrimSpace(components[0]), title: title})
	}
	if len(incoming) == 0 && len(contact.ContactCompanies) > 0 {
		return ErrVCardUnsafeReplacement
	}
	projected := ContactVCardEmployments(contact)
	used := map[uint]bool{}
	for i := range incoming {
		for _, job := range projected {
			if job.JobID != 0 && !used[job.JobID] && job.Company == incoming[i].name && job.Title == incoming[i].title {
				incoming[i].jobID = job.JobID
				used[job.JobID] = true
				break
			}
		}
	}
	// Exact matches (including empty titles) are reserved first. Within a company
	// only one remaining input and one remaining job identify an edit. Multiple
	// candidates have no stable wire identity: reject rather than guess by order.
	for i := range incoming {
		if incoming[i].jobID != 0 {
			continue
		}
		var remaining []VCardEmployment
		for _, job := range projected {
			if job.JobID != 0 && !used[job.JobID] && job.Company == incoming[i].name {
				remaining = append(remaining, job)
			}
		}
		if len(remaining) == 0 {
			continue
		}
		unmatched := 0
		for _, entry := range incoming {
			if entry.jobID == 0 && entry.name == incoming[i].name {
				unmatched++
			}
		}
		if len(remaining) != 1 || unmatched != 1 {
			return ErrVCardUnsafeReplacement
		}
		// If an identical old role was reserved above, the wire cannot tell which
		// of those IDs was edited. Reserving one arbitrarily must not manufacture
		// a unique residual identity for the other.
		for _, job := range projected {
			if used[job.JobID] && job.Company == remaining[0].Company && job.Title == remaining[0].Title {
				return ErrVCardUnsafeReplacement
			}
		}
		incoming[i].jobID = remaining[0].JobID
		used[remaining[0].JobID] = true
	}
	// Reordering must retain the selected company when it is still present.
	// Replacing its ORG, however, selects the new company while preserving the
	// old employment, as the shared import/subscription contract requires.
	var primaryCompanyID *uint
	for _, job := range projected {
		if job.CompanyID == 0 {
			continue
		}
		if contact.CompanyID != nil && job.CompanyID != *contact.CompanyID {
			continue
		}
		for _, entry := range incoming {
			if entry.name == job.Company {
				id := job.CompanyID
				primaryCompanyID = &id
				break
			}
		}
		if primaryCompanyID != nil {
			break
		}
	}
	for _, entry := range incoming {
		var companyID uint
		if entry.jobID != 0 {
			for _, job := range projected {
				if job.JobID != entry.jobID {
					continue
				}
				companyID = job.CompanyID
				if job.Title != entry.title {
					updated := models.ContactCompany{ID: job.JobID, ContactID: contactID, JobPosition: strPtrOrNil(entry.title)}
					if err := updateContactRecord(tx, &updated, contactID, vaultID, ErrContactJobNotFound, "job_position"); err != nil {
						return err
					}
				}
				break
			}
		} else {
			// Company names are not identities. Reuse the associated company even
			// when its stored name has whitespace; never choose between two local
			// same-name company identities for an unidentifiable new role.
			for _, job := range projected {
				if job.Company != entry.name {
					continue
				}
				if companyID != 0 && companyID != job.CompanyID {
					return ErrVCardUnsafeReplacement
				}
				companyID = job.CompanyID
			}
			if companyID == 0 {
				var company models.Company
				err := tx.Where("vault_id = ? AND name = ?", vaultID, entry.name).Order("id ASC").First(&company).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					company = models.Company{VaultID: vaultID, Name: entry.name}
					if err := tx.Create(&company).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				}
				companyID = company.ID
			}
			job := models.ContactCompany{ContactID: contactID, CompanyID: companyID, JobPosition: strPtrOrNil(entry.title)}
			if err := createContactRecord(tx, &job, contactID, vaultID); err != nil {
				return err
			}
		}
		if primaryCompanyID == nil {
			id := companyID
			primaryCompanyID = &id
		}
	}
	if len(incoming) == 0 {
		primaryCompanyID = nil
	}
	contact.CompanyID = primaryCompanyID
	return updateContactProfile(tx, contact, vaultID, "company_id")
}

// vCard group names are case-insensitive, just like property names.
func vcardOrganizationTitle(card vcard.Card, group string) *vcard.Field {
	for _, title := range card[vcard.FieldTitle] {
		if strings.EqualFold(title.Group, group) {
			return title
		}
	}
	return nil
}
