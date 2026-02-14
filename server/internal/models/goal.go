package models

import "time"

type Goal struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID string    `json:"contact_id" gorm:"type:text;not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Active    bool      `json:"active" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Contact Contact  `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Streaks []Streak `json:"streaks,omitempty" gorm:"foreignKey:GoalID"`
}

type Streak struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	GoalID     uint      `json:"goal_id" gorm:"not null;index"`
	HappenedAt time.Time `json:"happened_at" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Goal Goal `json:"goal,omitempty" gorm:"foreignKey:GoalID"`
}
