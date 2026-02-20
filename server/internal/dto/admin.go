package dto

import "time"

type AdminUserResponse struct {
	ID                      string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID               string    `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName               string    `json:"first_name" example:"John"`
	LastName                string    `json:"last_name" example:"Doe"`
	Email                   string    `json:"email" example:"user@example.com"`
	IsAccountAdministrator  bool      `json:"is_account_administrator" example:"true"`
	IsInstanceAdministrator bool      `json:"is_instance_administrator" example:"false"`
	Disabled                bool      `json:"disabled" example:"false"`
	ContactCount            int64     `json:"contact_count" example:"42"`
	StorageUsed             int64     `json:"storage_used" example:"10485760"`
	VaultCount              int64     `json:"vault_count" example:"2"`
	CreatedAt               time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}

type AdminToggleUserRequest struct {
	Disabled *bool `json:"disabled" example:"true"`
}

type AdminSetAdminRequest struct {
	IsInstanceAdministrator bool `json:"is_instance_administrator" example:"true"`
}

type SystemSettingItem struct {
	Key   string `json:"key" example:"oauth.github.key"`
	Value string `json:"value" example:"my-client-id"`
}

type SystemSettingsResponse struct {
	Settings []SystemSettingItem `json:"settings"`
}

type UpdateSystemSettingsRequest struct {
	Settings []SystemSettingItem `json:"settings" validate:"required"`
}

type InstanceInfoResponse struct {
	Version             string   `json:"version" example:"v0.1.5"`
	RegistrationEnabled bool     `json:"registration_enabled" example:"true"`
	PasswordAuthEnabled bool     `json:"password_auth_enabled" example:"true"`
	OAuthProviders      []string `json:"oauth_providers" example:"github,google"`
	WebAuthnEnabled     bool     `json:"webauthn_enabled" example:"true"`
	AppName             string   `json:"app_name" example:"Bonds"`
}
