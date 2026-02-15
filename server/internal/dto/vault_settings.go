package dto

import "time"

// --- Vault Settings Index + Update ---

type VaultSettingsResponse struct {
	ID                string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name              string    `json:"name" example:"Family"`
	Description       string    `json:"description" example:"Vault for family contacts"`
	DefaultTemplateID *uint     `json:"default_template_id" example:"1"`
	ShowGroupTab      bool      `json:"show_group_tab" example:"true"`
	ShowTasksTab      bool      `json:"show_tasks_tab" example:"true"`
	ShowFilesTab      bool      `json:"show_files_tab" example:"true"`
	ShowJournalTab    bool      `json:"show_journal_tab" example:"true"`
	ShowCompaniesTab  bool      `json:"show_companies_tab" example:"true"`
	ShowReportsTab    bool      `json:"show_reports_tab" example:"true"`
	ShowCalendarTab   bool      `json:"show_calendar_tab" example:"true"`
	CreatedAt         time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt         time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type UpdateVaultSettingsRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255" example:"Family"`
	Description string `json:"description" validate:"max=1024" example:"Vault for family contacts"`
}

// --- Tab Visibility ---

type UpdateTabVisibilityRequest struct {
	ShowGroupTab     *bool `json:"show_group_tab" example:"true"`
	ShowTasksTab     *bool `json:"show_tasks_tab" example:"true"`
	ShowFilesTab     *bool `json:"show_files_tab" example:"true"`
	ShowJournalTab   *bool `json:"show_journal_tab" example:"true"`
	ShowCompaniesTab *bool `json:"show_companies_tab" example:"true"`
	ShowReportsTab   *bool `json:"show_reports_tab" example:"true"`
	ShowCalendarTab  *bool `json:"show_calendar_tab" example:"true"`
}

// --- Default Template ---

type UpdateDefaultTemplateRequest struct {
	DefaultTemplateID *uint `json:"default_template_id" example:"1"`
}

// --- Vault Users Management ---

type AddVaultUserRequest struct {
	Email      string `json:"email" validate:"required,email" example:"user@example.com"`
	Permission int    `json:"permission" validate:"required" example:"100"`
}

type UpdateVaultUserPermRequest struct {
	Permission int `json:"permission" validate:"required" example:"100"`
}

type VaultUserResponse struct {
	ID         uint   `json:"id" example:"1"`
	UserID     string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email      string `json:"email" example:"user@example.com"`
	FirstName  string `json:"first_name" example:"John"`
	LastName   string `json:"last_name" example:"Doe"`
	Permission int    `json:"permission" example:"100"`
}

// --- Labels ---

type CreateLabelRequest struct {
	Name        string `json:"name" validate:"required,min=1" example:"Important"`
	Description string `json:"description" example:"Contacts marked as important"`
	BgColor     string `json:"bg_color" example:"#3B82F6"`
	TextColor   string `json:"text_color" example:"#FFFFFF"`
}

type UpdateLabelRequest struct {
	Name        string `json:"name" validate:"required,min=1" example:"Important"`
	Description string `json:"description" example:"Contacts marked as important"`
	BgColor     string `json:"bg_color" example:"#3B82F6"`
	TextColor   string `json:"text_color" example:"#FFFFFF"`
}

type LabelResponse struct {
	ID          uint      `json:"id" example:"1"`
	Name        string    `json:"name" example:"Important"`
	Slug        string    `json:"slug" example:"important"`
	Description string    `json:"description" example:"Contacts marked as important"`
	BgColor     string    `json:"bg_color" example:"#3B82F6"`
	TextColor   string    `json:"text_color" example:"#FFFFFF"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// --- Tags ---

type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=1" example:"Travel"`
}

type UpdateTagRequest struct {
	Name string `json:"name" validate:"required,min=1" example:"Travel"`
}

type TagResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Travel"`
	Slug      string    `json:"slug" example:"travel"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// --- Important Date Types ---

type CreateImportantDateTypeRequest struct {
	Label string `json:"label" validate:"required,min=1" example:"Birthdate"`
}

type UpdateImportantDateTypeRequest struct {
	Label string `json:"label" validate:"required,min=1" example:"Birthdate"`
}

type ImportantDateTypeResponse struct {
	ID           uint      `json:"id" example:"1"`
	Label        string    `json:"label" example:"Birthdate"`
	InternalType string    `json:"internal_type" example:"birthdate"`
	CanBeDeleted bool      `json:"can_be_deleted" example:"false"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt    time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// --- Mood Tracking Parameters ---

type CreateMoodTrackingParameterRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	HexColor string `json:"hex_color" validate:"required" example:"#10B981"`
	Position *int   `json:"position" example:"1"`
}

type UpdateMoodTrackingParameterRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	HexColor string `json:"hex_color" validate:"required" example:"#10B981"`
	Position *int   `json:"position" example:"1"`
}

type MoodTrackingParameterResponse struct {
	ID        uint      `json:"id" example:"1"`
	Label     string    `json:"label" example:"Birthdate"`
	HexColor  string    `json:"hex_color" example:"#10B981"`
	Position  *int      `json:"position" example:"1"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// --- Life Event Categories + Types ---

type CreateLifeEventCategoryRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type UpdateLifeEventCategoryRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type LifeEventCategoryResponse struct {
	ID           uint                    `json:"id" example:"1"`
	Label        string                  `json:"label" example:"Birthdate"`
	CanBeDeleted bool                    `json:"can_be_deleted" example:"false"`
	Position     *int                    `json:"position" example:"1"`
	Types        []LifeEventTypeResponse `json:"types,omitempty"`
	CreatedAt    time.Time               `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt    time.Time               `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type CreateLifeEventTypeRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type UpdateLifeEventTypeRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type LifeEventTypeResponse struct {
	ID           uint      `json:"id" example:"1"`
	CategoryID   uint      `json:"category_id" example:"1"`
	Label        string    `json:"label" example:"Birthdate"`
	CanBeDeleted bool      `json:"can_be_deleted" example:"false"`
	Position     *int      `json:"position" example:"1"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt    time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

// --- Quick Fact Templates ---

type CreateQuickFactTemplateRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type UpdateQuickFactTemplateRequest struct {
	Label    string `json:"label" validate:"required,min=1" example:"Birthdate"`
	Position *int   `json:"position" example:"1"`
}

type QuickFactTemplateResponse struct {
	ID        uint      `json:"id" example:"1"`
	Label     string    `json:"label" example:"Birthdate"`
	Position  int       `json:"position" example:"1"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
