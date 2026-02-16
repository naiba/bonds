package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrGroupNotFound = errors.New("group not found")

type GroupService struct {
	db *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

func (s *GroupService) Create(vaultID string, req dto.CreateGroupRequest) (*dto.GroupResponse, error) {
	group := models.Group{
		VaultID: vaultID,
		Name:    req.Name,
	}
	if err := s.db.Create(&group).Error; err != nil {
		return nil, err
	}
	resp := toGroupResponse(&group)
	return &resp, nil
}

func (s *GroupService) List(vaultID string) ([]dto.GroupResponse, error) {
	var groups []models.Group
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	result := make([]dto.GroupResponse, len(groups))
	for i, g := range groups {
		result[i] = toGroupResponse(&g)
	}
	return result, nil
}

func (s *GroupService) Get(id uint, vaultID string) (*dto.GroupResponse, error) {
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Preload("Contacts").First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	resp := toGroupResponseWithContacts(&group)
	return &resp, nil
}

func (s *GroupService) AddContactToGroup(contactID string, req dto.AddContactToGroupRequest) error {
	cg := models.ContactGroup{
		GroupID:         req.GroupID,
		ContactID:       contactID,
		GroupTypeRoleID: req.GroupTypeRoleID,
	}
	return s.db.Create(&cg).Error
}

func (s *GroupService) RemoveContactFromGroup(contactID string, groupID uint) error {
	return s.db.Where("contact_id = ? AND group_id = ?", contactID, groupID).Delete(&models.ContactGroup{}).Error
}

func (s *GroupService) Update(id uint, vaultID string, req dto.UpdateGroupRequest) (*dto.GroupResponse, error) {
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	group.Name = req.Name
	group.GroupTypeID = req.GroupTypeID
	if err := s.db.Save(&group).Error; err != nil {
		return nil, err
	}
	resp := toGroupResponse(&group)
	return &resp, nil
}

func (s *GroupService) Delete(id uint, vaultID string) error {
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&models.ContactGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(&group).Error
	})
}

func toGroupResponse(g *models.Group) dto.GroupResponse {
	return dto.GroupResponse{
		ID:          g.ID,
		VaultID:     g.VaultID,
		GroupTypeID: g.GroupTypeID,
		Name:        g.Name,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func toGroupResponseWithContacts(g *models.Group) dto.GroupResponse {
	resp := toGroupResponse(g)
	contacts := make([]dto.GroupContactResponse, len(g.Contacts))
	for i, c := range g.Contacts {
		contacts[i] = dto.GroupContactResponse{
			ID:        c.ID,
			FirstName: ptrToStr(c.FirstName),
			LastName:  ptrToStr(c.LastName),
		}
	}
	resp.Contacts = contacts
	return resp
}
