package dto

import "time"

// --- Vault Settings Index + Update ---

type VaultSettingsResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	DefaultTemplateID *uint     `json:"default_template_id"`
	ShowGroupTab      bool      `json:"show_group_tab"`
	ShowTasksTab      bool      `json:"show_tasks_tab"`
	ShowFilesTab      bool      `json:"show_files_tab"`
	ShowJournalTab    bool      `json:"show_journal_tab"`
	ShowCompaniesTab  bool      `json:"show_companies_tab"`
	ShowReportsTab    bool      `json:"show_reports_tab"`
	ShowCalendarTab   bool      `json:"show_calendar_tab"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UpdateVaultSettingsRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1024"`
}

// --- Tab Visibility ---

type UpdateTabVisibilityRequest struct {
	ShowGroupTab     *bool `json:"show_group_tab"`
	ShowTasksTab     *bool `json:"show_tasks_tab"`
	ShowFilesTab     *bool `json:"show_files_tab"`
	ShowJournalTab   *bool `json:"show_journal_tab"`
	ShowCompaniesTab *bool `json:"show_companies_tab"`
	ShowReportsTab   *bool `json:"show_reports_tab"`
	ShowCalendarTab  *bool `json:"show_calendar_tab"`
}

// --- Default Template ---

type UpdateDefaultTemplateRequest struct {
	DefaultTemplateID *uint `json:"default_template_id"`
}

// --- Vault Users Management ---

type AddVaultUserRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Permission int    `json:"permission" validate:"required"`
}

type UpdateVaultUserPermRequest struct {
	Permission int `json:"permission" validate:"required"`
}

type VaultUserResponse struct {
	ID         uint   `json:"id"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Permission int    `json:"permission"`
}

// --- Labels ---

type CreateLabelRequest struct {
	Name        string `json:"name" validate:"required,min=1"`
	Description string `json:"description"`
	BgColor     string `json:"bg_color"`
	TextColor   string `json:"text_color"`
}

type UpdateLabelRequest struct {
	Name        string `json:"name" validate:"required,min=1"`
	Description string `json:"description"`
	BgColor     string `json:"bg_color"`
	TextColor   string `json:"text_color"`
}

type LabelResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	BgColor     string    `json:"bg_color"`
	TextColor   string    `json:"text_color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// --- Tags ---

type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

type UpdateTagRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

type TagResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Important Date Types ---

type CreateImportantDateTypeRequest struct {
	Label string `json:"label" validate:"required,min=1"`
}

type UpdateImportantDateTypeRequest struct {
	Label string `json:"label" validate:"required,min=1"`
}

type ImportantDateTypeResponse struct {
	ID           uint      `json:"id"`
	Label        string    `json:"label"`
	InternalType string    `json:"internal_type"`
	CanBeDeleted bool      `json:"can_be_deleted"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// --- Mood Tracking Parameters ---

type CreateMoodTrackingParameterRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	HexColor string `json:"hex_color" validate:"required"`
	Position *int   `json:"position"`
}

type UpdateMoodTrackingParameterRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	HexColor string `json:"hex_color" validate:"required"`
	Position *int   `json:"position"`
}

type MoodTrackingParameterResponse struct {
	ID        uint      `json:"id"`
	Label     string    `json:"label"`
	HexColor  string    `json:"hex_color"`
	Position  *int      `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Life Event Categories + Types ---

type CreateLifeEventCategoryRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type UpdateLifeEventCategoryRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type LifeEventCategoryResponse struct {
	ID           uint                    `json:"id"`
	Label        string                  `json:"label"`
	CanBeDeleted bool                    `json:"can_be_deleted"`
	Position     *int                    `json:"position"`
	Types        []LifeEventTypeResponse `json:"types,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

type CreateLifeEventTypeRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type UpdateLifeEventTypeRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type LifeEventTypeResponse struct {
	ID           uint      `json:"id"`
	CategoryID   uint      `json:"category_id"`
	Label        string    `json:"label"`
	CanBeDeleted bool      `json:"can_be_deleted"`
	Position     *int      `json:"position"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// --- Quick Fact Templates ---

type CreateQuickFactTemplateRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type UpdateQuickFactTemplateRequest struct {
	Label    string `json:"label" validate:"required,min=1"`
	Position *int   `json:"position"`
}

type QuickFactTemplateResponse struct {
	ID        uint      `json:"id"`
	Label     string    `json:"label"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
