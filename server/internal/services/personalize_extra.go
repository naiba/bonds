package services

import "gorm.io/gorm"

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
	if !cfg.hasPosition {
		return ErrPersonalizeEntityNotSortable
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		return resequencePositionRows(tx, cfg.table, "id", "account_id = ?", []interface{}{accountID}, id, position, ErrPersonalizeEntityNotFound)
	})
}
