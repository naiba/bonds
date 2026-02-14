package models

import "time"

type MoodTrackingParameter struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID             string    `json:"vault_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	HexColor            string    `json:"hex_color" gorm:"not null"`
	Position            *int      `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Vault              Vault               `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	MoodTrackingEvents []MoodTrackingEvent `json:"mood_tracking_events,omitempty" gorm:"foreignKey:MoodTrackingParameterID"`
}

type MoodTrackingEvent struct {
	ID                      uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID               string    `json:"contact_id" gorm:"type:text;not null;index"`
	MoodTrackingParameterID uint      `json:"mood_tracking_parameter_id" gorm:"not null;index"`
	RatedAt                 time.Time `json:"rated_at" gorm:"not null"`
	Note                    *string   `json:"note" gorm:"type:text"`
	NumberOfHoursSlept      *int      `json:"number_of_hours_slept"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	Contact               Contact               `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	MoodTrackingParameter MoodTrackingParameter `json:"mood_tracking_parameter,omitempty" gorm:"foreignKey:MoodTrackingParameterID"`
}
