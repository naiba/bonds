package models

import "time"

type LifeMetric struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	Label     string    `json:"label" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Vault Vault `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
}

// ContactLifeMetric is the pivot table for life metric events.
// Each row represents ONE "+1" increment event; created_at records when the event happened.
// UserID tracks which user performed the increment (nullable for backward compatibility).
type ContactLifeMetric struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID    string    `json:"contact_id" gorm:"type:text;not null;index"`
	LifeMetricID uint      `json:"life_metric_id" gorm:"not null;index"`
	UserID       string    `json:"user_id" gorm:"type:text;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ContactLifeMetric) TableName() string {
	return "contact_life_metric"
}
