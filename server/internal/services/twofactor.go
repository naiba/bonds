package services

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

var (
	ErrInvalidTOTPCode  = errors.New("invalid totp code")
	ErrTwoFactorNotSet  = errors.New("two factor not set up")
	ErrTwoFactorPending = errors.New("two factor authentication required")
)

type TwoFactorService struct {
	db *gorm.DB
}

func NewTwoFactorService(db *gorm.DB) *TwoFactorService {
	return &TwoFactorService{db: db}
}

func (s *TwoFactorService) Enable(userID string) (*dto.TwoFactorSetupResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bonds",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	secret := key.Secret()
	recoveryCodes, err := generateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}

	codesJSON, err := json.Marshal(recoveryCodes)
	if err != nil {
		return nil, err
	}
	codesStr := string(codesJSON)

	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"two_factor_secret":         secret,
		"two_factor_recovery_codes": codesStr,
		"two_factor_confirmed_at":   nil,
	}).Error; err != nil {
		return nil, err
	}

	return &dto.TwoFactorSetupResponse{
		Secret:        secret,
		QRCodeURL:     key.URL(),
		RecoveryCodes: recoveryCodes,
	}, nil
}

func (s *TwoFactorService) Confirm(userID string, code string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if user.TwoFactorSecret == nil {
		return ErrTwoFactorNotSet
	}

	if !totp.Validate(code, *user.TwoFactorSecret) {
		return ErrInvalidTOTPCode
	}

	now := time.Now()
	return s.db.Model(&user).Update("two_factor_confirmed_at", now).Error
}

func (s *TwoFactorService) Disable(userID string, code string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if user.TwoFactorSecret == nil {
		return ErrTwoFactorNotSet
	}

	if !totp.Validate(code, *user.TwoFactorSecret) {
		return ErrInvalidTOTPCode
	}

	return s.db.Model(&user).Updates(map[string]interface{}{
		"two_factor_secret":         nil,
		"two_factor_recovery_codes": nil,
		"two_factor_confirmed_at":   nil,
	}).Error
}

func (s *TwoFactorService) Validate(userID string, code string) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrUserNotFound
		}
		return false, err
	}

	if user.TwoFactorConfirmedAt == nil || user.TwoFactorSecret == nil {
		return false, ErrTwoFactorNotSet
	}

	if totp.Validate(code, *user.TwoFactorSecret) {
		return true, nil
	}

	if user.TwoFactorRecoveryCodes == nil {
		return false, nil
	}

	var codes []string
	if err := json.Unmarshal([]byte(*user.TwoFactorRecoveryCodes), &codes); err != nil {
		return false, err
	}

	for i, c := range codes {
		if c == code {
			codes = append(codes[:i], codes[i+1:]...)
			codesJSON, err := json.Marshal(codes)
			if err != nil {
				return false, err
			}
			codesStr := string(codesJSON)
			if err := s.db.Model(&user).Update("two_factor_recovery_codes", codesStr).Error; err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

func (s *TwoFactorService) IsEnabled(userID string) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrUserNotFound
		}
		return false, err
	}
	return user.TwoFactorConfirmedAt != nil, nil
}

func generateRecoveryCodes(count int) ([]string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code := make([]byte, 8)
		for j := range code {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			if err != nil {
				return nil, err
			}
			code[j] = charset[idx.Int64()]
		}
		codes[i] = string(code)
	}
	return codes, nil
}
