package services

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
)

func contactVCardJob(contact *models.Contact) (organization, title string) {
	// Job information is stored in ContactCompany, not the legacy Contact
	// fields. Project the most recently created entry with its own title;
	// mixing in a legacy title would incorrectly pair two different jobs (#297).
	var latest *models.ContactCompany
	for i := range contact.ContactCompanies {
		job := &contact.ContactCompanies[i]
		if latest == nil || job.CreatedAt.After(latest.CreatedAt) ||
			(job.CreatedAt.Equal(latest.CreatedAt) && job.ID > latest.ID) {
			latest = job
		}
	}
	if latest != nil {
		// Broken or historical cross-vault references cannot disclose a company.
		if latest.Company.ID == 0 || latest.Company.VaultID != contact.VaultID {
			return "", ""
		}
		return strings.TrimSpace(latest.Company.Name), ptrToStr(latest.JobPosition)
	}
	// Preserve contacts imported or created before multiple jobs were supported.
	if contact.Company != nil && contact.Company.VaultID == contact.VaultID {
		organization = strings.TrimSpace(contact.Company.Name)
	}
	return organization, ptrToStr(contact.JobPosition)
}
