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

func (s *ContactInformationService) List(contactID, vaultID string) ([]dto.ContactInformationResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
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

func (s *ContactInformationService) Create(contactID, vaultID string, req dto.CreateContactInformationRequest) (*dto.ContactInformationResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
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

func (s *ContactInformationService) Update(id uint, contactID, vaultID string, req dto.UpdateContactInformationRequest) (*dto.ContactInformationResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
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

func (s *ContactInformationService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactInformation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactInformationNotFound
	}
	return nil
}

// FindByIdentity locates contact_information rows in a vault whose data
// matches a given identity value (case-insensitive). When typeID > 0 the
// search is further constrained to that ContactInformationType.
func (s *ContactInformationService) FindByIdentity(vaultID, data string, typeID uint) ([]dto.ContactInformationByIdentityMatch, error) {
	if vaultID == "" || data == "" {
		return []dto.ContactInformationByIdentityMatch{}, nil
	}

	type joinedRow struct {
		models.ContactInformation
		FirstName string
		LastName  string
	}

	q := s.db.
		Table("contact_information AS ci").
		Select("ci.*, c.first_name AS first_name, c.last_name AS last_name").
		Joins("JOIN contacts c ON c.id = ci.contact_id").
		Where("c.vault_id = ?", vaultID).
		Where("LOWER(ci.data) = LOWER(?)", data)
	if typeID > 0 {
		q = q.Where("ci.type_id = ?", typeID)
	}

	var rows []joinedRow
	if err := q.Order("ci.created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]dto.ContactInformationByIdentityMatch, len(rows))
	for i, r := range rows {
		ci := r.ContactInformation
		out[i] = dto.ContactInformationByIdentityMatch{
			ContactID:          ci.ContactID,
			ContactFirstName:   r.FirstName,
			ContactLastName:    r.LastName,
			ContactInformation: toContactInformationResponse(&ci),
		}
	}
	return out, nil
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
