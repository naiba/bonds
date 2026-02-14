package models

import "time"

type Journal struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID     string    `json:"vault_id" gorm:"type:text;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Description *string   `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Vault          Vault           `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Posts          []Post          `json:"posts,omitempty" gorm:"foreignKey:JournalID"`
	SlicesOfLife   []SliceOfLife   `json:"slices_of_life,omitempty" gorm:"foreignKey:JournalID"`
	JournalMetrics []JournalMetric `json:"journal_metrics,omitempty" gorm:"foreignKey:JournalID"`
}

type JournalMetric struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	JournalID uint      `json:"journal_id" gorm:"not null;index"`
	Label     string    `json:"label" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Journal     Journal      `json:"journal,omitempty" gorm:"foreignKey:JournalID"`
	PostMetrics []PostMetric `json:"post_metrics,omitempty" gorm:"foreignKey:JournalMetricID"`
}
