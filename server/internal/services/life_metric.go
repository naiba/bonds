package services

import (
	"errors"
	"time"

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

func (s *LifeMetricService) List(vaultID string, userID string) ([]dto.LifeMetricResponse, error) {
	var metrics []models.LifeMetric
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&metrics).Error; err != nil {
		return nil, err
	}
	result := make([]dto.LifeMetricResponse, len(metrics))
	for i, m := range metrics {
		result[i] = s.toLifeMetricResponse(&m, userID)
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
	resp := s.toLifeMetricResponse(&metric, "")
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
	resp := s.toLifeMetricResponse(&metric, "")
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

func (s *LifeMetricService) Increment(metricID uint, vaultID string, userID string) (*dto.LifeMetricResponse, error) {
	var metric models.LifeMetric
	if err := s.db.Where("id = ? AND vault_id = ?", metricID, vaultID).First(&metric).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeMetricNotFound
		}
		return nil, err
	}
	event := models.ContactLifeMetric{
		ContactID:    userID,
		LifeMetricID: metricID,
		UserID:       userID,
	}
	if err := s.db.Create(&event).Error; err != nil {
		return nil, err
	}
	resp := s.toLifeMetricResponse(&metric, userID)
	return &resp, nil
}

func (s *LifeMetricService) GetDetail(metricID uint, vaultID string, userID string, year int) (*dto.LifeMetricDetailResponse, error) {
	var metric models.LifeMetric
	if err := s.db.Where("id = ? AND vault_id = ?", metricID, vaultID).First(&metric).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLifeMetricNotFound
		}
		return nil, err
	}

	stats := s.calcStats(metricID, userID)

	monthNames := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}

	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	var events []models.ContactLifeMetric
	query := s.db.Where("life_metric_id = ? AND user_id = ? AND created_at >= ? AND created_at < ?",
		metricID, userID, yearStart, yearEnd)
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}

	monthMap := make(map[int]int)
	for _, e := range events {
		m := int(e.CreatedAt.Month())
		monthMap[m]++
	}

	months := make([]dto.LifeMetricMonthData, 12)
	maxEvents := 0
	for i := 0; i < 12; i++ {
		count := monthMap[i+1]
		months[i] = dto.LifeMetricMonthData{
			Month:        i + 1,
			FriendlyName: monthNames[i],
			Events:       count,
		}
		if count > maxEvents {
			maxEvents = count
		}
	}

	return &dto.LifeMetricDetailResponse{
		ID:        metric.ID,
		VaultID:   metric.VaultID,
		Label:     metric.Label,
		Stats:     stats,
		Months:    months,
		MaxEvents: maxEvents,
		CreatedAt: metric.CreatedAt,
		UpdatedAt: metric.UpdatedAt,
	}, nil
}

func (s *LifeMetricService) calcStats(metricID uint, userID string) dto.LifeMetricStats {
	if userID == "" {
		return dto.LifeMetricStats{}
	}

	now := time.Now().UTC()

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	var weekly, monthly, yearly int64
	base := s.db.Model(&models.ContactLifeMetric{}).Where("life_metric_id = ? AND user_id = ?", metricID, userID)

	base.Where("created_at >= ?", weekStart).Count(&weekly)

	s.db.Model(&models.ContactLifeMetric{}).
		Where("life_metric_id = ? AND user_id = ?", metricID, userID).
		Where("created_at >= ?", monthStart).Count(&monthly)

	s.db.Model(&models.ContactLifeMetric{}).
		Where("life_metric_id = ? AND user_id = ?", metricID, userID).
		Where("created_at >= ?", yearStart).Count(&yearly)

	return dto.LifeMetricStats{
		WeeklyEvents:  int(weekly),
		MonthlyEvents: int(monthly),
		YearlyEvents:  int(yearly),
	}
}

func (s *LifeMetricService) toLifeMetricResponse(m *models.LifeMetric, userID string) dto.LifeMetricResponse {
	return dto.LifeMetricResponse{
		ID:        m.ID,
		VaultID:   m.VaultID,
		Label:     m.Label,
		Stats:     s.calcStats(m.ID, userID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
