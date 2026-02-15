package dto

type UpdateDefaultTabRequest struct {
	DefaultActivityTab string `json:"default_activity_tab" validate:"required"`
}

type UpdatePostSliceRequest struct {
	SliceOfLifeID uint `json:"slice_of_life_id"`
}

type AddressContactItem struct {
	ContactID string `json:"contact_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
}

type MostConsultedContactItem struct {
	ContactID     string `json:"contact_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	NumberOfViews int    `json:"number_of_views"`
}

type VaultReminderItem struct {
	ReminderResponse
	ContactFirstName string `json:"contact_first_name"`
	ContactLastName  string `json:"contact_last_name"`
}

type ReportIndexItem struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}
