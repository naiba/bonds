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

func (s *RelationshipTypeService) List(accountID string, groupTypeID uint) ([]dto.RelationshipTypeResponse, error) {
	var gt models.RelationshipGroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipGroupTypeNotFound
		}
		return nil, err
	}
	var types []models.RelationshipType
	if err := s.db.Where("relationship_group_type_id = ?", groupTypeID).Order("id ASC").Find(&types).Error; err != nil {
		return nil, err
	}
	result := make([]dto.RelationshipTypeResponse, len(types))
	for i, rt := range types {
		result[i] = toRelationshipTypeResponse(&rt)
	}
	return result, nil
}

func (s *RelationshipTypeService) Create(accountID string, groupTypeID uint, req dto.CreateRelationshipTypeRequest) (*dto.RelationshipTypeResponse, error) {
	var gt models.RelationshipGroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipGroupTypeNotFound
		}
		return nil, err
	}

	var rt models.RelationshipType
	err := s.db.Transaction(func(tx *gorm.DB) error {
		rt = models.RelationshipType{
			RelationshipGroupTypeID: groupTypeID,
			Name:                    strPtrOrNil(req.Name),
			NameReverseRelationship: strPtrOrNil(req.NameReverseRelationship),
			Degree:                  req.Degree,
			CanBeDeleted:            true,
		}
		if err := tx.Create(&rt).Error; err != nil {
			return err
		}

		isSymmetric := req.Name == req.NameReverseRelationship
		if isSymmetric {
			// Symmetric type (e.g. Friend↔Friend): point to self.
			rt.ReverseRelationshipTypeID = &rt.ID
			return tx.Model(&rt).Update("reverse_relationship_type_id", rt.ID).Error
		}

		if req.NameReverseRelationship != "" {
			// Auto-create the reverse type and link them bidirectionally.
			reverseRT := models.RelationshipType{
				RelationshipGroupTypeID:  groupTypeID,
				Name:                     strPtrOrNil(req.NameReverseRelationship),
				NameReverseRelationship:  strPtrOrNil(req.Name),
				Degree:                   req.Degree,
				CanBeDeleted:             true,
				ReverseRelationshipTypeID: &rt.ID,
			}
			if err := tx.Create(&reverseRT).Error; err != nil {
				return err
			}
			rt.ReverseRelationshipTypeID = &reverseRT.ID
			return tx.Model(&rt).Update("reverse_relationship_type_id", reverseRT.ID).Error
		}
		return nil
	})
	if err != nil {
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
	rt.Degree = req.Degree
	if err := s.db.Save(&rt).Error; err != nil {
		return nil, err
	}

	// Sync name change to the reverse counterpart's NameReverseRelationship.
	// This keeps the display text consistent when a user renames one side.
	if rt.ReverseRelationshipTypeID != nil && *rt.ReverseRelationshipTypeID != rt.ID {
		s.db.Model(&models.RelationshipType{}).Where("id = ?", *rt.ReverseRelationshipTypeID).
			Update("name_reverse_relationship", req.Name)
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
	// Clear the reverse pointer on the counterpart before deleting, so it
	// doesn't dangle. Tolerant — ignore if counterpart already gone.
	if rt.ReverseRelationshipTypeID != nil && *rt.ReverseRelationshipTypeID != rt.ID {
		s.db.Model(&models.RelationshipType{}).Where("id = ?", *rt.ReverseRelationshipTypeID).
			Update("reverse_relationship_type_id", nil)
	}
	return s.db.Delete(&rt).Error
}

func toRelationshipTypeResponse(rt *models.RelationshipType) dto.RelationshipTypeResponse {
	return dto.RelationshipTypeResponse{
		ID:                        rt.ID,
		RelationshipGroupTypeID:   rt.RelationshipGroupTypeID,
		Name:                      ptrToStr(rt.Name),
		NameReverseRelationship:   ptrToStr(rt.NameReverseRelationship),
		ReverseRelationshipTypeID: rt.ReverseRelationshipTypeID,
		Degree:                    rt.Degree,
		CanBeDeleted:              rt.CanBeDeleted,
		CreatedAt:                 rt.CreatedAt,
		UpdatedAt:                 rt.UpdatedAt,
	}
}

// ListAll returns all relationship types across all groups for the account,
// including the group name for frontend grouped select rendering.
func (s *RelationshipTypeService) ListAll(accountID string) ([]dto.RelationshipTypeWithGroupResponse, error) {
	var groups []models.RelationshipGroupType
	if err := s.db.Where("account_id = ?", accountID).Preload("Types", func(db *gorm.DB) *gorm.DB {
		return db.Order("id ASC")
	}).Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	var result []dto.RelationshipTypeWithGroupResponse
	for _, g := range groups {
		groupName := ptrToStr(g.Name)
		for _, rt := range g.Types {
			result = append(result, dto.RelationshipTypeWithGroupResponse{
				ID:                      rt.ID,
				RelationshipGroupTypeID: rt.RelationshipGroupTypeID,
				GroupName:               groupName,
				Name:                    ptrToStr(rt.Name),
				NameReverseRelationship: ptrToStr(rt.NameReverseRelationship),
			})
		}
	}
	if result == nil {
		result = []dto.RelationshipTypeWithGroupResponse{}
	}
	return result, nil
}
