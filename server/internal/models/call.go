package models

import "time"

type CallReasonType struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID           string    `json:"account_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Account     Account      `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	CallReasons []CallReason `json:"call_reasons,omitempty" gorm:"foreignKey:CallReasonTypeID"`
}

type CallReason struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CallReasonTypeID    uint      `json:"call_reason_type_id" gorm:"not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	CallReasonType CallReasonType `json:"call_reason_type,omitempty" gorm:"foreignKey:CallReasonTypeID"`
}

type Call struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID    string    `json:"contact_id" gorm:"type:text;not null;index"`
	CallReasonID *uint     `json:"call_reason_id" gorm:"index"`
	AuthorID     *string   `json:"author_id" gorm:"type:text;index"`
	EmotionID    *uint     `json:"emotion_id" gorm:"index"`
	AuthorName   string    `json:"author_name" gorm:"not null"`
	CalledAt     time.Time `json:"called_at" gorm:"not null"`
	Duration     *int      `json:"duration"`
	Type         string    `json:"type" gorm:"not null"`
	Description  *string   `json:"description" gorm:"type:text"`
	Answered     bool      `json:"answered" gorm:"default:true"`
	WhoInitiated string    `json:"who_initiated" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Contact    Contact     `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Author     *User       `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	CallReason *CallReason `json:"call_reason,omitempty" gorm:"foreignKey:CallReasonID"`
	Emotion    *Emotion    `json:"emotion,omitempty" gorm:"foreignKey:EmotionID"`
}
