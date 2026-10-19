package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrDefaultTemplateOutsideVault = errors.New("default contact template must belong to the vault")

// CardDAVFilter stores only the selected presence conditions, never contact data.
type CardDAVFilter struct {
	Match  string   `json:"match"`
	Fields []string `json:"fields"`
}

type Vault struct {
	CardDAVFilter     *CardDAVFilter `json:"carddav_filter" gorm:"serializer:json;type:text"`
	ID                string         `json:"id" gorm:"primaryKey;type:text"`
	AccountID         string         `json:"account_id" gorm:"type:text;not null;index"`
	Type              string         `json:"type" gorm:"not null"`
	Name              string         `json:"name" gorm:"not null"`
	Description       *string        `json:"description"`
	DefaultTemplateID *uint          `json:"default_template_id" gorm:"index"`
	ShowGroupTab      bool           `json:"show_group_tab" gorm:"default:true"`
	ShowTasksTab      bool           `json:"show_tasks_tab" gorm:"default:true"`
	ShowFilesTab      bool           `json:"show_files_tab" gorm:"default:true"`
	ShowJournalTab    bool           `json:"show_journal_tab" gorm:"default:true"`
	ShowCompaniesTab  bool           `json:"show_companies_tab" gorm:"default:true"`
	ShowReportsTab    bool           `json:"show_reports_tab" gorm:"default:true"`
	ShowCalendarTab   bool           `json:"show_calendar_tab" gorm:"default:true"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	Account                   Account                    `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	Contacts                  []Contact                  `json:"contacts,omitempty" gorm:"foreignKey:VaultID"`
	Labels                    []Label                    `json:"labels,omitempty" gorm:"foreignKey:VaultID"`
	Users                     []User                     `json:"users,omitempty" gorm:"many2many:user_vault"`
	ContactImportantDateTypes []ContactImportantDateType `json:"contact_important_date_types,omitempty" gorm:"foreignKey:VaultID"`
	Companies                 []Company                  `json:"companies,omitempty" gorm:"foreignKey:VaultID"`
	Groups                    []Group                    `json:"groups,omitempty" gorm:"foreignKey:VaultID"`
	Journals                  []Journal                  `json:"journals,omitempty" gorm:"foreignKey:VaultID"`
	Tags                      []Tag                      `json:"tags,omitempty" gorm:"foreignKey:VaultID"`
	Loans                     []Loan                     `json:"loans,omitempty" gorm:"foreignKey:VaultID"`
	Files                     []File                     `json:"files,omitempty" gorm:"foreignKey:VaultID"`
	MoodTrackingParameters    []MoodTrackingParameter    `json:"mood_tracking_parameters,omitempty" gorm:"foreignKey:VaultID"`
	ActivityCategories        []ActivityCategory         `json:"activity_categories,omitempty" gorm:"foreignKey:VaultID"`
	Addresses                 []Address                  `json:"addresses,omitempty" gorm:"foreignKey:VaultID"`
	QuickFactsTemplateEntries []VaultQuickFactsTemplate  `json:"quick_facts_template_entries,omitempty" gorm:"foreignKey:VaultID"`
	LifeMetrics               []LifeMetric               `json:"life_metrics,omitempty" gorm:"foreignKey:VaultID"`
}

func (v *Vault) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

func (v *Vault) BeforeSave(tx *gorm.DB) error {
	if v.DefaultTemplateID == nil || v.ID == "" || !tx.Migrator().HasTable(&VaultContactTemplate{}) {
		return nil
	}
	var count int64
	if err := tx.Model(&VaultContactTemplate{}).
		Where("id = ? AND vault_id = ?", *v.DefaultTemplateID, v.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%w: template %d, vault %s", ErrDefaultTemplateOutsideVault, *v.DefaultTemplateID, v.ID)
	}
	return nil
}
