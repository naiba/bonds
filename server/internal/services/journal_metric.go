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
	return s.db.Transaction(func(tx *gorm.DB) error {
		var metric models.JournalMetric
		if err := tx.Where("id = ? AND journal_id = ?", id, journalID).First(&metric).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrJournalMetricNotFound
			}
			return err
		}
		if err := tx.Where("journal_metric_id = ?", metric.ID).Delete(&models.PostMetric{}).Error; err != nil {
			return err
		}
		return tx.Delete(&metric).Error
	})
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
