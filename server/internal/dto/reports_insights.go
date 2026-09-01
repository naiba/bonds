package dto

import "time"

// --- Demographics -----------------------------------------------------------

// DemographicBucket is one value of one dimension, with how many contacts carry
// it. Value is the stable row id (or a synthetic key for derived dimensions like
// age) so a rename never changes what a bucket means; Label is what to display.
type DemographicBucket struct {
	Value string `json:"value" example:"2"`
	Label string `json:"label" example:"Female"`
	Count int    `json:"count" example:"48"`
}

// DemographicDimension is one groupable attribute. Unset is reported alongside
// the buckets rather than folded into them: for most vaults the interesting
// number is how much of the address book is unfilled, and a chart that silently
// drops the blanks makes 3% coverage look like 100%.
type DemographicDimension struct {
	Key     string              `json:"key" example:"gender"`
	Known   int                 `json:"known" example:"120"`
	Unset   int                 `json:"unset" example:"22"`
	Buckets []DemographicBucket `json:"buckets"`
}

type DemographicsReportResponse struct {
	TotalContacts int                    `json:"total_contacts" example:"142"`
	Dimensions    []DemographicDimension `json:"dimensions"`
}

// --- Map --------------------------------------------------------------------

type MapContactItem struct {
	ContactID   string `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContactName string `json:"contact_name" example:"John Doe"`
}

// MapPoint is one geocoded address and everyone who lives at it.
type MapPoint struct {
	AddressID uint             `json:"address_id" example:"12"`
	Latitude  float64          `json:"latitude" example:"51.5072"`
	Longitude float64          `json:"longitude" example:"-0.1276"`
	City      string           `json:"city" example:"London"`
	Province  string           `json:"province" example:"Greater London"`
	Country   string           `json:"country" example:"United Kingdom"`
	Contacts  []MapContactItem `json:"contacts"`
}

// MapCountryItem lets the map draw a choropleth even when nothing is geocoded,
// which is the normal state of a vault imported from a phone address book.
//
// ContactIDs carries who is behind ContactCount, because the client merges
// spellings of one country ("USA", "United States") onto a single map feature
// — and a person with an address under each spelling must still count once
// after the merge, which bare counts cannot express. The IDs reveal nothing
// the report's points do not already carry.
type MapCountryItem struct {
	Country      string   `json:"country" example:"United Kingdom"`
	AddressCount int      `json:"address_count" example:"227"`
	ContactCount int      `json:"contact_count" example:"180"`
	ContactIDs   []string `json:"contact_ids"`
	Geocoded     int      `json:"geocoded" example:"40"`
}

type MapReportResponse struct {
	TotalAddresses int                  `json:"total_addresses" example:"379"`
	GeocodedCount  int                  `json:"geocoded_count" example:"40"`
	Points         []MapPoint           `json:"points"`
	Countries      []MapCountryItem     `json:"countries"`
	Attribution    []AddressAttribution `json:"attribution"`
}

// --- Interactions -----------------------------------------------------------

// InteractionBucket is one calendar month, as YYYY-MM. Bucketing happens in Go
// rather than SQL because SQLite and PostgreSQL disagree on date functions and
// the server supports both.
type InteractionBucket struct {
	Period string `json:"period" example:"2026-08"`
	Count  int    `json:"count" example:"37"`
}

// InteractionChannel is one activity type that counts as an interaction —
// "WhatsApp", "Phone call", "In-person meeting". Count and Months both cover
// the reported window only.
type InteractionChannel struct {
	ActivityTypeID uint                `json:"activity_type_id" example:"195"`
	Label          string              `json:"label" example:"Phone call"`
	Icon           string              `json:"icon" example:"phone"`
	Color          string              `json:"color" example:"#1677ff"`
	Count          int                 `json:"count" example:"212"`
	Months         []InteractionBucket `json:"months"`
}

// InteractionContactItem is one person's cadence.
//
// Unlike the report's headline figures, every field here is measured over ALL
// recorded history rather than the requested window. A rhythm confined to the
// window could not describe a friendship whose normal gap is longer than it,
// and "last spoken to" would be a lie if it could only look back two years.
//
// MedianGapDays is the median number of days between consecutive interactions,
// which is a far better description of a relationship's rhythm than a mean —
// one three-year gap should not turn a weekly friend into a yearly one.
type InteractionContactItem struct {
	ContactID     string     `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContactName   string     `json:"contact_name" example:"John Doe"`
	Count         int        `json:"count" example:"64"`
	FirstAt       *time.Time `json:"first_at" example:"2021-03-04T00:00:00Z"`
	LastAt        *time.Time `json:"last_at" example:"2026-08-19T00:00:00Z"`
	DaysSinceLast *int       `json:"days_since_last" example:"6"`
	MedianGapDays *int       `json:"median_gap_days" example:"9"`
}

// InteractionsReportResponse describes one window of the activity log.
//
// The window contract: TotalActivities, TotalInteractions, Months and every
// Channel count describe the requested window only, so changing `months`
// changes all of them together. MostFrequent and GoneQuiet are the deliberate
// exception and are measured over all history — see InteractionContactItem.
type InteractionsReportResponse struct {
	// TotalActivities counts every activity in the window regardless of type,
	// so a vault whose types are all unflagged can be told why its report is
	// empty instead of being shown a blank chart.
	TotalActivities int `json:"total_activities" example:"7643"`
	// TotalInteractions counts only activities whose type counts as an
	// interaction, within the window.
	TotalInteractions int `json:"total_interactions" example:"1204"`
	// InteractionTypesConfigured reports whether any activity type in this
	// vault is flagged to count as an interaction at all. It separates the two
	// reasons a report can be empty: nothing is flagged (fix the type
	// settings), or types are flagged but nothing happened in the window.
	InteractionTypesConfigured bool `json:"interaction_types_configured" example:"true"`
	// ContactCount is how many people appear in the per-contact lists, which
	// are all-history; it is not a count for the window. Contacts that are
	// archived, deleted, or otherwise no longer listed are excluded, the same
	// as they are from the lists themselves.
	ContactCount int                      `json:"contact_count" example:"88"`
	Months       []InteractionBucket      `json:"months"`
	Channels     []InteractionChannel     `json:"channels"`
	MostFrequent []InteractionContactItem `json:"most_frequent"`
	GoneQuiet    []InteractionContactItem `json:"gone_quiet"`
}
