package models

import "time"

type SliceOfLife struct {
	ID               uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	JournalID        uint      `json:"journal_id" gorm:"not null;index"`
	FileCoverImageID *uint     `json:"file_cover_image_id" gorm:"index"`
	Name             string    `json:"name" gorm:"not null"`
	Description      *string   `json:"description"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Journal Journal `json:"journal,omitempty" gorm:"foreignKey:JournalID"`
	File    *File   `json:"file,omitempty" gorm:"foreignKey:FileCoverImageID"`
	Posts   []Post  `json:"posts,omitempty" gorm:"foreignKey:SliceOfLifeID"`
}

func (SliceOfLife) TableName() string {
	return "slice_of_lives"
}
