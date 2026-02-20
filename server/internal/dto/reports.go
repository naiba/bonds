package dto

type AddressReportItem struct {
	Country  string `json:"country" example:"US"`
	Province string `json:"province" example:"California"`
	City     string `json:"city" example:"San Francisco"`
	Count    int    `json:"count" example:"10"`
}

type ImportantDateReportItem struct {
	ContactID     string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName     string `json:"first_name" example:"John"`
	LastName      string `json:"last_name" example:"Doe"`
	Label         string `json:"label" example:"Birthday"`
	Day           *int   `json:"day" example:"15"`
	Month         *int   `json:"month" example:"6"`
	Year          *int   `json:"year" example:"1990"`
	CalendarType  string `json:"calendar_type" example:"gregorian"`
	OriginalDay   *int   `json:"original_day" example:"15"`
	OriginalMonth *int   `json:"original_month" example:"6"`
	OriginalYear  *int   `json:"original_year" example:"1990"`
}

type MoodReportItem struct {
	ParameterLabel string `json:"parameter_label" example:"Awesome"`
	HexColor       string `json:"hex_color" example:"#3B82F6"`
	Count          int    `json:"count" example:"10"`
}

type ReportOverviewResponse struct {
	TotalContacts       int `json:"total_contacts" example:"42"`
	TotalAddresses      int `json:"total_addresses" example:"10"`
	TotalImportantDates int `json:"total_important_dates" example:"15"`
	TotalMoodEntries    int `json:"total_mood_entries" example:"8"`
}
