package dto

type VCardImportResponse struct {
	ImportedCount int               `json:"imported_count" example:"10"`
	Contacts      []ContactResponse `json:"contacts"`
}
