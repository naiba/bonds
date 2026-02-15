package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrRelationshipNotFound = errors.New("relationship not found")

type RelationshipService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewRelationshipService(db *gorm.DB) *RelationshipService {
	return &RelationshipService{db: db}
}

func (s *RelationshipService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *RelationshipService) List(contactID, vaultID string) ([]dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var relationships []models.Relationship
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&relationships).Error; err != nil {
		return nil, err
	}
	result := make([]dto.RelationshipResponse, len(relationships))
	for i, r := range relationships {
		result[i] = toRelationshipResponse(&r)
	}
	return result, nil
}

func (s *RelationshipService) Create(contactID, vaultID string, req dto.CreateRelationshipRequest) (*dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	relationship := models.Relationship{
		ContactID:          contactID,
		RelationshipTypeID: req.RelationshipTypeID,
		RelatedContactID:   req.RelatedContactID,
	}
	if err := s.db.Create(&relationship).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Relationship"
		s.feedRecorder.Record(contactID, "", ActionRelationshipAdded, "Added a relationship", &relationship.ID, &entityType)
	}

	resp := toRelationshipResponse(&relationship)
	return &resp, nil
}

func (s *RelationshipService) Update(id uint, contactID, vaultID string, req dto.UpdateRelationshipRequest) (*dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var relationship models.Relationship
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&relationship).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipNotFound
		}
		return nil, err
	}
	relationship.RelationshipTypeID = req.RelationshipTypeID
	relationship.RelatedContactID = req.RelatedContactID
	if err := s.db.Save(&relationship).Error; err != nil {
		return nil, err
	}
	resp := toRelationshipResponse(&relationship)
	return &resp, nil
}

func (s *RelationshipService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.Relationship{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRelationshipNotFound
	}
	return nil
}

func toRelationshipResponse(r *models.Relationship) dto.RelationshipResponse {
	return dto.RelationshipResponse{
		ID:                 r.ID,
		ContactID:          r.ContactID,
		RelatedContactID:   r.RelatedContactID,
		RelationshipTypeID: r.RelationshipTypeID,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}
