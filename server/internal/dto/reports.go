package dto

type AddressReportItem struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Count    int    `json:"count"`
}

type ImportantDateReportItem struct {
	ContactID string `json:"contact_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Label     string `json:"label"`
	Day       *int   `json:"day"`
	Month     *int   `json:"month"`
	Year      *int   `json:"year"`
}

type MoodReportItem struct {
	ParameterLabel string `json:"parameter_label"`
	HexColor       string `json:"hex_color"`
	Count          int    `json:"count"`
}
