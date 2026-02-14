package dto

type VCardImportResponse struct {
	ImportedCount int               `json:"imported_count"`
	Contacts      []ContactResponse `json:"contacts"`
}
