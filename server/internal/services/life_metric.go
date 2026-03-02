package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrLifeMetricNotFound = errors.New("life metric not found")

type LifeMetricService struct {
	db *gorm.DB
}

func NewLifeMetricService(db *gorm.DB) *LifeMetricService {
	return &LifeMetricService{db: db}
}

func (s *LifeMetricService) List(vaultID string) ([]dto.LifeMetricResponse, error) {
	var metrics []models.LifeMetric
	// BUG FIX (#56): Must Preload Contacts so the linked contacts column
	// in the frontend list is populated after adding contacts.
	if err := s.db.Preload("Contacts").Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&metrics).Error; err != nil {
		return nil, err
	}
	result := make([]dto.LifeMetricResponse, len(metrics))
	for i, m := range metrics {
		result[i] = toLifeMetricResponse(&m)
	}
	return result, nil
}

func (s *LifeMetricService) Create(vaultID string, req dto.CreateLifeMetricRequest) (*dto.LifeMetricResponse, error) {
	metric := models.LifeMetric{
		VaultID: vaultID,
		Label:   req.Label,
	}
	if err := s.db.Create(&metric).Error; err != nil {
		return nil, err
	}
	resp := toLifeMetricResponse(&metric)
	return &resp, nil
}

func (s *LifeMetricService) Update(id uint, vaultID string, req dto.UpdateLifeMetricRequest) (*dto.LifeMetricResponse, error) {
	var metric models.LifeMetric
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&metric).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeMetricNotFound
		}
		return nil, err
	}
	metric.Label = req.Label
	if err := s.db.Save(&metric).Error; err != nil {
		return nil, err
	}
	resp := toLifeMetricResponse(&metric)
	return &resp, nil
}

func (s *LifeMetricService) Delete(id uint, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.LifeMetric{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLifeMetricNotFound
	}
	return nil
}

func (s *LifeMetricService) AddContact(id uint, vaultID string, contactID string) error {
	var metric models.LifeMetric
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&metric).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeMetricNotFound
		}
		return err
	}
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	clm := models.ContactLifeMetric{
		ContactID:    contactID,
		LifeMetricID: id,
	}
	return s.db.Create(&clm).Error
}

// RemoveContact removes a contact association from a life metric.
// BUG FIX (#56): This endpoint was missing — the frontend tried to call
// DELETE /lifeMetrics/:id/contacts/:contactId but no route existed.
func (s *LifeMetricService) RemoveContact(id uint, vaultID string, contactID string) error {
	var metric models.LifeMetric
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&metric).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLifeMetricNotFound
		}
		return err
	}
	result := s.db.Where("life_metric_id = ? AND contact_id = ?", id, contactID).Delete(&models.ContactLifeMetric{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrContactNotFound
	}
	return nil
}

func toLifeMetricResponse(m *models.LifeMetric) dto.LifeMetricResponse {
	resp := dto.LifeMetricResponse{
		ID:        m.ID,
		VaultID:   m.VaultID,
		Label:     m.Label,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	contacts := make([]dto.LifeMetricContactBrief, len(m.Contacts))
	for i, c := range m.Contacts {
		contacts[i] = dto.LifeMetricContactBrief{
			ID:        c.ID,
			FirstName: ptrToStr(c.FirstName),
			LastName:  ptrToStr(c.LastName),
		}
	}
	resp.Contacts = contacts
	return resp
}
