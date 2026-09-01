package models

import "time"

const (
	ContentOwnerPostSection = "post_section"
	ContentOwnerActivity    = "activity"
	ContentOwnerNote        = "note"
)

// ContentFileReference keeps uploaded files alive while Markdown content uses
// bonds-file:<id>. The explicit primary key is required for reliable SQLite
// pivot-table migrations.
type ContentFileReference struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	VaultID   string    `json:"vault_id" gorm:"type:text;not null;index;uniqueIndex:idx_content_file_reference"`
	FileID    uint      `json:"file_id" gorm:"not null;index;uniqueIndex:idx_content_file_reference"`
	OwnerType string    `json:"owner_type" gorm:"size:32;not null;index;uniqueIndex:idx_content_file_reference"`
	OwnerID   uint      `json:"owner_id" gorm:"not null;index;uniqueIndex:idx_content_file_reference"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	File File `json:"-" gorm:"foreignKey:FileID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
