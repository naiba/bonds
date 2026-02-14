package dto

import "time"

type CreateRelationshipRequest struct {
	RelationshipTypeID uint   `json:"relationship_type_id" validate:"required"`
	RelatedContactID   string `json:"related_contact_id" validate:"required"`
}

type UpdateRelationshipRequest struct {
	RelationshipTypeID uint   `json:"relationship_type_id" validate:"required"`
	RelatedContactID   string `json:"related_contact_id" validate:"required"`
}

type RelationshipResponse struct {
	ID                 uint      `json:"id"`
	ContactID          string    `json:"contact_id"`
	RelatedContactID   string    `json:"related_contact_id"`
	RelationshipTypeID uint      `json:"relationship_type_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
