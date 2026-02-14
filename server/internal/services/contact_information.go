package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrContactInformationNotFound = errors.New("contact information not found")

type ContactInformationService struct {
	db *gorm.DB
}

func NewContactInformationService(db *gorm.DB) *ContactInformationService {
	return &ContactInformationService{db: db}
}

func (s *ContactInformationService) List(contactID string) ([]dto.ContactInformationResponse, error) {
	var items []models.ContactInformation
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]dto.ContactInformationResponse, len(items))
	for i, item := range items {
		result[i] = toContactInformationResponse(&item)
	}
	return result, nil
}

func (s *ContactInformationService) Create(contactID string, req dto.CreateContactInformationRequest) (*dto.ContactInformationResponse, error) {
	pref := true
	if req.Pref != nil {
		pref = *req.Pref
	}
	item := models.ContactInformation{
		ContactID: contactID,
		TypeID:    req.TypeID,
		Data:      req.Data,
		Kind:      strPtrOrNil(req.Kind),
		Pref:      pref,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	resp := toContactInformationResponse(&item)
	return &resp, nil
}

func (s *ContactInformationService) Update(id uint, contactID string, req dto.UpdateContactInformationRequest) (*dto.ContactInformationResponse, error) {
	var item models.ContactInformation
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactInformationNotFound
		}
		return nil, err
	}
	item.TypeID = req.TypeID
	item.Data = req.Data
	item.Kind = strPtrOrNil(req.Kind)
	if req.Pref != nil {
		item.Pref = *req.Pref
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	resp := toContactInformationResponse(&item)
	return &resp, nil
}

func (s *ContactInformationService) Delete(id uint, contactID string) error {
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactInformation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactInformationNotFound
	}
	return nil
}

func toContactInformationResponse(ci *models.ContactInformation) dto.ContactInformationResponse {
	return dto.ContactInformationResponse{
		ID:        ci.ID,
		ContactID: ci.ContactID,
		TypeID:    ci.TypeID,
		Data:      ci.Data,
		Kind:      ptrToStr(ci.Kind),
		Pref:      ci.Pref,
		CreatedAt: ci.CreatedAt,
		UpdatedAt: ci.UpdatedAt,
	}
}
