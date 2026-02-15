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
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&metrics).Error; err != nil {
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

func toLifeMetricResponse(m *models.LifeMetric) dto.LifeMetricResponse {
	return dto.LifeMetricResponse{
		ID:        m.ID,
		VaultID:   m.VaultID,
		Label:     m.Label,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
