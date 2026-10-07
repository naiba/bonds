package services

import (
	"errors"
	"io"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type ContactAvatarService struct {
	db *gorm.DB
}

func NewContactAvatarService(db *gorm.DB) *ContactAvatarService {
	return &ContactAvatarService{db: db}
}

func (s *ContactAvatarService) UpdateAvatar(contactID, vaultID, userID string, fileID uint) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.FileID = &fileID
	if err := updateContactProfile(s.db, &contact, vaultID, "file_id"); err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactAvatarService) DeleteAvatar(contactID, vaultID, userID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.FileID = nil
	if err := updateContactProfile(s.db, &contact, vaultID, "file_id"); err != nil {
		return nil, err
	}
	if err := reloadContactWithSameVaultFirstMetThrough(s.db, &contact, vaultID); err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	resp, err := toContactResponse(&contact, false, formatter)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadContactAvatar commits the file and profile reference together. The
// HTTP route must not Save an earlier contact snapshot after upload: a merge
// may have deleted that source or filled the survivor's other profile fields.
func (s *VaultFileService) UploadContactAvatar(contactID, vaultID, userID, filename, mimeType string, size int64, data io.Reader) (*dto.VaultFileResponse, error) {
	return s.upload(vaultID, contactID, userID, "avatar", filename, mimeType, size, data, func(tx *gorm.DB, file *models.File) error {
		contact := models.Contact{ID: contactID, FileID: &file.ID}
		return updateContactProfile(tx, &contact, vaultID, "file_id")
	})
}
