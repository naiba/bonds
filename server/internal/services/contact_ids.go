package services

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxContactIDsPerRequest = 500

var (
	ErrContactIDsLimitExceeded = errors.New("contact IDs limit exceeded")
	ErrContactIDInvalid        = errors.New("contact ID is invalid")
)

func validateAndDedupeContactIDs(contactIDs []string) ([]string, error) {
	// Enforce the limit before deduplication so repeated IDs cannot bypass it.
	if len(contactIDs) > maxContactIDsPerRequest {
		return nil, ErrContactIDsLimitExceeded
	}
	for _, contactID := range contactIDs {
		parsedID, err := uuid.Parse(contactID)
		if err != nil || parsedID == uuid.Nil {
			return nil, ErrContactIDInvalid
		}
	}
	return dedupeContactIDs(contactIDs), nil
}

// lockContactsBelongToVault must run inside the caller's write transaction,
// before inserting associations or locking their rows. Validation outside that
// transaction cannot prevent a merge from deleting the owner before insertion.
// Use the same contact-ID order as merging to avoid opposing parent locks.
func lockContactsBelongToVault(tx *gorm.DB, contactIDs []string, vaultID string) error {
	lockedContactIDs := dedupeContactIDs(append([]string(nil), contactIDs...))
	if len(lockedContactIDs) == 0 {
		return nil
	}
	sort.Strings(lockedContactIDs)

	var contacts []models.Contact
	if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id IN ? AND vault_id = ?", lockedContactIDs, vaultID).
		Order("id ASC").
		Find(&contacts).Error; err != nil {
		return err
	}
	if len(contacts) != len(lockedContactIDs) {
		return ErrContactNotFound
	}
	return nil
}

// LockContactForWrite lets protocol adapters use the same ownership boundary as
// service writes. The caller must keep the transaction open through all child
// writes; a pre-transaction contact lookup is only discovery, not validation.
func LockContactForWrite(tx *gorm.DB, contactID, vaultID string) error {
	return lockContactsBelongToVault(tx, []string{contactID}, vaultID)
}
