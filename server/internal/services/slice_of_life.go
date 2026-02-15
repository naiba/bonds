package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrSliceOfLifeNotFound = errors.New("slice of life not found")

type SliceOfLifeService struct {
	db *gorm.DB
}

func NewSliceOfLifeService(db *gorm.DB) *SliceOfLifeService {
	return &SliceOfLifeService{db: db}
}

func (s *SliceOfLifeService) List(journalID uint, vaultID string) ([]dto.SliceOfLifeResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var slices []models.SliceOfLife
	if err := s.db.Where("journal_id = ?", journalID).Order("created_at DESC").Find(&slices).Error; err != nil {
		return nil, err
	}
	result := make([]dto.SliceOfLifeResponse, len(slices))
	for i, sl := range slices {
		result[i] = toSliceOfLifeResponse(&sl)
	}
	return result, nil
}

func (s *SliceOfLifeService) Create(journalID uint, vaultID string, req dto.CreateSliceOfLifeRequest) (*dto.SliceOfLifeResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	slice := models.SliceOfLife{
		JournalID:   journalID,
		Name:        req.Name,
		Description: strPtrOrNil(req.Description),
	}
	if err := s.db.Create(&slice).Error; err != nil {
		return nil, err
	}
	resp := toSliceOfLifeResponse(&slice)
	return &resp, nil
}

func (s *SliceOfLifeService) Get(id uint, journalID uint, vaultID string) (*dto.SliceOfLifeResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var slice models.SliceOfLife
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&slice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSliceOfLifeNotFound
		}
		return nil, err
	}
	resp := toSliceOfLifeResponse(&slice)
	return &resp, nil
}

func (s *SliceOfLifeService) Update(id uint, journalID uint, vaultID string, req dto.UpdateSliceOfLifeRequest) (*dto.SliceOfLifeResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var slice models.SliceOfLife
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&slice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSliceOfLifeNotFound
		}
		return nil, err
	}
	slice.Name = req.Name
	slice.Description = strPtrOrNil(req.Description)
	if err := s.db.Save(&slice).Error; err != nil {
		return nil, err
	}
	resp := toSliceOfLifeResponse(&slice)
	return &resp, nil
}

func (s *SliceOfLifeService) Delete(id uint, journalID uint, vaultID string) error {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJournalNotFound
		}
		return err
	}
	result := s.db.Where("id = ? AND journal_id = ?", id, journalID).Delete(&models.SliceOfLife{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSliceOfLifeNotFound
	}
	return nil
}

func (s *SliceOfLifeService) UpdateCover(id uint, journalID uint, vaultID string, fileID uint) (*dto.SliceOfLifeResponse, error) {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJournalNotFound
		}
		return nil, err
	}
	var slice models.SliceOfLife
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&slice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSliceOfLifeNotFound
		}
		return nil, err
	}
	slice.FileCoverImageID = &fileID
	if err := s.db.Save(&slice).Error; err != nil {
		return nil, err
	}
	resp := toSliceOfLifeResponse(&slice)
	return &resp, nil
}

func (s *SliceOfLifeService) RemoveCover(id uint, journalID uint, vaultID string) error {
	var journal models.Journal
	if err := s.db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJournalNotFound
		}
		return err
	}
	var slice models.SliceOfLife
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&slice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSliceOfLifeNotFound
		}
		return err
	}
	return s.db.Model(&slice).Update("file_cover_image_id", nil).Error
}

func toSliceOfLifeResponse(sl *models.SliceOfLife) dto.SliceOfLifeResponse {
	return dto.SliceOfLifeResponse{
		ID:               sl.ID,
		JournalID:        sl.JournalID,
		Name:             sl.Name,
		Description:      ptrToStr(sl.Description),
		FileCoverImageID: sl.FileCoverImageID,
		CreatedAt:        sl.CreatedAt,
		UpdatedAt:        sl.UpdatedAt,
	}
}
