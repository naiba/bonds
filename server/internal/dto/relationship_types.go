package dto

import "time"

type CreateRelationshipTypeRequest struct {
	Name                    string `json:"name" validate:"required"`
	NameReverseRelationship string `json:"name_reverse_relationship" validate:"required"`
}

type UpdateRelationshipTypeRequest struct {
	Name                    string `json:"name" validate:"required"`
	NameReverseRelationship string `json:"name_reverse_relationship" validate:"required"`
}

type RelationshipTypeResponse struct {
	ID                      uint      `json:"id"`
	RelationshipGroupTypeID uint      `json:"relationship_group_type_id"`
	Name                    string    `json:"name"`
	NameReverseRelationship string    `json:"name_reverse_relationship"`
	CanBeDeleted            bool      `json:"can_be_deleted"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}
