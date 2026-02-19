package dto

type VCardImportResponse struct {
	ImportedCount int               `json:"imported_count" example:"10"`
	SkippedCount  int               `json:"skipped_count" example:"0"`
	Errors        []string          `json:"errors,omitempty"`
	Contacts      []ContactResponse `json:"contacts"`
}
