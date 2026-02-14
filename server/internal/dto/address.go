package dto

import "time"

type CreateAddressRequest struct {
	Line1         string   `json:"line_1"`
	Line2         string   `json:"line_2"`
	City          string   `json:"city"`
	Province      string   `json:"province"`
	PostalCode    string   `json:"postal_code"`
	Country       string   `json:"country"`
	AddressTypeID *uint    `json:"address_type_id"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	IsPastAddress bool     `json:"is_past_address"`
}

type UpdateAddressRequest struct {
	Line1         string   `json:"line_1"`
	Line2         string   `json:"line_2"`
	City          string   `json:"city"`
	Province      string   `json:"province"`
	PostalCode    string   `json:"postal_code"`
	Country       string   `json:"country"`
	AddressTypeID *uint    `json:"address_type_id"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	IsPastAddress bool     `json:"is_past_address"`
}

type AddressResponse struct {
	ID            uint      `json:"id"`
	VaultID       string    `json:"vault_id"`
	Line1         string    `json:"line_1"`
	Line2         string    `json:"line_2"`
	City          string    `json:"city"`
	Province      string    `json:"province"`
	PostalCode    string    `json:"postal_code"`
	Country       string    `json:"country"`
	AddressTypeID *uint     `json:"address_type_id"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	IsPastAddress bool      `json:"is_past_address"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
