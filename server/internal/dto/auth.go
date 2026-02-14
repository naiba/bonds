package dto

import "time"

type RegisterRequest struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code"`
}

type RefreshRequest struct {
	Token string `json:"token" validate:"required"`
}

type AuthResponse struct {
	Token             string       `json:"token"`
	ExpiresAt         time.Time    `json:"expires_at"`
	User              UserResponse `json:"user"`
	RequiresTwoFactor bool         `json:"requires_two_factor,omitempty"`
	TempToken         string       `json:"temp_token,omitempty"`
}

type TwoFactorSetupResponse struct {
	Secret        string   `json:"secret"`
	QRCodeURL     string   `json:"qr_code_url"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type TwoFactorVerifyRequest struct {
	Code string `json:"code" validate:"required"`
}

type TwoFactorStatusResponse struct {
	Enabled bool `json:"enabled"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}
