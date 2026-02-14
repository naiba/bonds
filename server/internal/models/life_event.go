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
	ID                uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TimelineEventID   uint      `json:"timeline_event_id" gorm:"not null;index"`
	LifeEventTypeID   uint      `json:"life_event_type_id" gorm:"not null;index"`
	EmotionID         *uint     `json:"emotion_id" gorm:"index"`
	HappenedAt        time.Time `json:"happened_at" gorm:"type:date;not null"`
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
	ContactID       string    `json:"contact_id" gorm:"type:text;not null;index"`
	TimelineEventID uint      `json:"timeline_event_id" gorm:"not null;index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (TimelineEventParticipant) TableName() string {
	return "timeline_event_participants"
}

type LifeEventParticipant struct {
	ContactID   string    `json:"contact_id" gorm:"type:text;not null;index"`
	LifeEventID uint      `json:"life_event_id" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LifeEventParticipant) TableName() string {
	return "life_event_participants"
}
