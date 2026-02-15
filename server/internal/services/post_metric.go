package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPostMetricNotFound = errors.New("post metric not found")

type PostMetricService struct {
	db *gorm.DB
}

func NewPostMetricService(db *gorm.DB) *PostMetricService {
	return &PostMetricService{db: db}
}

func (s *PostMetricService) Create(postID uint, journalID uint, req dto.CreatePostMetricRequest) (*dto.PostMetricResponse, error) {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	var jm models.JournalMetric
	if err := s.db.Where("id = ? AND journal_id = ?", req.JournalMetricID, journalID).First(&jm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalMetricNotFound
		}
		return nil, err
	}

	pm := models.PostMetric{
		PostID:          postID,
		JournalMetricID: req.JournalMetricID,
		Value:           req.Value,
		Label:           &jm.Label,
	}
	if err := s.db.Create(&pm).Error; err != nil {
		return nil, err
	}
	resp := toPostMetricResponse(&pm)
	return &resp, nil
}

func (s *PostMetricService) Delete(metricID uint, postID uint, journalID uint) error {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND post_id = ?", metricID, postID).Delete(&models.PostMetric{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPostMetricNotFound
	}
	return nil
}

func toPostMetricResponse(pm *models.PostMetric) dto.PostMetricResponse {
	return dto.PostMetricResponse{
		ID:              pm.ID,
		PostID:          pm.PostID,
		JournalMetricID: pm.JournalMetricID,
		Value:           pm.Value,
		Label:           ptrToStr(pm.Label),
		CreatedAt:       pm.CreatedAt,
	}
}
