package dto

type UpdateDefaultTabRequest struct {
	DefaultActivityTab string `json:"default_activity_tab" validate:"required" example:"notes"`
}

type UpdatePostSliceRequest struct {
	SliceOfLifeID uint `json:"slice_of_life_id" example:"1"`
}

type AddressContactItem struct {
	ContactID string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName string `json:"first_name" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
	City      string `json:"city" example:"San Francisco"`
	Province  string `json:"province" example:"California"`
	Country   string `json:"country" example:"US"`
}

type MostConsultedContactItem struct {
	ContactID     string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName     string `json:"first_name" example:"John"`
	LastName      string `json:"last_name" example:"Doe"`
	NumberOfViews int    `json:"number_of_views" example:"10"`
}

type VaultReminderItem struct {
	ReminderResponse
	ContactFirstName string `json:"contact_first_name" example:"John"`
	ContactLastName  string `json:"contact_last_name" example:"Doe"`
}

type ReportIndexItem struct {
	Key  string `json:"key" example:"addresses"`
	Name string `json:"name" example:"Address Report"`
}
