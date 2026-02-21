package services

import (
	"errors"

	"github.com/markbates/goth"
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

func (s *OAuthService) ListAvailableProviders() []map[string]string {
	result := []map[string]string{}
	for _, p := range goth.GetProviders() {
		entry := map[string]string{
			"name": p.Name(),
		}
		var op models.OAuthProvider
		if err := s.db.Where("name = ?", p.Name()).First(&op).Error; err == nil && op.DisplayName != "" {
			entry["display_name"] = op.DisplayName
		}
		result = append(result, entry)
	}
	return result
}
