package dto

import "time"

type VaultFileResponse struct {
	ID        uint      `json:"id"`
	VaultID   string    `json:"vault_id"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	MimeType  string    `json:"mime_type"`
	Type      string    `json:"type"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
