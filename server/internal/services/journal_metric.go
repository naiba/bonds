package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrJournalMetricNotFound = errors.New("journal metric not found")

type JournalMetricService struct {
	db *gorm.DB
}

func NewJournalMetricService(db *gorm.DB) *JournalMetricService {
	return &JournalMetricService{db: db}
}

func (s *JournalMetricService) List(journalID uint, vaultID string) ([]dto.JournalMetricResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var metrics []models.JournalMetric
	if err := s.db.Where("journal_id = ?", journalID).Order("created_at DESC").Find(&metrics).Error; err != nil {
		return nil, err
	}
	result := make([]dto.JournalMetricResponse, len(metrics))
	for i, m := range metrics {
		result[i] = toJournalMetricResponse(&m)
	}
	return result, nil
}

func (s *JournalMetricService) Create(journalID uint, vaultID string, req dto.CreateJournalMetricRequest) (*dto.JournalMetricResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	metric := models.JournalMetric{
		JournalID: journalID,
		Label:     req.Label,
	}
	if err := s.db.Create(&metric).Error; err != nil {
		return nil, err
	}
	resp := toJournalMetricResponse(&metric)
	return &resp, nil
}

func (s *JournalMetricService) Delete(id uint, journalID uint, vaultID string) error {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJournalNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND journal_id = ?", id, journalID).Delete(&models.JournalMetric{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrJournalMetricNotFound
	}
	return nil
}

func toJournalMetricResponse(m *models.JournalMetric) dto.JournalMetricResponse {
	return dto.JournalMetricResponse{
		ID:        m.ID,
		JournalID: m.JournalID,
		Label:     m.Label,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
