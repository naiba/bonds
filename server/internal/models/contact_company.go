package models

import "time"

// ContactCompany is the many-to-many join table between Contact and Company.
// A contact can work at multiple companies, each with a different job position.
type ContactCompany struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ContactID   string    `json:"contact_id" gorm:"type:text;not null;index"`
	CompanyID   uint      `json:"company_id" gorm:"not null;index"`
	JobPosition *string   `json:"job_position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Contact Contact `json:"contact,omitempty" gorm:"foreignKey:ContactID"`
	Company Company `json:"company,omitempty" gorm:"foreignKey:CompanyID"`
}
