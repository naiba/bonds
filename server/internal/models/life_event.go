package models

import "time"

type LifeEventCategory struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID             string    `json:"vault_id" gorm:"type:text;not null;index"`
	Position            *int      `json:"position"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:false"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Vault          Vault           `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	LifeEventTypes []LifeEventType `json:"life_event_types,omitempty" gorm:"foreignKey:LifeEventCategoryID"`
}

type LifeEventType struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	LifeEventCategoryID uint      `json:"life_event_category_id" gorm:"not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:false"`
	Position            *int      `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	LifeEventCategory LifeEventCategory `json:"life_event_category,omitempty" gorm:"foreignKey:LifeEventCategoryID"`
	LifeEvents        []LifeEvent       `json:"life_events,omitempty" gorm:"foreignKey:LifeEventTypeID"`
}

type TimelineEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index"`
	StartedAt time.Time `json:"started_at" gorm:"type:date;not null"`
	Label     *string   `json:"label"`
	Collapsed bool      `json:"collapsed" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Vault        Vault       `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	LifeEvents   []LifeEvent `json:"life_events,omitempty" gorm:"foreignKey:TimelineEventID"`
	Participants []Contact   `json:"participants,omitempty" gorm:"many2many:timeline_event_participants"`
}

type LifeEvent struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TimelineEventID uint      `json:"timeline_event_id" gorm:"not null;index"`
	LifeEventTypeID uint      `json:"life_event_type_id" gorm:"not null;index"`
	EmotionID       *uint     `json:"emotion_id" gorm:"index"`
	HappenedAt      time.Time `json:"happened_at" gorm:"type:date;not null"`
	// CalendarType / OriginalDay / OriginalMonth / OriginalYear preserve the
	// user's input when they record a life event in a non-Gregorian calendar
	// (e.g. lunar). HappenedAt always stores the Gregorian projection so
	// existing queries, sorting and timeline rendering keep working unchanged;
	// the Original* triple lets the UI render the lunar string and lets edits
	// re-display the user's original input instead of a back-converted value
	// that may have drifted by a day.
	// Defaults to "gregorian" so legacy rows pre-dating the column read as
	// gregorian without a backfill needing to touch them on every boot.
	CalendarType      string    `json:"calendar_type" gorm:"default:'gregorian'"`
	OriginalDay       *int      `json:"original_day"`
	OriginalMonth     *int      `json:"original_month"`
	OriginalYear      *int      `json:"original_year"`
	Collapsed         bool      `json:"collapsed" gorm:"default:false"`
	Summary           *string   `json:"summary"`
	Description       *string   `json:"description" gorm:"type:text"`
	Costs             *int      `json:"costs"`
	CurrencyID        *uint     `json:"currency_id" gorm:"index"`
	PaidByContactID   *string   `json:"paid_by_contact_id" gorm:"type:text;index"`
	DurationInMinutes *int      `json:"duration_in_minutes"`
	Distance          *int      `json:"distance"`
	DistanceUnit      *string   `json:"distance_unit" gorm:"size:2"`
	FromPlace         *string   `json:"from_place"`
	ToPlace           *string   `json:"to_place"`
	Place             *string   `json:"place"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	TimelineEvent TimelineEvent `json:"timeline_event,omitempty" gorm:"foreignKey:TimelineEventID"`
	LifeEventType LifeEventType `json:"life_event_type,omitempty" gorm:"foreignKey:LifeEventTypeID"`
	Currency      *Currency     `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
	Emotion       *Emotion      `json:"emotion,omitempty" gorm:"foreignKey:EmotionID"`
	PaidBy        *Contact      `json:"paid_by,omitempty" gorm:"foreignKey:PaidByContactID"`
	Participants  []Contact     `json:"participants,omitempty" gorm:"many2many:life_event_participants"`
}

type TimelineEventParticipant struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID       string    `json:"contact_id" gorm:"type:text;not null;uniqueIndex:idx_timeline_event_participant_unique"`
	TimelineEventID uint      `json:"timeline_event_id" gorm:"not null;uniqueIndex:idx_timeline_event_participant_unique"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (TimelineEventParticipant) TableName() string {
	return "timeline_event_participants"
}

type LifeEventParticipant struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID   string    `json:"contact_id" gorm:"type:text;not null;uniqueIndex:idx_life_event_participant_unique"`
	LifeEventID uint      `json:"life_event_id" gorm:"not null;uniqueIndex:idx_life_event_participant_unique"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LifeEventParticipant) TableName() string {
	return "life_event_participants"
}
