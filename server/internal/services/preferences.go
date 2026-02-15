package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type PreferenceService struct {
	db *gorm.DB
}

func NewPreferenceService(db *gorm.DB) *PreferenceService {
	return &PreferenceService{db: db}
}

func (s *PreferenceService) Get(userID string) (*dto.PreferencesResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	tz := ""
	if user.Timezone != nil {
		tz = *user.Timezone
	}
	return &dto.PreferencesResponse{
		NameOrder:      user.NameOrder,
		DateFormat:     user.DateFormat,
		Timezone:       tz,
		Locale:         user.Locale,
		NumberFormat:   user.NumberFormat,
		DistanceFormat: user.DistanceFormat,
		DefaultMapSite: user.DefaultMapSite,
		HelpShown:      user.HelpShown,
	}, nil
}

func (s *PreferenceService) UpdateNameOrder(userID string, req dto.UpdateNameOrderRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("name_order", req.NameOrder).Error
}

func (s *PreferenceService) UpdateDateFormat(userID string, req dto.UpdateDateFormatRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("date_format", req.DateFormat).Error
}

func (s *PreferenceService) UpdateTimezone(userID string, req dto.UpdateTimezoneRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("timezone", req.Timezone).Error
}

func (s *PreferenceService) UpdateLocale(userID string, req dto.UpdateLocaleRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("locale", req.Locale).Error
}

func (s *PreferenceService) UpdateNumberFormat(userID string, req dto.UpdateNumberFormatRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("number_format", req.NumberFormat).Error
}

func (s *PreferenceService) UpdateDistanceFormat(userID string, req dto.UpdateDistanceFormatRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("distance_format", req.DistanceFormat).Error
}

func (s *PreferenceService) UpdateMapsPreference(userID string, req dto.UpdateMapsPreferenceRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("default_map_site", req.DefaultMapSite).Error
}

func (s *PreferenceService) UpdateHelpShown(userID string, req dto.UpdateHelpShownRequest) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("help_shown", req.HelpShown).Error
}

func (s *PreferenceService) UpdateAll(userID string, req dto.UpdatePreferencesRequest) (*dto.PreferencesResponse, error) {
	updates := map[string]interface{}{}
	if req.NameOrder != "" {
		updates["name_order"] = req.NameOrder
	}
	if req.DateFormat != "" {
		updates["date_format"] = req.DateFormat
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if req.Locale != "" {
		updates["locale"] = req.Locale
	}
	if req.NumberFormat != "" {
		updates["number_format"] = req.NumberFormat
	}
	if req.DistanceFormat != "" {
		updates["distance_format"] = req.DistanceFormat
	}
	if req.DefaultMapSite != "" {
		updates["default_map_site"] = req.DefaultMapSite
	}
	if req.HelpShown != nil {
		updates["help_shown"] = *req.HelpShown
	}
	if len(updates) == 0 {
		return s.Get(userID)
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(userID)
}
