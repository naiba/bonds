package services

import (
	"errors"

	"github.com/naiba/bonds/internal/models"
)

var ErrOAuthTokenNotFound = errors.New("oauth token not found")

func (s *OAuthService) ListProviders(userID string) ([]map[string]string, error) {
	var tokens []models.UserToken
	if err := s.db.Where("user_id = ?", userID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	result := make([]map[string]string, len(tokens))
	for i, t := range tokens {
		result[i] = map[string]string{
			"driver":    t.Driver,
			"driver_id": t.DriverID,
		}
	}
	return result, nil
}

func (s *OAuthService) UnlinkProvider(userID, driver string) error {
	result := s.db.Where("user_id = ? AND driver = ?", userID, driver).Delete(&models.UserToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOAuthTokenNotFound
	}
	return nil
}
