package models

import "time"

type ActivityCategory struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID             string    `json:"vault_id" gorm:"type:text;not null;index"`
	Position            *int      `json:"position"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:false"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Vault         Vault          `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	ActivityTypes []ActivityType `json:"activity_types,omitempty" gorm:"foreignKey:ActivityCategoryID"`
}

type ActivityType struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ActivityCategoryID  uint      `json:"activity_category_id" gorm:"not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:false"`
	Position            *int      `json:"position"`
	SystemKind          *string   `json:"system_kind" gorm:"size:64;index"`
	Icon                *string   `json:"icon" gorm:"size:64"`
	Color               *string   `json:"color" gorm:"size:32"`
	CountsAsInteraction bool      `json:"counts_as_interaction" gorm:"not null;default:false"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	ActivityCategory ActivityCategory `json:"activity_category,omitempty" gorm:"foreignKey:ActivityCategoryID"`
	Activities       []Activity       `json:"activities,omitempty" gorm:"foreignKey:ActivityTypeID"`
}

type Activity struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID         string     `json:"vault_id" gorm:"type:text;not null;index;uniqueIndex:idx_activity_source"`
	ParentID        *uint      `json:"parent_id" gorm:"index"`
	ActivityTypeID  *uint      `json:"activity_type_id" gorm:"index"`
	SubjectUserID   *string    `json:"subject_user_id" gorm:"type:text;index"`
	SubjectUserName *string    `json:"subject_user_name" gorm:"type:text"`
	EmotionID       *uint      `json:"emotion_id" gorm:"index"`
	StartDate       *time.Time `json:"start_date" gorm:"type:date;index"`
	StartPrecision  string     `json:"start_precision" gorm:"size:16;not null;default:'day'"`
	EndDate         *time.Time `json:"end_date" gorm:"type:date;index"`
	EndPrecision    string     `json:"end_precision" gorm:"size:16"`
	EndStatus       string     `json:"end_status" gorm:"size:16;not null;default:'none'"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear preserve the
	// user's input when they record an activity in a non-Gregorian calendar
	// (e.g. lunar). StartDate stores the Gregorian projection used for queries
	// and sorting;
	// the Original* triple lets the UI render the lunar string and lets edits
	// re-display the user's original input instead of a back-converted value
	// that may have drifted by a day.
	// Defaults to "gregorian" so legacy rows pre-dating the column read as
	// gregorian without a backfill needing to touch them on every boot.
	CalendarType      string    `json:"calendar_type" gorm:"default:'gregorian'"`
	OriginalDay       *int      `json:"original_day"`
	OriginalMonth     *int      `json:"original_month"`
	OriginalYear      *int      `json:"original_year"`
	Title             string    `json:"title" gorm:"not null"`
	Description       *string   `json:"description" gorm:"type:text"`
	DescriptionFormat string    `json:"description_format" gorm:"size:16;not null;default:'plain'"`
	Costs             *int      `json:"costs"`
	CurrencyID        *uint     `json:"currency_id" gorm:"index"`
	PaidByContactID   *string   `json:"paid_by_contact_id" gorm:"type:text;index"`
	DurationInMinutes *int      `json:"duration_in_minutes"`
	Distance          *int      `json:"distance"`
	DistanceUnit      *string   `json:"distance_unit" gorm:"size:2"`
	FromPlace         *string   `json:"from_place"`
	ToPlace           *string   `json:"to_place"`
	Place             *string   `json:"place"`
	SourceType        *string   `json:"source_type" gorm:"size:64;uniqueIndex:idx_activity_source"`
	SourceUUID        *string   `json:"source_uuid" gorm:"size:191;uniqueIndex:idx_activity_source"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Vault        Vault         `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Parent       *Activity     `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Milestones   []Activity    `json:"milestones,omitempty" gorm:"foreignKey:ParentID"`
	ActivityType *ActivityType `json:"activity_type,omitempty" gorm:"foreignKey:ActivityTypeID"`
	Currency     *Currency     `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
	Emotion      *Emotion      `json:"emotion,omitempty" gorm:"foreignKey:EmotionID"`
	PaidBy       *Contact      `json:"paid_by,omitempty" gorm:"foreignKey:PaidByContactID"`
	Participants []Contact     `json:"participants,omitempty" gorm:"many2many:activity_participants"`
}

type ActivityParticipant struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID  string    `json:"contact_id" gorm:"type:text;not null;uniqueIndex:idx_activity_participant_unique"`
	ActivityID uint      `json:"activity_id" gorm:"not null;uniqueIndex:idx_activity_participant_unique;index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ActivityParticipant) TableName() string {
	return "activity_participants"
}
