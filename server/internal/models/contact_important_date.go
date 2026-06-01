package models

import (
	"time"

	"gorm.io/gorm"
)

type ContactImportantDateType struct {
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID string `json:"vault_id" gorm:"type:text;not null;index"`
	Label   string `json:"label" gorm:"not null"`
	// LabelTranslationKey is the i18n key the row was seeded from. Persisting it
	// lets PersonalizeService.SyncAllTranslations re-translate the row when the
	// user changes their locale — without it, seeded rows freeze at whichever
	// language was active during vault creation. Optional because user-created
	// custom types do not have a translation key.
	LabelTranslationKey *string   `json:"label_translation_key"`
	InternalType        *string   `json:"internal_type"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Vault Vault `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
}

type ContactImportantDate struct {
	ID                         uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID                  string         `json:"contact_id" gorm:"type:text;not null;index"`
	UUID                       *string        `json:"uuid" gorm:"type:text;index"`
	Vcalendar                  *string        `json:"vcalendar" gorm:"type:text"`
	DistantUUID                *string        `json:"distant_uuid" gorm:"size:256"`
	DistantEtag                *string        `json:"distant_etag" gorm:"size:256"`
	DistantURI                 *string        `json:"distant_uri" gorm:"size:2096"`
	ContactImportantDateTypeID *uint          `json:"contact_important_date_type_id" gorm:"index"`
	Label                      string         `json:"label" gorm:"not null"`
	Day                        *int           `json:"day"`
	Month                      *int           `json:"month"`
	Year                       *int           `json:"year"`
	CalendarType               string         `json:"calendar_type" gorm:"default:'gregorian'"`
	OriginalDay                *int           `json:"original_day"`
	OriginalMonth              *int           `json:"original_month"`
	OriginalYear               *int           `json:"original_year"`
	IsAgeBased                 bool           `json:"is_age_based" gorm:"default:false"`
	IsYearUnknown              bool           `json:"is_year_unknown" gorm:"default:false"`
	RemindMe                   bool           `json:"remind_me" gorm:"default:false"`
	DeletedAt                  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`

	Contact                  Contact                   `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	ContactImportantDateType *ContactImportantDateType `json:"contact_important_date_type,omitempty" gorm:"foreignKey:ContactImportantDateTypeID"`
}
