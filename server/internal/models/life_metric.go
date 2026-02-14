package models

import "time"

type LifeMetric struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	Label     string    `json:"label" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Vault    Vault     `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Contacts []Contact `json:"contacts,omitempty" gorm:"many2many:contact_life_metric"`
}

type ContactLifeMetric struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID    string    `json:"contact_id" gorm:"type:text;not null;index"`
	LifeMetricID uint      `json:"life_metric_id" gorm:"not null;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ContactLifeMetric) TableName() string {
	return "contact_life_metric"
}
