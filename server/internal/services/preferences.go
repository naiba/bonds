package services

import (
	"errors"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidNameOrder = errors.New("name_order must contain at least one variable like %first_name%")

	// validNameOrderVars lists all allowed template variables for name_order.
	validNameOrderVars = map[string]bool{
		"first_name":  true,
		"last_name":   true,
		"middle_name": true,
		"nickname":    true,
		"maiden_name": true,
	}
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
		NameOrder:                 user.NameOrder,
		DateFormat:                user.DateFormat,
		Timezone:                  tz,
		Locale:                    user.Locale,
		NumberFormat:              user.NumberFormat,
		DistanceFormat:            user.DistanceFormat,
		DefaultMapSite:            user.DefaultMapSite,
		HelpShown:                 user.HelpShown,
		EnableAlternativeCalendar: user.EnableAlternativeCalendar,
	}, nil
}

func (s *PreferenceService) UpdateNameOrder(userID string, req dto.UpdateNameOrderRequest) error {
	if err := ValidateNameOrder(req.NameOrder); err != nil {
		return err
	}
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

// ValidateNameOrder checks that a name_order template string contains at least
// one valid %variable% and all %...% tokens reference known fields.
func ValidateNameOrder(nameOrder string) error {
	if strings.Count(nameOrder, "%") < 2 {
		return ErrInvalidNameOrder
	}
	if strings.Count(nameOrder, "%")%2 != 0 {
		return ErrInvalidNameOrder
	}
	// Extract variables between % pairs
	foundVar := false
	parts := strings.Split(nameOrder, "%")
	// parts[0] is before first %, parts[1] is var name, parts[2] is between, etc.
	for i := 1; i < len(parts); i += 2 {
		varName := parts[i]
		if !validNameOrderVars[varName] {
			return errors.New("unknown variable in name_order: %" + varName + "%")
		}
		foundVar = true
	}
	if !foundVar {
		return ErrInvalidNameOrder
	}
	return nil
}

func (s *PreferenceService) UpdateAll(userID string, req dto.UpdatePreferencesRequest) (*dto.PreferencesResponse, error) {
	updates := map[string]interface{}{}
	if req.NameOrder != "" {
		if err := ValidateNameOrder(req.NameOrder); err != nil {
			return nil, err
		}
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
	if req.EnableAlternativeCalendar != nil {
		updates["enable_alternative_calendar"] = *req.EnableAlternativeCalendar
	}
	if len(updates) == 0 {
		return s.Get(userID)
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Get(userID)
}
