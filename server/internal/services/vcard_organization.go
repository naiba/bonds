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
	var primaryCompanyID *uint
	used := map[uint]bool{}
	for _, field := range organizations {
		components := field.Organization()
		if len(components) == 0 || strings.TrimSpace(components[0]) == "" {
			continue
		}
		name := strings.TrimSpace(components[0])
		title := vcardOrganizationTitle(card, field.Group)
		// Match the same normalized projection we exported before a name lookup.
		// Names are not identities: whitespace or another same-name company must
		// never turn an unchanged vCard into a new company or employment record.
		var companyID, knownCompanyID uint
		for _, employment := range ContactVCardEmployments(contact) {
			if employment.Company != name {
				continue
			}
			if knownCompanyID == 0 {
				knownCompanyID = employment.CompanyID
			}
			if used[employment.JobID] {
				continue
			}
			if companyID == 0 {
				companyID = employment.CompanyID
			}
			if title != nil && employment.Title == strings.TrimSpace(title.Value) {
				companyID = employment.CompanyID
				break
			}
		}
		// Consuming a job does not consume its company: another role can reuse it.
		if companyID == 0 {
			companyID = knownCompanyID
		}
		var company models.Company
		query := tx.Where("vault_id = ?", vaultID)
		if companyID != 0 {
			query = query.Where("id = ?", companyID)
		} else {
			query = query.Where("name = ?", name).Order("id ASC")
		}
		err := query.First(&company).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			company = models.Company{VaultID: vaultID, Name: name}
			if err := tx.Create(&company).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		var selected *models.ContactCompany
		var candidates []*models.ContactCompany
		for i := range contact.ContactCompanies {
			job := &contact.ContactCompanies[i]
			if job.CompanyID != company.ID || used[job.ID] {
				continue
			}
			candidates = append(candidates, job)
			if title != nil && strings.TrimSpace(ptrToStr(job.JobPosition)) == strings.TrimSpace(title.Value) {
				selected = job
				break
			}
		}
		// Several jobs at one company have no unique wire identifier. Preserve them
		// unless an exact title identifies one; a new title becomes a separate job.
		if selected == nil && (len(candidates) == 1 || (title == nil && len(candidates) > 0)) {
			selected = candidates[0]
		}
		if selected == nil {
			job := models.ContactCompany{ContactID: contactID, CompanyID: company.ID}
			if title != nil {
				job.JobPosition = strPtrOrNil(title.Value)
			}
			if err := createContactRecord(tx, &job, contactID, vaultID); err != nil {
				return err
			}
			job.Company = company
			contact.ContactCompanies = append(contact.ContactCompanies, job)
			selected = &contact.ContactCompanies[len(contact.ContactCompanies)-1]
		} else if title != nil && strings.TrimSpace(ptrToStr(selected.JobPosition)) != strings.TrimSpace(title.Value) {
			selected.JobPosition = strPtrOrNil(title.Value)
			if err := updateContactRecord(tx, selected, contactID, vaultID, ErrContactJobNotFound, "job_position"); err != nil {
				return err
			}
		}
		used[selected.ID] = true
		if primaryCompanyID == nil {
			id := company.ID
			primaryCompanyID = &id
		}
	}
	if primaryCompanyID == nil && len(contact.ContactCompanies) > 0 {
		// An empty ORG cannot identify which existing employment to remove. Do not
		// claim it cleared the company while jobs still export it, or erase local jobs.
		return ErrVCardUnsafeReplacement
	}
	contact.CompanyID = primaryCompanyID
	return updateContactProfile(tx, contact, vaultID, "company_id")
}

func vcardOrganizationTitle(card vcard.Card, group string) *vcard.Field {
	for _, title := range card[vcard.FieldTitle] {
		if title.Group == group {
			return title
		}
	}
	return nil
}
