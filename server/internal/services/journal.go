package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrJournalNotFound = errors.New("journal not found")

type JournalService struct {
	db *gorm.DB
}

func NewJournalService(db *gorm.DB) *JournalService {
	return &JournalService{db: db}
}

func (s *JournalService) List(vaultID string) ([]dto.JournalResponse, error) {
	var journals []models.Journal
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&journals).Error; err != nil {
		return nil, err
	}
	result := make([]dto.JournalResponse, len(journals))
	for i, j := range journals {
		var count int64
		s.db.Model(&models.Post{}).Where("journal_id = ?", j.ID).Count(&count)
		result[i] = toJournalResponse(&j, int(count))
	}
	return result, nil
}

func (s *JournalService) Create(vaultID string, req dto.CreateJournalRequest) (*dto.JournalResponse, error) {
	journal := models.Journal{
		VaultID:     vaultID,
		Name:        req.Name,
		Description: strPtrOrNil(req.Description),
	}
	if err := s.db.Create(&journal).Error; err != nil {
		return nil, err
	}
	resp := toJournalResponse(&journal, 0)
	return &resp, nil
}

func (s *JournalService) Get(id uint) (*dto.JournalResponse, error) {
	var journal models.Journal
	if err := s.db.First(&journal, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var count int64
	s.db.Model(&models.Post{}).Where("journal_id = ?", journal.ID).Count(&count)
	resp := toJournalResponse(&journal, int(count))
	return &resp, nil
}

func (s *JournalService) Update(id uint, req dto.UpdateJournalRequest) (*dto.JournalResponse, error) {
	var journal models.Journal
	if err := s.db.First(&journal, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	journal.Name = req.Name
	journal.Description = strPtrOrNil(req.Description)
	if err := s.db.Save(&journal).Error; err != nil {
		return nil, err
	}
	var count int64
	s.db.Model(&models.Post{}).Where("journal_id = ?", journal.ID).Count(&count)
	resp := toJournalResponse(&journal, int(count))
	return &resp, nil
}

func (s *JournalService) Delete(id uint) error {
	var journal models.Journal
	if err := s.db.First(&journal, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJournalNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("journal_id = ?", id).Delete(&models.Post{}).Error; err != nil {
			return err
		}
		return tx.Delete(&journal).Error
	})
}

func toJournalResponse(j *models.Journal, postCount int) dto.JournalResponse {
	return dto.JournalResponse{
		ID:          j.ID,
		VaultID:     j.VaultID,
		Name:        j.Name,
		Description: ptrToStr(j.Description),
		PostCount:   postCount,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}
}
