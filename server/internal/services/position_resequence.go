package services

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type positionRow struct {
	ID       uint
	Position sql.NullInt64
}

func resequencePositionRows(tx *gorm.DB, table, idColumn, ownerWhere string, ownerArgs []interface{}, movedID uint, targetIndex int, notFound error) error {
	var rows []positionRow
	query := fmt.Sprintf(
		"SELECT %s AS id, position FROM %s WHERE %s ORDER BY CASE WHEN position IS NULL THEN 1 ELSE 0 END, position ASC, %s ASC",
		idColumn, table, ownerWhere, idColumn,
	)
	if err := tx.Raw(query, ownerArgs...).Scan(&rows).Error; err != nil {
		return err
	}

	oldIndex := -1
	for i, row := range rows {
		if row.ID == movedID {
			oldIndex = i
			break
		}
	}
	if oldIndex == -1 {
		return notFound
	}

	newIndex := targetIndex
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(rows) {
		newIndex = len(rows) - 1
	}

	moved := rows[oldIndex]
	rows = append(rows[:oldIndex], rows[oldIndex+1:]...)
	rows = append(rows[:newIndex], append([]positionRow{moved}, rows[newIndex:]...)...)

	// Rewriting all siblings that changed prevents duplicate positions from making the drag a no-op.
	now := time.Now()
	stmt := fmt.Sprintf("UPDATE %s SET position = ?, updated_at = ? WHERE %s = ? AND %s", table, idColumn, ownerWhere)
	for index, row := range rows {
		if row.Position.Valid && int(row.Position.Int64) == index {
			continue
		}
		args := append([]interface{}{index, now, row.ID}, ownerArgs...)
		if err := tx.Exec(stmt, args...).Error; err != nil {
			return err
		}
	}
	return nil
}
