package dto

import "time"

type RegisterRequest struct {
	FirstName string `json:"first_name" validate:"required" example:"John"`
	LastName  string `json:"last_name" validate:"required" example:"Doe"`
	Email     string `json:"email" validate:"required,email" example:"user@example.com"`
	Password  string `json:"password" validate:"required,min=8" example:"secureP@ss123"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"secureP@ss123"`
	TOTPCode string `json:"totp_code" example:"123456"`
}

type RefreshRequest struct {
	Token string `json:"token" validate:"required" example:"eyJhbGciOiJIUzI1NiIs..."`
}

type AuthResponse struct {
	Token             string       `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresAt         time.Time    `json:"expires_at" example:"2026-01-15T10:30:00Z"`
	User              UserResponse `json:"user"`
	RequiresTwoFactor bool         `json:"requires_two_factor,omitempty" example:"false"`
	TempToken         string       `json:"temp_token,omitempty" example:"eyJhbGciOiJIUzI1NiIs..."`
}

type TwoFactorSetupResponse struct {
	Secret        string   `json:"secret" example:"JBSWY3DPEHPK3PXP"`
	QRCodeURL     string   `json:"qr_code_url" example:"otpauth://totp/Bonds:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Bonds"`
	RecoveryCodes []string `json:"recovery_codes" example:"abc12345"`
}

type TwoFactorVerifyRequest struct {
	Code string `json:"code" validate:"required" example:"123456"`
}

type TwoFactorStatusResponse struct {
	Enabled bool `json:"enabled" example:"true"`
}

type UserResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID string    `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FirstName string    `json:"first_name" example:"John"`
	LastName  string    `json:"last_name" example:"Doe"`
	Email     string    `json:"email" example:"user@example.com"`
	IsAdmin   bool      `json:"is_admin" example:"true"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
