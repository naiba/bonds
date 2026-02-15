package dto

import "time"

type CreateAddressRequest struct {
	Line1         string   `json:"line_1" example:"123 Main Street"`
	Line2         string   `json:"line_2" example:"Apt 4B"`
	City          string   `json:"city" example:"San Francisco"`
	Province      string   `json:"province" example:"California"`
	PostalCode    string   `json:"postal_code" example:"94102"`
	Country       string   `json:"country" example:"US"`
	AddressTypeID *uint    `json:"address_type_id" example:"1"`
	Latitude      *float64 `json:"latitude" example:"37.7749"`
	Longitude     *float64 `json:"longitude" example:"-122.4194"`
	IsPastAddress bool     `json:"is_past_address" example:"false"`
}

type UpdateAddressRequest struct {
	Line1         string   `json:"line_1" example:"123 Main Street"`
	Line2         string   `json:"line_2" example:"Apt 4B"`
	City          string   `json:"city" example:"San Francisco"`
	Province      string   `json:"province" example:"California"`
	PostalCode    string   `json:"postal_code" example:"94102"`
	Country       string   `json:"country" example:"US"`
	AddressTypeID *uint    `json:"address_type_id" example:"1"`
	Latitude      *float64 `json:"latitude" example:"37.7749"`
	Longitude     *float64 `json:"longitude" example:"-122.4194"`
	IsPastAddress bool     `json:"is_past_address" example:"false"`
}

type AddressResponse struct {
	ID            uint      `json:"id" example:"1"`
	VaultID       string    `json:"vault_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Line1         string    `json:"line_1" example:"123 Main Street"`
	Line2         string    `json:"line_2" example:"Apt 4B"`
	City          string    `json:"city" example:"San Francisco"`
	Province      string    `json:"province" example:"California"`
	PostalCode    string    `json:"postal_code" example:"94102"`
	Country       string    `json:"country" example:"US"`
	AddressTypeID *uint     `json:"address_type_id" example:"1"`
	Latitude      *float64  `json:"latitude" example:"37.7749"`
	Longitude     *float64  `json:"longitude" example:"-122.4194"`
	IsPastAddress bool      `json:"is_past_address" example:"false"`
	CreatedAt     time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
