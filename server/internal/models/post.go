package models

import "time"

type PostTemplate struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID           string    `json:"account_id" gorm:"type:text;not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            int       `json:"position" gorm:"not null"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Account              Account               `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	PostTemplateSections []PostTemplateSection `json:"post_template_sections,omitempty" gorm:"foreignKey:PostTemplateID"`
}

type PostTemplateSection struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PostTemplateID      uint      `json:"post_template_id" gorm:"not null;index"`
	Label               *string   `json:"label"`
	LabelTranslationKey *string   `json:"label_translation_key"`
	Position            int       `json:"position" gorm:"not null"`
	CanBeDeleted        bool      `json:"can_be_deleted" gorm:"default:true"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	PostTemplate PostTemplate `json:"post_template,omitempty" gorm:"foreignKey:PostTemplateID"`
}

type Post struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	JournalID     uint      `json:"journal_id" gorm:"not null;index"`
	SliceOfLifeID *uint     `json:"slice_of_life_id" gorm:"index"`
	Title         *string   `json:"title"`
	ViewCount     int       `json:"view_count" gorm:"default:0"`
	Published     bool      `json:"published" gorm:"default:false"`
	WrittenAt     time.Time `json:"written_at" gorm:"not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Journal      Journal       `json:"journal,omitempty" gorm:"foreignKey:JournalID"`
	SliceOfLife  *SliceOfLife  `json:"slice_of_life,omitempty" gorm:"foreignKey:SliceOfLifeID"`
	PostSections []PostSection `json:"post_sections,omitempty" gorm:"foreignKey:PostID"`
	Contacts     []Contact     `json:"contacts,omitempty" gorm:"many2many:contact_post"`
	Tags         []Tag         `json:"tags,omitempty" gorm:"many2many:post_tag"`
	PostMetrics  []PostMetric  `json:"post_metrics,omitempty" gorm:"foreignKey:PostID"`
}

type PostSection struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PostID    uint      `json:"post_id" gorm:"not null;index"`
	Position  int       `json:"position" gorm:"not null"`
	Label     string    `json:"label" gorm:"not null"`
	Content   *string   `json:"content" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Post Post `json:"post,omitempty" gorm:"foreignKey:PostID"`
}

type PostMetric struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PostID          uint      `json:"post_id" gorm:"not null;index"`
	JournalMetricID uint      `json:"journal_metric_id" gorm:"not null;index"`
	Value           int       `json:"value" gorm:"not null"`
	Label           *string   `json:"label"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Post          Post          `json:"post,omitempty" gorm:"foreignKey:PostID"`
	JournalMetric JournalMetric `json:"journal_metric,omitempty" gorm:"foreignKey:JournalMetricID"`
}

type ContactPost struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PostID    uint      `json:"post_id" gorm:"not null;index"`
	ContactID string    `json:"contact_id" gorm:"type:text;not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ContactPost) TableName() string {
	return "contact_post"
}

type PostTag struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TagID     uint      `json:"tag_id" gorm:"not null;index"`
	PostID    uint      `json:"post_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PostTag) TableName() string {
	return "post_tag"
}
