package dto

import "time"

type CreateDavSubscriptionRequest struct {
	URI       string `json:"uri" validate:"required" example:"https://dav.example.com/addressbooks/user/contacts/"`
	Username  string `json:"username" validate:"required" example:"user@example.com"`
	Password  string `json:"password" validate:"required" example:"app-password"`
	SyncWay   uint8  `json:"sync_way" example:"2"`
	Frequency int    `json:"frequency" example:"180"`
	AddressBookPath string `json:"address_book_path" example:"/dav.php/addressbooks/user/contacts/"`
}

type UpdateDavSubscriptionRequest struct {
	URI       string `json:"uri" example:"https://dav.example.com/addressbooks/user/contacts/"`
	Username  string `json:"username" example:"user@example.com"`
	Password  string `json:"password" example:"new-password"`
	SyncWay   uint8  `json:"sync_way" example:"2"`
	Frequency int    `json:"frequency" example:"180"`
	Active    *bool  `json:"active" example:"true"`
	AddressBookPath string `json:"address_book_path" example:"/dav.php/addressbooks/user/contacts/"`
}

type TestDavConnectionRequest struct {
	URI      string `json:"uri" validate:"required" example:"https://dav.example.com/addressbooks/user/contacts/"`
	Username string `json:"username" validate:"required" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"app-password"`
}

type DavSubscriptionResponse struct {
	ID                 string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VaultID            string     `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	URI                string     `json:"uri" example:"https://dav.example.com/addressbooks/user/contacts/"`
	Username           string     `json:"username" example:"user@example.com"`
	AddressBookPath    string     `json:"address_book_path" example:"/dav.php/addressbooks/user/contacts/"`
	Active             bool       `json:"active" example:"true"`
	SyncWay            uint8      `json:"sync_way" example:"2"`
	Frequency          int        `json:"frequency" example:"180"`
	LastSynchronizedAt *time.Time `json:"last_synchronized_at"`
	CreatedAt          time.Time  `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt          time.Time  `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}

type AddressBookInfo struct {
	Name string `json:"name" example:"Contacts"`
	Path string `json:"path" example:"/dav.php/addressbooks/user/contacts/"`
}

type TestDavConnectionResponse struct {
	Success      bool              `json:"success" example:"true"`
	AddressBooks []AddressBookInfo `json:"address_books,omitempty"`
	Error        string            `json:"error,omitempty"`
}

type DavSyncLogResponse struct {
	ID         uint      `json:"id" example:"1"`
	ContactID  *string   `json:"contact_id"`
	DistantURI string    `json:"distant_uri" example:"/addressbooks/user/contacts/abc.vcf"`
	Action     string    `json:"action" example:"created"`
	Error      *string   `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type TriggerSyncResponse struct {
	Created int `json:"created" example:"5"`
	Updated int `json:"updated" example:"3"`
	Deleted int `json:"deleted" example:"1"`
	Skipped int `json:"skipped" example:"10"`
	Errors  int `json:"errors" example:"0"`
}
