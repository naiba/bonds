package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrGroupTypeNotFound = errors.New("group type not found")
var ErrGroupTypeRoleNotFound = errors.New("group type role not found")

type GroupTypeRoleService struct {
	db *gorm.DB
}

func NewGroupTypeRoleService(db *gorm.DB) *GroupTypeRoleService {
	return &GroupTypeRoleService{db: db}
}

func (s *GroupTypeRoleService) Create(accountID string, groupTypeID uint, req dto.CreateGroupTypeRoleRequest) (*dto.GroupTypeRoleResponse, error) {
	var gt models.GroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupTypeNotFound
		}
		return nil, err
	}
	role := models.GroupTypeRole{
		GroupTypeID: groupTypeID,
		Label:       strPtrOrNil(req.Label),
		Position:    req.Position,
	}
	if err := s.db.Create(&role).Error; err != nil {
		return nil, err
	}
	resp := toGroupTypeRoleResponse(&role)
	return &resp, nil
}

func (s *GroupTypeRoleService) Update(accountID string, groupTypeID uint, roleID uint, req dto.UpdateGroupTypeRoleRequest) (*dto.GroupTypeRoleResponse, error) {
	var gt models.GroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupTypeNotFound
		}
		return nil, err
	}
	var role models.GroupTypeRole
	if err := s.db.Where("id = ? AND group_type_id = ?", roleID, groupTypeID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupTypeRoleNotFound
		}
		return nil, err
	}
	role.Label = strPtrOrNil(req.Label)
	role.Position = req.Position
	if err := s.db.Save(&role).Error; err != nil {
		return nil, err
	}
	resp := toGroupTypeRoleResponse(&role)
	return &resp, nil
}

func (s *GroupTypeRoleService) Delete(accountID string, groupTypeID uint, roleID uint) error {
	var gt models.GroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupTypeNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND group_type_id = ?", roleID, groupTypeID).Delete(&models.GroupTypeRole{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGroupTypeRoleNotFound
	}
	return nil
}

func (s *GroupTypeRoleService) UpdatePosition(accountID string, groupTypeID uint, roleID uint, position int) error {
	var gt models.GroupType
	if err := s.db.Where("id = ? AND account_id = ?", groupTypeID, accountID).First(&gt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupTypeNotFound
		}
		return err
	}
	result := s.db.Model(&models.GroupTypeRole{}).Where("id = ? AND group_type_id = ?", roleID, groupTypeID).Update("position", position)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGroupTypeRoleNotFound
	}
	return nil
}

func toGroupTypeRoleResponse(r *models.GroupTypeRole) dto.GroupTypeRoleResponse {
	return dto.GroupTypeRoleResponse{
		ID:          r.ID,
		GroupTypeID: r.GroupTypeID,
		Label:       ptrToStr(r.Label),
		Position:    r.Position,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
