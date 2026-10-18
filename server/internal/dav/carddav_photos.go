package dav

import (
	"bytes"
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

// Inline photos are materialized and base64 encoded for each vCard. Keep a
// fixed 10 MiB ceiling even when administrators allow larger stored uploads.
const maxCardDAVPhotoBytes = 10 << 20

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
	if !info.Mode().IsRegular() || info.Size() > maxCardDAVPhotoBytes {
		return nil
	}
	data, imageType, err := readCardDAVPhoto(file)
	if err != nil {
		return fmt.Errorf("read CardDAV photo: %w", err)
	}
	if imageType == "" {
		return nil
	}
	card.Set(vcard.FieldPhoto, &vcard.Field{
		Value:  base64.StdEncoding.EncodeToString(data),
		Params: vcard.Params{"ENCODING": {"b"}, vcard.ParamType: {imageType}},
	})
	return nil
}

func readCardDAVPhoto(reader io.Reader) ([]byte, string, error) {
	// Upload metadata is not proof of image content. Sniff before allocating the
	// body, and bound the actual read as well as Stat: a file can grow after Stat.
	// Otherwise even a metadata-only PROPFIND can exhaust memory on one avatar.
	header := make([]byte, 512)
	n, err := io.ReadFull(reader, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", err
	}
	header = header[:n]
	var imageType string
	switch http.DetectContentType(header) {
	case "image/jpeg":
		imageType = "JPEG"
	case "image/png":
		imageType = "PNG"
	case "image/gif":
		imageType = "GIF"
	case "image/webp":
		imageType = "WEBP"
	default:
		return nil, "", nil
	}
	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(header), io.LimitReader(reader, maxCardDAVPhotoBytes+1-int64(n))))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxCardDAVPhotoBytes {
		return nil, "", nil
	}
	return data, imageType, nil
}
