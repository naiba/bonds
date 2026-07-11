package dto

import "time"

type CreateVaultRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255" example:"Family"`
	Description string `json:"description" validate:"max=1024" example:"Vault for family contacts"`
}

type UpdateVaultRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255" example:"Family"`
	Description string `json:"description" validate:"max=1024" example:"Vault for family contacts"`
}

type VaultResponse struct {
	ID                 string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID          string    `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name               string    `json:"name" example:"Family"`
	Description        string    `json:"description" example:"Vault for family contacts"`
	NameOrder          *string   `json:"name_order" example:"%first_name% %last_name%"`
	EffectiveNameOrder string    `json:"effective_name_order" example:"%first_name% %last_name%"`
	DefaultActivityTab string    `json:"default_activity_tab" example:"activity"`
	UserContactID      string    `json:"user_contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShowGroupTab       bool      `json:"show_group_tab" example:"true"`
	ShowTasksTab       bool      `json:"show_tasks_tab" example:"true"`
	ShowFilesTab       bool      `json:"show_files_tab" example:"true"`
	ShowJournalTab     bool      `json:"show_journal_tab" example:"true"`
	ShowCompaniesTab   bool      `json:"show_companies_tab" example:"true"`
	ShowReportsTab     bool      `json:"show_reports_tab" example:"true"`
	ShowCalendarTab    bool      `json:"show_calendar_tab" example:"true"`
	CreatedAt          time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt          time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
