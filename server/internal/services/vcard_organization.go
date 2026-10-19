package services

import (
	"errors"
	"strings"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// reconcileVCardOrganization runs inside the caller's contact transaction.
// ORG must populate both the exported primary company and the employment model:
// ignoring it made valid CardDAV creates fail the organization presence filter.
func reconcileVCardOrganization(tx *gorm.DB, card vcard.Card, contactID, vaultID string) error {
	if len(card[vcard.FieldOrganization]) == 0 {
		// Older clients omit ORG; omission must not erase a local company selection.
		return nil
	}
	if err := lockContactsBelongToVault(tx, []string{contactID}, vaultID); err != nil {
		return err
	}
	contact := models.Contact{ID: contactID}
	name := strings.TrimSpace(card.Value(vcard.FieldOrganization))
	if name != "" {
		// Companies are shared within a Vault. Select or create, never rename an
		// existing company or link an identically named company from another Vault.
		var company models.Company
		err := tx.Where("vault_id = ? AND name = ?", vaultID, name).Order("id").First(&company).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			company = models.Company{VaultID: vaultID, Name: name}
			if err := tx.Create(&company).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		contact.CompanyID = &company.ID
		var job models.ContactCompany
		err = tx.Where("contact_id = ? AND company_id = ?", contactID, company.ID).First(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			job = models.ContactCompany{ContactID: contactID, CompanyID: company.ID, JobPosition: strPtrOrNil(card.Value(vcard.FieldTitle))}
			if err := createContactRecord(tx, &job, contactID, vaultID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	// A vCard has no employment IDs. Changing or explicitly clearing its primary
	// company must preserve other jobs and their locally maintained details.
	return updateContactProfile(tx, &contact, vaultID, "company_id")
}
