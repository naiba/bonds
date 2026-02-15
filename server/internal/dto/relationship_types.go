package dto

import "time"

type CreateRelationshipTypeRequest struct {
	Name                    string `json:"name" validate:"required" example:"Parent"`
	NameReverseRelationship string `json:"name_reverse_relationship" validate:"required" example:"Child"`
}

type UpdateRelationshipTypeRequest struct {
	Name                    string `json:"name" validate:"required" example:"Parent"`
	NameReverseRelationship string `json:"name_reverse_relationship" validate:"required" example:"Child"`
}

type RelationshipTypeResponse struct {
	ID                      uint      `json:"id" example:"1"`
	RelationshipGroupTypeID uint      `json:"relationship_group_type_id" example:"1"`
	Name                    string    `json:"name" example:"Parent"`
	NameReverseRelationship string    `json:"name_reverse_relationship" example:"Child"`
	CanBeDeleted            bool      `json:"can_be_deleted" example:"false"`
	CreatedAt               time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
	UpdatedAt               time.Time `json:"updated_at" example:"2026-01-15T10:30:00Z"`
}
