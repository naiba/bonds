package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrContactLabelNotFound = errors.New("contact label not found")

type ContactLabelService struct {
	db *gorm.DB
}

func NewContactLabelService(db *gorm.DB) *ContactLabelService {
	return &ContactLabelService{db: db}
}

func (s *ContactLabelService) List(contactID, vaultID string) ([]dto.ContactLabelResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var pivots []models.ContactLabel
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&pivots).Error; err != nil {
		return nil, err
	}
	if len(pivots) == 0 {
		return []dto.ContactLabelResponse{}, nil
	}

	labelIDs := make([]uint, len(pivots))
	for i, p := range pivots {
		labelIDs[i] = p.LabelID
	}
	var labels []models.Label
	if err := s.db.Where("id IN ?", labelIDs).Find(&labels).Error; err != nil {
		return nil, err
	}
	labelMap := make(map[uint]models.Label)
	for _, l := range labels {
		labelMap[l.ID] = l
	}

	result := make([]dto.ContactLabelResponse, len(pivots))
	for i, p := range pivots {
		l := labelMap[p.LabelID]
		result[i] = dto.ContactLabelResponse{
			ID:        p.ID,
			LabelID:   p.LabelID,
			Name:      l.Name,
			BgColor:   l.BgColor,
			TextColor: l.TextColor,
			CreatedAt: p.CreatedAt,
		}
	}
	return result, nil
}

func (s *ContactLabelService) Add(contactID, vaultID string, req dto.AddContactLabelRequest) (*dto.ContactLabelResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var label models.Label
	if err := s.db.Where("id = ? AND vault_id = ?", req.LabelID, vaultID).First(&label).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}

	pivot := models.ContactLabel{
		LabelID:   req.LabelID,
		ContactID: contactID,
	}
	if err := s.db.Create(&pivot).Error; err != nil {
		return nil, err
	}

	resp := dto.ContactLabelResponse{
		ID:        pivot.ID,
		LabelID:   pivot.LabelID,
		Name:      label.Name,
		BgColor:   label.BgColor,
		TextColor: label.TextColor,
		CreatedAt: pivot.CreatedAt,
	}
	return &resp, nil
}

func (s *ContactLabelService) Update(contactID, vaultID string, pivotID uint, req dto.UpdateContactLabelRequest) (*dto.ContactLabelResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var pivot models.ContactLabel
	if err := s.db.Where("id = ? AND contact_id = ?", pivotID, contactID).First(&pivot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactLabelNotFound
		}
		return nil, err
	}
	var newLabel models.Label
	if err := s.db.Where("id = ? AND vault_id = ?", req.LabelID, vaultID).First(&newLabel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}
	pivot.LabelID = req.LabelID
	if err := s.db.Save(&pivot).Error; err != nil {
		return nil, err
	}
	resp := dto.ContactLabelResponse{
		ID:        pivot.ID,
		LabelID:   pivot.LabelID,
		Name:      newLabel.Name,
		BgColor:   newLabel.BgColor,
		TextColor: newLabel.TextColor,
		CreatedAt: pivot.CreatedAt,
	}
	return &resp, nil
}

func (s *ContactLabelService) Remove(contactID, vaultID string, labelID uint) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", labelID, contactID).Delete(&models.ContactLabel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactLabelNotFound
	}
	return nil
}
