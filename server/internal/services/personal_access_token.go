package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

const (
	tokenPrefix = "bonds_"
	tokenLength = 40
)

var (
	ErrTokenNotFound = errors.New("personal access token not found")
	ErrTokenExpired  = errors.New("personal access token expired")
	ErrTokenNameDuplicate = errors.New("token name already exists")
)

type PersonalAccessTokenService struct {
	db *gorm.DB
}

func NewPersonalAccessTokenService(db *gorm.DB) *PersonalAccessTokenService {
	return &PersonalAccessTokenService{db: db}
}

func (s *PersonalAccessTokenService) Create(userID, accountID string, req dto.CreatePersonalAccessTokenRequest) (*dto.PersonalAccessTokenCreatedResponse, error) {
	// Check duplicate name for same user
	var count int64
	s.db.Model(&models.PersonalAccessToken{}).Where("user_id = ? AND name = ?", userID, req.Name).Count(&count)
	if count > 0 {
		return nil, ErrTokenNameDuplicate
	}

	// Generate random token
	rawToken, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash token for storage (SHA-256)
	hash := hashToken(rawToken)

	// Create hint (last 6 chars)
	hint := "..." + rawToken[len(rawToken)-6:]

	token := models.PersonalAccessToken{
		UserID:    userID,
		AccountID: accountID,
		Name:      req.Name,
		TokenHash: hash,
		TokenHint: hint,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.db.Create(&token).Error; err != nil {
		return nil, err
	}

	return &dto.PersonalAccessTokenCreatedResponse{
		ID:        token.ID,
		Name:      token.Name,
		Token:     rawToken,
		TokenHint: hint,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (s *PersonalAccessTokenService) List(userID string) ([]dto.PersonalAccessTokenResponse, error) {
	var tokens []models.PersonalAccessToken
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PersonalAccessTokenResponse, len(tokens))
	for i, t := range tokens {
		result[i] = toPersonalAccessTokenResponse(&t)
	}
	return result, nil
}

func (s *PersonalAccessTokenService) Delete(id uint, userID string) error {
	var token models.PersonalAccessToken
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTokenNotFound
		}
		return err
	}
	return s.db.Delete(&token).Error
}

// ValidateToken looks up a token by its raw value, validates expiration,
// updates last_used_at, and returns the associated user info.
func (s *PersonalAccessTokenService) ValidateToken(rawToken string) (*models.PersonalAccessToken, error) {
	hash := hashToken(rawToken)

	var token models.PersonalAccessToken
	if err := s.db.Where("token_hash = ?", hash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Update last_used_at asynchronously (fire and forget)
	now := time.Now()
	s.db.Model(&token).Update("last_used_at", &now)

	return &token, nil
}

func toPersonalAccessTokenResponse(t *models.PersonalAccessToken) dto.PersonalAccessTokenResponse {
	return dto.PersonalAccessTokenResponse{
		ID:         t.ID,
		Name:       t.Name,
		TokenHint:  t.TokenHint,
		ExpiresAt:  t.ExpiresAt,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
	}
}

func generateRandomToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + hex.EncodeToString(b), nil
}

func hashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}