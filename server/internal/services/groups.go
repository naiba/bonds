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
		VaultID:     vaultID,
		Name:        req.Name,
		GroupTypeID: req.GroupTypeID,
	}
	if err := s.db.Create(&group).Error; err != nil {
		return nil, err
	}
	resp := toGroupResponse(&group)
	return &resp, nil
}

func (s *GroupService) List(vaultID, userID string) ([]dto.GroupResponse, error) {
	var groups []models.Group
	if err := s.db.Where("vault_id = ?", vaultID).Preload("Contacts", "vault_id = ?", vaultID).Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.GroupResponse, len(groups))
	for i, g := range groups {
		resp, err := toGroupResponseWithContacts(&g, formatter)
		if err != nil {
			return nil, err
		}
		result[i] = resp
	}
	return result, nil
}

func (s *GroupService) Get(id uint, vaultID, userID string) (*dto.GroupResponse, error) {
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Preload("Contacts", "vault_id = ?", vaultID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	resp, err := toGroupResponseWithContacts(&group, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *GroupService) ListByContact(contactID, vaultID string) ([]dto.GroupResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var groups []models.Group
	if err := s.db.Joins("JOIN contact_group ON contact_group.group_id = groups.id").
		Where("contact_group.contact_id = ? AND groups.vault_id = ?", contactID, vaultID).
		Order("groups.created_at DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	result := make([]dto.GroupResponse, len(groups))
	for i, g := range groups {
		result[i] = toGroupResponse(&g)
	}
	return result, nil
}

func (s *GroupService) AddContactToGroup(contactID, vaultID string, req dto.AddContactToGroupRequest) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", req.GroupID, vaultID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
	cg := models.ContactGroup{
		GroupID:         req.GroupID,
		ContactID:       contactID,
		GroupTypeRoleID: req.GroupTypeRoleID,
	}
	return s.db.Create(&cg).Error
}

func (s *GroupService) RemoveContactFromGroup(contactID, vaultID string, groupID uint) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var group models.Group
	if err := s.db.Where("id = ? AND vault_id = ?", groupID, vaultID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
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

func toGroupResponseWithContacts(g *models.Group, formatter *contactNameFormatter) (dto.GroupResponse, error) {
	resp := toGroupResponse(g)
	contacts := make([]dto.GroupContactResponse, len(g.Contacts))
	for i := range g.Contacts {
		name, err := formatter.format(&g.Contacts[i], "")
		if err != nil {
			return dto.GroupResponse{}, err
		}
		contacts[i] = dto.GroupContactResponse{
			ID:        g.Contacts[i].ID,
			Name:      name,
			FirstName: ptrToStr(g.Contacts[i].FirstName),
			LastName:  ptrToStr(g.Contacts[i].LastName),
		}
	}
	resp.Contacts = contacts
	return resp, nil
}
