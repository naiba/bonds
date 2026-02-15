package dto

import "time"

type CreateRelationshipRequest struct {
	RelationshipTypeID uint   `json:"relationship_type_id" validate:"required" example:"1"`
	RelatedContactID   string `json:"related_contact_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateRelationshipRequest struct {
	RelationshipTypeID uint   `json:"relationship_type_id" validate:"required" example:"1"`
	RelatedContactID   string `json:"related_contact_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type RelationshipResponse struct {
	ID                 uint      `json:"id" example:"1"`
	ContactID          string    `json:"contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RelatedContactID   string    `json:"related_contact_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RelationshipTypeID uint      `json:"relationship_type_id" example:"1"`
	CreatedAt          time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt          time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
