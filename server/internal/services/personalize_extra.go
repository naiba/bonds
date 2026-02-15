package services

import (
	"fmt"
)

// UpdatePosition updates the position of an entity in a personalize table
func (s *PersonalizeService) UpdatePosition(accountID, entity string, id uint, position int) error {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return ErrUnknownEntityType
	}

	result := s.db.Exec(
		fmt.Sprintf("UPDATE %s SET position = ?, updated_at = datetime('now') WHERE id = ? AND account_id = ?", cfg.table),
		position, id, accountID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPersonalizeEntityNotFound
	}
	return nil
}
