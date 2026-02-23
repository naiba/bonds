package models

import "time"

type WebAuthnCredential struct {
	ID              uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          string `json:"user_id" gorm:"type:text;not null;index"`
	CredentialID    []byte `json:"-" gorm:"type:bytea;not null;uniqueIndex"`
	PublicKey       []byte `json:"-" gorm:"type:bytea;not null"`
	AttestationType string `json:"attestation_type" gorm:"type:text"`
	AAGUID          []byte `json:"-" gorm:"type:bytea"`
	SignCount       uint32 `json:"sign_count" gorm:"default:0"`
	Name            string `json:"name" gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
