package dto

// MonicaImportResponse is the response structure for Monica 4.x JSON import
type MonicaImportResponse struct {
	ImportedContacts      int      `json:"imported_contacts" example:"10"`
	ImportedNotes         int      `json:"imported_notes" example:"5"`
	ImportedCalls         int      `json:"imported_calls" example:"3"`
	ImportedTasks         int      `json:"imported_tasks" example:"2"`
	ImportedReminders     int      `json:"imported_reminders" example:"4"`
	ImportedAddresses     int      `json:"imported_addresses" example:"8"`
	ImportedLifeEvents    int      `json:"imported_life_events" example:"2"`
	ImportedRelationships int      `json:"imported_relationships" example:"6"`
	ImportedPhotos        int      `json:"imported_photos" example:"12"`
	ImportedDocuments     int      `json:"imported_documents" example:"1"`
	SkippedCount          int      `json:"skipped_count" example:"0"`
	Errors                []string `json:"errors,omitempty"`
}
