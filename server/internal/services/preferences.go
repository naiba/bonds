package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidNameOrder = errors.New("name_order must contain at least one variable like %first_name%")
	ErrInvalidWeekStart = errors.New("week_start must be sunday or monday")

	// ErrUnsupportedLocale is returned when a caller tries to persist a locale
	// code that the embedded i18n bundle does not load. Without this guard the
	// preference would silently be ignored at translation time (the UI falls
	// back to English), making the saved value look like a no-op to the user.
	ErrUnsupportedLocale = errors.New("locale is not supported")

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
		WeekStart:                 normalizedWeekStart(user.WeekStart),
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
	if !i18n.IsSupported(req.Locale) {
		return ErrUnsupportedLocale
	}
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

func ValidateNameOrder(nameOrder string) error {
	foundVar, err := validateNameOrderSegment(nameOrder, false)
	if err != nil {
		return err
	}
	if !foundVar {
		return ErrInvalidNameOrder
	}
	return nil
}

func validateNameOrderSegment(segment string, inConditional bool) (bool, error) {
	foundVar := false
	for i := 0; i < len(segment); {
		switch segment[i] {
		case '%':
			end := strings.IndexByte(segment[i+1:], '%')
			if end < 0 {
				return false, fmt.Errorf("%w: unmatched %% in name_order", ErrInvalidNameOrder)
			}
			varName := segment[i+1 : i+1+end]
			if !validNameOrderVars[varName] {
				return false, fmt.Errorf("%w: unknown variable in name_order: %%%s%%", ErrInvalidNameOrder, varName)
			}
			foundVar = true
			i += end + 2
		case '{':
			if inConditional {
				return false, fmt.Errorf("%w: nested conditional blocks are not allowed", ErrInvalidNameOrder)
			}
			end := strings.IndexByte(segment[i+1:], '}')
			if end < 0 {
				return false, fmt.Errorf("%w: unclosed conditional block", ErrInvalidNameOrder)
			}
			block := segment[i+1 : i+1+end]
			blockFoundVar, err := validateNameOrderConditional(block)
			if err != nil {
				return false, err
			}
			foundVar = foundVar || blockFoundVar
			i += end + 2
		case '}':
			return false, fmt.Errorf("%w: unopened conditional block", ErrInvalidNameOrder)
		default:
			i++
		}
	}
	return foundVar, nil
}

func validateNameOrderConditional(block string) (bool, error) {
	conditionField, template, ok := strings.Cut(block, "?")
	if !ok || conditionField == "" || template == "" {
		return false, fmt.Errorf("%w: malformed conditional block", ErrInvalidNameOrder)
	}
	if strings.ContainsAny(conditionField, "{}% ") {
		return false, fmt.Errorf("%w: malformed conditional condition", ErrInvalidNameOrder)
	}
	if !validNameOrderVars[conditionField] {
		return false, fmt.Errorf("%w: unknown conditional field: %s", ErrInvalidNameOrder, conditionField)
	}
	return validateNameOrderSegment(template, true)
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
	if req.WeekStart != "" {
		if !isValidWeekStart(req.WeekStart) {
			return nil, ErrInvalidWeekStart
		}
		updates["week_start"] = req.WeekStart
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	if req.Locale != "" {
		if !i18n.IsSupported(req.Locale) {
			return nil, ErrUnsupportedLocale
		}
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

func isValidWeekStart(weekStart string) bool {
	return weekStart == "sunday" || weekStart == "monday"
}

func normalizedWeekStart(weekStart string) string {
	if isValidWeekStart(weekStart) {
		return weekStart
	}
	return "sunday"
}
