package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UpdatePosition moves entity `id` to index `position` in the account's
// ordered list for the given entity type. The whole list is re-sequenced
// (0..N-1) inside a single transaction so positions stay contiguous and
// unique — the previous implementation only set the moved row's column,
// which produced ties and left the rendered order unchanged when the
// (position ASC, id ASC) tiebreaker kept the original order.
func (s *PersonalizeService) UpdatePosition(accountID, entity string, id uint, position int) error {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return ErrUnknownEntityType
	}

	type orderedRow struct {
		ID       uint
		Position int
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var rows []orderedRow
		if err := tx.Raw(
			fmt.Sprintf("SELECT id, position FROM %s WHERE account_id = ? ORDER BY position ASC, id ASC", cfg.table),
			accountID,
		).Scan(&rows).Error; err != nil {
			return err
		}

		oldIdx := -1
		for i, r := range rows {
			if r.ID == id {
				oldIdx = i
				break
			}
		}
		if oldIdx == -1 {
			return ErrPersonalizeEntityNotFound
		}

		newIdx := position
		if newIdx < 0 {
			newIdx = 0
		}
		if newIdx >= len(rows) {
			newIdx = len(rows) - 1
		}
		if oldIdx == newIdx {
			return nil
		}

		moved := rows[oldIdx]
		rows = append(rows[:oldIdx], rows[oldIdx+1:]...)
		rows = append(rows[:newIdx], append([]orderedRow{moved}, rows[newIdx:]...)...)

		now := time.Now()
		stmt := fmt.Sprintf("UPDATE %s SET position = ?, updated_at = ? WHERE id = ? AND account_id = ?", cfg.table)
		for i, r := range rows {
			if r.Position == i {
				continue
			}
			if err := tx.Exec(stmt, i, now, r.ID, accountID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
