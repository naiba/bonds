package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrRelationshipGroupTypeNotFound = errors.New("relationship group type not found")
var ErrRelationshipTypeNotFound = errors.New("relationship type not found")
var ErrRelationshipTypeCannotBeDeleted = errors.New("relationship type cannot be deleted")

type RelationshipTypeService struct {
	db *gorm.DB
}

func NewRelationshipTypeService(db *gorm.DB) *RelationshipTypeService {
	return &RelationshipTypeService{db: db}
}

func (s *RelationshipTypeService) Create(accountID string, groupTypeID uint, req dto.CreateRelationshipTypeRequest) (*dto.RelationshipTypeResponse, error) {
	var gt models.RelationshipGroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipGroupTypeNotFound
		}
		return nil, err
	}
	rt := models.RelationshipType{
		RelationshipGroupTypeID: groupTypeID,
		Name:                    strPtrOrNil(req.Name),
		NameReverseRelationship: strPtrOrNil(req.NameReverseRelationship),
		CanBeDeleted:            true,
	}
	if err := s.db.Create(&rt).Error; err != nil {
		return nil, err
	}
	resp := toRelationshipTypeResponse(&rt)
	return &resp, nil
}

func (s *RelationshipTypeService) Update(accountID string, groupTypeID uint, typeID uint, req dto.UpdateRelationshipTypeRequest) (*dto.RelationshipTypeResponse, error) {
	var gt models.RelationshipGroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipGroupTypeNotFound
		}
		return nil, err
	}
	var rt models.RelationshipType
	if err := s.db.Where("id = ? AND relationship_group_type_id = ?", typeID, groupTypeID).First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipTypeNotFound
		}
		return nil, err
	}
	rt.Name = strPtrOrNil(req.Name)
	rt.NameReverseRelationship = strPtrOrNil(req.NameReverseRelationship)
	if err := s.db.Save(&rt).Error; err != nil {
		return nil, err
	}
	resp := toRelationshipTypeResponse(&rt)
	return &resp, nil
}

func (s *RelationshipTypeService) Delete(accountID string, groupTypeID uint, typeID uint) error {
	var gt models.RelationshipGroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRelationshipGroupTypeNotFound
		}
		return err
	}
	var rt models.RelationshipType
	if err := s.db.Where("id = ? AND relationship_group_type_id = ?", typeID, groupTypeID).First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRelationshipTypeNotFound
		}
		return err
	}
	if !rt.CanBeDeleted {
		return ErrRelationshipTypeCannotBeDeleted
	}
	return s.db.Delete(&rt).Error
}

func toRelationshipTypeResponse(rt *models.RelationshipType) dto.RelationshipTypeResponse {
	return dto.RelationshipTypeResponse{
		ID:                      rt.ID,
		RelationshipGroupTypeID: rt.RelationshipGroupTypeID,
		Name:                    ptrToStr(rt.Name),
		NameReverseRelationship: ptrToStr(rt.NameReverseRelationship),
		CanBeDeleted:            rt.CanBeDeleted,
		CreatedAt:               rt.CreatedAt,
		UpdatedAt:               rt.UpdatedAt,
	}
}
