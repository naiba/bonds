package dav

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
)

func (b *CardDAVBackend) addContactPhoto(card vcard.Card, contact *models.Contact) error {
	if contact.File == nil || b.uploadDir == "" {
		return nil
	}
	// Uploaded avatars use local UUID paths, which the URL-only vCard builder
	// intentionally omits. Embed their bytes: DAV clients cannot use JWT-only
	// download URLs. Do this before hashing the card so photo changes affect ETags.
	_, path, err := services.NewVaultFileService(b.db, b.uploadDir).ResolvePath(contact.File.ID, contact.VaultID)
	if errors.Is(err, services.ErrFileNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	rootPath, err := filepath.Abs(b.uploadDir)
	if err != nil {
		return err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relativePath, err := filepath.Rel(rootPath, absolutePath)
	if err != nil || !filepath.IsLocal(relativePath) {
		return nil
	}
	file, err := openCardDAVPhoto(rootPath, relativePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open CardDAV photo: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read CardDAV photo: %w", err)
	}
	var imageType string
	switch http.DetectContentType(data) {
	case "image/jpeg":
		imageType = "JPEG"
	case "image/png":
		imageType = "PNG"
	case "image/gif":
		imageType = "GIF"
	case "image/webp":
		imageType = "WEBP"
	default:
		return nil
	}
	card.Set(vcard.FieldPhoto, &vcard.Field{
		Value:  base64.StdEncoding.EncodeToString(data),
		Params: vcard.Params{"ENCODING": {"b"}, vcard.ParamType: {imageType}},
	})
	return nil
}
