package services

import (
	"sort"
	"strings"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// VCardEmployment is the common projection used by exports and presence filters.
type VCardEmployment struct {
	Company   string
	Title     string
	JobID     uint
	CompanyID uint
}

// ContactVCardEmployments prefers identified Jobs, while a title with neither
// a company nor a selected Job is an independent employment. In particular,
// merging Jobs into that contact must not hide the reviewed standalone title.
func ContactVCardEmployments(contact *models.Contact) []VCardEmployment {
	result := make([]VCardEmployment, 0, len(contact.ContactCompanies))
	for _, job := range contact.ContactCompanies {
		if job.Company.VaultID != "" && job.Company.VaultID != contact.VaultID {
			continue
		}
		result = append(result, VCardEmployment{Company: strings.TrimSpace(job.Company.Name), Title: strings.TrimSpace(ptrToStr(job.JobPosition)), JobID: job.ID, CompanyID: job.CompanyID})
	}
	// Older records can already contain both a legacy employer and unrelated
	// Jobs. Preserve that employer on reads too, before any write materializes it.
	if len(contact.ContactCompanies) == 0 || hasStandaloneEmployment(contact) || hasUnrepresentedLegacyCompany(contact) {
		employment := VCardEmployment{Title: strings.TrimSpace(ptrToStr(contact.JobPosition))}
		if contact.Company != nil && (contact.Company.VaultID == "" || contact.Company.VaultID == contact.VaultID) {
			employment.Company = strings.TrimSpace(contact.Company.Name)
			employment.CompanyID = contact.Company.ID
		}
		if employment.Company != "" || employment.Title != "" {
			result = append(result, employment)
		}
	}
	// Company and title are mutable and several jobs can share both. Persisted
	// job identity prevents a secondary edit from retargeting the next DAV PUT.
	sort.SliceStable(result, func(i, j int) bool {
		if contact.PreferredJobID != nil && (result[i].JobID == *contact.PreferredJobID) != (result[j].JobID == *contact.PreferredJobID) {
			return result[i].JobID == *contact.PreferredJobID
		}
		if contact.CompanyID != nil && (result[i].CompanyID == *contact.CompanyID) != (result[j].CompanyID == *contact.CompanyID) {
			return result[i].CompanyID == *contact.CompanyID
		}
		return result[i].JobID < result[j].JobID
	})

	return result
}

// hasStandaloneEmployment distinguishes an independent title from a normal
// Job's compatibility mirror. No synthetic company or duplicate job is needed.
func hasStandaloneEmployment(contact *models.Contact) bool {
	return contact.CompanyID == nil && contact.PreferredJobID == nil && strings.TrimSpace(ptrToStr(contact.JobPosition)) != ""
}

// loadContactEmployment loads only the relations needed for employment identity.
func loadContactEmployment(db *gorm.DB, contactID, vaultID string) (*models.Contact, error) {
	var contact models.Contact
	err := db.Preload("Company").Preload("ContactCompanies").Preload("ContactCompanies.Company").
		First(&contact, "id = ? AND vault_id = ?", contactID, vaultID).Error
	return &contact, err
}

// hasUnrepresentedLegacyCompany identifies a real old relationship, not a
// compatibility mirror of an existing Job. Names and titles are mutable, so
// company identity alone determines whether Jobs already express it.
func hasUnrepresentedLegacyCompany(contact *models.Contact) bool {
	if contact.CompanyID == nil || contact.Company == nil || contact.Company.VaultID != contact.VaultID {
		return false
	}
	for _, job := range contact.ContactCompanies {
		if job.CompanyID == *contact.CompanyID {
			return false
		}
	}
	return true
}

// materializeLegacyEmployment preserves an independent old employer before
// editing or combining Jobs. The read projection keeps old mixed records
// visible; mutation also needs a real job identity so later edits stay targeted.
// Existing mirrors are not copied. Call with employment relations loaded and
// the parent contact locked, inside the transaction that combines the records.
func materializeLegacyEmployment(tx *gorm.DB, contact *models.Contact) error {
	if !hasUnrepresentedLegacyCompany(contact) {
		return nil
	}
	job := models.ContactCompany{ContactID: contact.ID, CompanyID: contact.Company.ID, JobPosition: contact.JobPosition}
	if err := createContactRecord(tx, &job, contact.ID, contact.VaultID); err != nil {
		return err
	}
	job.Company = *contact.Company
	contact.ContactCompanies = append(contact.ContactCompanies, job)
	return nil
}
