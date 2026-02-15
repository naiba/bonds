package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type VaultMoodParamService struct {
	db *gorm.DB
}

func NewVaultMoodParamService(db *gorm.DB) *VaultMoodParamService {
	return &VaultMoodParamService{db: db}
}

func (s *VaultMoodParamService) List(vaultID string) ([]dto.MoodTrackingParameterResponse, error) {
	var params []models.MoodTrackingParameter
	if err := s.db.Where("vault_id = ?", vaultID).Order("position ASC").Find(&params).Error; err != nil {
		return nil, err
	}
	result := make([]dto.MoodTrackingParameterResponse, len(params))
	for i, p := range params {
		result[i] = toMoodParamResponse(&p)
	}
	return result, nil
}

func (s *VaultMoodParamService) Create(vaultID string, req dto.CreateMoodTrackingParameterRequest) (*dto.MoodTrackingParameterResponse, error) {
	label := req.Label
	param := models.MoodTrackingParameter{
		VaultID:  vaultID,
		Label:    &label,
		HexColor: req.HexColor,
		Position: req.Position,
	}
	if err := s.db.Create(&param).Error; err != nil {
		return nil, err
	}
	resp := toMoodParamResponse(&param)
	return &resp, nil
}

func (s *VaultMoodParamService) Update(id uint, vaultID string, req dto.UpdateMoodTrackingParameterRequest) (*dto.MoodTrackingParameterResponse, error) {
	var param models.MoodTrackingParameter
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&param).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMoodParamNotFound
		}
		return nil, err
	}
	label := req.Label
	param.Label = &label
	param.HexColor = req.HexColor
	param.Position = req.Position
	if err := s.db.Save(&param).Error; err != nil {
		return nil, err
	}
	resp := toMoodParamResponse(&param)
	return &resp, nil
}

func (s *VaultMoodParamService) UpdatePosition(id uint, vaultID string, position int) (*dto.MoodTrackingParameterResponse, error) {
	var param models.MoodTrackingParameter
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&param).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMoodParamNotFound
		}
		return nil, err
	}
	param.Position = &position
	if err := s.db.Save(&param).Error; err != nil {
		return nil, err
	}
	resp := toMoodParamResponse(&param)
	return &resp, nil
}

func (s *VaultMoodParamService) Delete(id uint, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.MoodTrackingParameter{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMoodParamNotFound
	}
	return nil
}

func toMoodParamResponse(p *models.MoodTrackingParameter) dto.MoodTrackingParameterResponse {
	return dto.MoodTrackingParameterResponse{
		ID:        p.ID,
		Label:     ptrToStr(p.Label),
		HexColor:  p.HexColor,
		Position:  p.Position,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
