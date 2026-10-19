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

// ContactVCardEmployments prefers the normal jobs model. Legacy fields are a
// fallback for old contacts and standalone imported titles, never a second copy
// that can override an edited job. Keep all jobs visible, in deterministic order.
func ContactVCardEmployments(contact *models.Contact) []VCardEmployment {
	result := make([]VCardEmployment, 0, len(contact.ContactCompanies))
	for _, job := range contact.ContactCompanies {
		if job.Company.VaultID != "" && job.Company.VaultID != contact.VaultID {
			continue
		}
		result = append(result, VCardEmployment{Company: strings.TrimSpace(job.Company.Name), Title: strings.TrimSpace(ptrToStr(job.JobPosition)), JobID: job.ID, CompanyID: job.CompanyID})
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
	if len(contact.ContactCompanies) == 0 {
		employment := VCardEmployment{Title: strings.TrimSpace(ptrToStr(contact.JobPosition))}
		if contact.Company != nil && (contact.Company.VaultID == "" || contact.Company.VaultID == contact.VaultID) {
			employment.Company = strings.TrimSpace(contact.Company.Name)
			employment.CompanyID = contact.Company.ID
		}
		if employment.Company != "" || employment.Title != "" {
			result = append(result, employment)
		}
	}
	return result
}

// loadContactEmployment loads only the relations needed for employment identity.
func loadContactEmployment(db *gorm.DB, contactID, vaultID string) (*models.Contact, error) {
	var contact models.Contact
	err := db.Preload("Company").Preload("ContactCompanies").Preload("ContactCompanies.Company").
		First(&contact, "id = ? AND vault_id = ?", contactID, vaultID).Error
	return &contact, err
}
