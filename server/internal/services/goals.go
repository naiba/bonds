package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrGoalNotFound = errors.New("goal not found")

type GoalService struct {
	db *gorm.DB
}

func NewGoalService(db *gorm.DB) *GoalService {
	return &GoalService{db: db}
}

func (s *GoalService) List(contactID, vaultID string) ([]dto.GoalResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var goals []models.Goal
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&goals).Error; err != nil {
		return nil, err
	}
	result := make([]dto.GoalResponse, len(goals))
	for i, g := range goals {
		result[i] = toGoalResponse(&g)
	}
	return result, nil
}

func (s *GoalService) Create(contactID, vaultID string, req dto.CreateGoalRequest) (*dto.GoalResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	goal := models.Goal{
		ContactID: contactID,
		Name:      req.Name,
	}
	if err := s.db.Create(&goal).Error; err != nil {
		return nil, err
	}
	resp := toGoalResponse(&goal)
	return &resp, nil
}

func (s *GoalService) Get(id uint, contactID, vaultID string) (*dto.GoalResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var goal models.Goal
	if err := s.db.Preload("Streaks", func(db *gorm.DB) *gorm.DB {
		return db.Order("happened_at DESC")
	}).Where("id = ? AND contact_id = ?", id, contactID).First(&goal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	resp := toGoalResponseWithStreaks(&goal)
	return &resp, nil
}

func (s *GoalService) Update(id uint, contactID, vaultID string, req dto.UpdateGoalRequest) (*dto.GoalResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var goal models.Goal
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&goal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	goal.Name = req.Name
	if req.Active != nil {
		goal.Active = *req.Active
	}
	if err := s.db.Save(&goal).Error; err != nil {
		return nil, err
	}
	resp := toGoalResponse(&goal)
	return &resp, nil
}

func (s *GoalService) AddStreak(goalID uint, contactID, vaultID string, req dto.AddStreakRequest) (*dto.GoalResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var goal models.Goal
	if err := s.db.Where("id = ? AND contact_id = ?", goalID, contactID).First(&goal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	happenedAt := req.HappenedAt
	if happenedAt.IsZero() {
		happenedAt = time.Now()
	}
	streak := models.Streak{
		GoalID:     goalID,
		HappenedAt: happenedAt,
	}
	if err := s.db.Create(&streak).Error; err != nil {
		return nil, err
	}
	return s.Get(goalID, contactID, vaultID)
}

func (s *GoalService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	var goal models.Goal
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&goal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGoalNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("goal_id = ?", id).Delete(&models.Streak{}).Error; err != nil {
			return err
		}
		return tx.Delete(&goal).Error
	})
}

func toGoalResponse(g *models.Goal) dto.GoalResponse {
	return dto.GoalResponse{
		ID:        g.ID,
		ContactID: g.ContactID,
		Name:      g.Name,
		Active:    g.Active,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

func toGoalResponseWithStreaks(g *models.Goal) dto.GoalResponse {
	resp := toGoalResponse(g)
	streaks := make([]dto.StreakResponse, len(g.Streaks))
	for i, s := range g.Streaks {
		streaks[i] = dto.StreakResponse{
			ID:         s.ID,
			GoalID:     s.GoalID,
			HappenedAt: s.HappenedAt,
			CreatedAt:  s.CreatedAt,
		}
	}
	resp.Streaks = streaks
	return resp
}
