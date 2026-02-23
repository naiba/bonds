package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrEncryptionFailed     = errors.New("encryption failed")
	ErrDecryptionFailed     = errors.New("decryption failed")
)

type DavClientService struct {
	db            *gorm.DB
	encryptionKey []byte
}

func NewDavClientService(db *gorm.DB, jwtSecret string) *DavClientService {
	hash := sha256.Sum256([]byte(jwtSecret))
	return &DavClientService{
		db:            db,
		encryptionKey: hash[:],
	}
}

func (s *DavClientService) encryptPassword(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", ErrEncryptionFailed
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrEncryptionFailed
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrEncryptionFailed
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (s *DavClientService) decryptPassword(ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", ErrDecryptionFailed
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrDecryptionFailed
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}
	return string(plaintext), nil
}

func (s *DavClientService) Create(vaultID, userID string, req dto.CreateDavSubscriptionRequest) (*dto.DavSubscriptionResponse, error) {
	encryptedPwd, err := s.encryptPassword(req.Password)
	if err != nil {
		return nil, err
	}

	syncWay := req.SyncWay
	if syncWay == 0 {
		syncWay = 2
	}
	frequency := req.Frequency
	if frequency == 0 {
		frequency = 180
	}

	sub := models.AddressBookSubscription{
		UserID:          userID,
		VaultID:         vaultID,
		URI:             req.URI,
		AddressBookPath: req.AddressBookPath,
		Username:        req.Username,
		Password:        encryptedPwd,
		SyncWay:         syncWay,
		Frequency:       frequency,
		Capabilities:    "{}",
	}
	if err := s.db.Create(&sub).Error; err != nil {
		return nil, err
	}

	resp := toDavSubscriptionResponse(&sub)
	return &resp, nil
}

func (s *DavClientService) List(vaultID string) ([]dto.DavSubscriptionResponse, error) {
	var subs []models.AddressBookSubscription
	if err := s.db.Where("vault_id = ?", vaultID).Order("created_at DESC").Find(&subs).Error; err != nil {
		return nil, err
	}
	result := make([]dto.DavSubscriptionResponse, len(subs))
	for i, sub := range subs {
		result[i] = toDavSubscriptionResponse(&sub)
	}
	return result, nil
}

func (s *DavClientService) Get(id, vaultID string) (*dto.DavSubscriptionResponse, error) {
	var sub models.AddressBookSubscription
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	resp := toDavSubscriptionResponse(&sub)
	return &resp, nil
}

func (s *DavClientService) Update(id, vaultID string, req dto.UpdateDavSubscriptionRequest) (*dto.DavSubscriptionResponse, error) {
	var sub models.AddressBookSubscription
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	if req.URI != "" {
		sub.URI = req.URI
	}
	if req.Username != "" {
		sub.Username = req.Username
	}
	if req.Password != "" {
		encryptedPwd, err := s.encryptPassword(req.Password)
		if err != nil {
			return nil, err
		}
		sub.Password = encryptedPwd
	}
	if req.SyncWay != 0 {
		sub.SyncWay = req.SyncWay
	}
	if req.Frequency != 0 {
		sub.Frequency = req.Frequency
	}
	if req.Active != nil {
		sub.Active = *req.Active
	}

	if req.AddressBookPath != "" {
		sub.AddressBookPath = req.AddressBookPath
	}
	if err := s.db.Save(&sub).Error; err != nil {
		return nil, err
	}
	resp := toDavSubscriptionResponse(&sub)
	return &resp, nil
}

func (s *DavClientService) Delete(id, vaultID string) error {
	result := s.db.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.AddressBookSubscription{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (s *DavClientService) GetDecryptedPassword(id, vaultID string) (sub *models.AddressBookSubscription, password string, err error) {
	sub = &models.AddressBookSubscription{}
	if err = s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrSubscriptionNotFound
		}
		return nil, "", err
	}
	password, err = s.decryptPassword(sub.Password)
	if err != nil {
		return nil, "", err
	}
	return sub, password, nil
}

func (s *DavClientService) ListDue() ([]models.AddressBookSubscription, error) {
	var subs []models.AddressBookSubscription
	if err := s.db.Where("active = ?", true).Find(&subs).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	due := make([]models.AddressBookSubscription, 0, len(subs))
	for _, sub := range subs {
		if sub.LastSynchronizedAt == nil {
			due = append(due, sub)
			continue
		}
		nextSync := sub.LastSynchronizedAt.Add(time.Duration(sub.Frequency) * time.Minute)
		if !now.Before(nextSync) {
			due = append(due, sub)
		}
	}
	return due, nil
}

func (s *DavClientService) UpdateSyncStatus(id string, syncToken *string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_synchronized_at": now,
	}
	if syncToken != nil {
		updates["distant_sync_token"] = *syncToken
	}
	return s.db.Model(&models.AddressBookSubscription{}).Where("id = ?", id).Updates(updates).Error
}

func toDavSubscriptionResponse(sub *models.AddressBookSubscription) dto.DavSubscriptionResponse {
	return dto.DavSubscriptionResponse{
		ID:                 sub.ID,
		VaultID:            sub.VaultID,
		URI:                sub.URI,
		Username:           sub.Username,
		AddressBookPath:    sub.AddressBookPath,
		Active:             sub.Active,
		SyncWay:            sub.SyncWay,
		Frequency:          sub.Frequency,
		LastSynchronizedAt: sub.LastSynchronizedAt,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
	}
}
