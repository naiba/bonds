package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Contact struct {
	ID             string         `json:"id" gorm:"primaryKey;type:text"`
	VaultID        string         `json:"vault_id" gorm:"type:text;not null;index:idx_contact_vault"`
	GenderID       *uint          `json:"gender_id" gorm:"index"`
	PronounID      *uint          `json:"pronoun_id" gorm:"index"`
	TemplateID     *uint          `json:"template_id" gorm:"index"`
	CompanyID      *uint          `json:"company_id" gorm:"index"`
	FileID         *uint          `json:"file_id" gorm:"index"`
	ReligionID     *uint          `json:"religion_id" gorm:"index"`
	FirstName      *string        `json:"first_name"`
	MiddleName     *string        `json:"middle_name"`
	LastName       *string        `json:"last_name"`
	Nickname       *string        `json:"nickname"`
	MaidenName     *string        `json:"maiden_name"`
	Suffix         *string        `json:"suffix"`
	Prefix         *string        `json:"prefix"`
	JobPosition    *string        `json:"job_position"`
	CanBeDeleted   bool           `json:"can_be_deleted" gorm:"default:true"`
	ShowQuickFacts bool           `json:"show_quick_facts" gorm:"default:false"`
	Listed         bool           `json:"listed" gorm:"default:true"`
	Vcard          *string        `json:"vcard" gorm:"type:mediumtext"`
	DistantUUID    *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag    *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI     *string        `json:"distant_uri" gorm:"size:2096"`
	LastUpdatedAt  *time.Time     `json:"last_updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	Vault               Vault                  `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	Gender              *Gender                `json:"gender,omitempty" gorm:"foreignKey:GenderID"`
	Pronoun             *Pronoun               `json:"pronoun,omitempty" gorm:"foreignKey:PronounID"`
	Template            *Template              `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Company             *Company               `json:"company,omitempty" gorm:"foreignKey:CompanyID"`
	File                *File                  `json:"file,omitempty" gorm:"foreignKey:FileID"`
	Religion            *Religion              `json:"religion,omitempty" gorm:"foreignKey:ReligionID"`
	Labels              []Label                `json:"labels,omitempty" gorm:"many2many:contact_label"`
	ContactInformations []ContactInformation   `json:"contact_informations,omitempty" gorm:"foreignKey:ContactID"`
	Notes               []Note                 `json:"notes,omitempty" gorm:"foreignKey:ContactID"`
	ImportantDates      []ContactImportantDate `json:"important_dates,omitempty" gorm:"foreignKey:ContactID"`
	Reminders           []ContactReminder      `json:"reminders,omitempty" gorm:"foreignKey:ContactID"`
	Tasks               []ContactTask          `json:"tasks,omitempty" gorm:"foreignKey:ContactID"`
	Calls               []Call                 `json:"calls,omitempty" gorm:"foreignKey:ContactID"`
	Pets                []Pet                  `json:"pets,omitempty" gorm:"foreignKey:ContactID"`
	Goals               []Goal                 `json:"goals,omitempty" gorm:"foreignKey:ContactID"`
	Groups              []Group                `json:"groups,omitempty" gorm:"many2many:contact_group"`
	Posts               []Post                 `json:"posts,omitempty" gorm:"many2many:contact_post"`
	LifeEvents          []LifeEvent            `json:"life_events,omitempty" gorm:"many2many:life_event_participants"`
	TimelineEvents      []TimelineEvent        `json:"timeline_events,omitempty" gorm:"many2many:timeline_event_participants"`
	MoodTrackingEvents  []MoodTrackingEvent    `json:"mood_tracking_events,omitempty" gorm:"foreignKey:ContactID"`
	Addresses           []Address              `json:"addresses,omitempty" gorm:"many2many:contact_address"`
	QuickFacts          []QuickFact            `json:"quick_facts,omitempty" gorm:"foreignKey:ContactID"`
	LifeMetrics         []LifeMetric           `json:"life_metrics,omitempty" gorm:"many2many:contact_life_metric"`
}

func (c *Contact) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
