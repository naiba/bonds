package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/avatar"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

type AvatarHandler struct {
	db               *gorm.DB
	vaultFileService *services.VaultFileService
}

func NewAvatarHandler(db *gorm.DB, vaultFileService *services.VaultFileService) *AvatarHandler {
	return &AvatarHandler{db: db, vaultFileService: vaultFileService}
}

func (h *AvatarHandler) GetAvatar(c echo.Context) error {
	contactID := c.Param("contact_id")

	var contact models.Contact
	if err := h.db.First(&contact, "id = ?", contactID).Error; err != nil {
		return response.NotFound(c, "err.contact_not_found")
	}

	if contact.FileID != nil {
		var file models.File
		if err := h.db.First(&file, *contact.FileID).Error; err == nil {
			filePath := filepath.Join(h.vaultFileService.UploadDir(), file.UUID)
			return c.File(filePath)
		}
	}

	name := buildContactName(&contact)
	pngData := avatar.GenerateInitials(name, 128)

	return c.Blob(http.StatusOK, "image/png", pngData)
}

func (h *AvatarHandler) UpdateAvatar(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var contact models.Contact
	if err := h.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		return response.NotFound(c, "err.contact_not_found")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "err.file_required", nil)
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" && mimeType != "image/webp" {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	file, err := h.vaultFileService.Upload(vaultID, contactID, "avatar", fileHeader.Filename, mimeType, fileHeader.Size, src)
	if err != nil {
		return response.InternalError(c, "err.failed_to_upload_file")
	}

	contact.FileID = &file.ID
	if err := h.db.Save(&contact).Error; err != nil {
		return response.InternalError(c, "err.failed_to_update_avatar")
	}

	return response.OK(c, file)
}

func (h *AvatarHandler) DeleteAvatar(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var contact models.Contact
	if err := h.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		return response.NotFound(c, "err.contact_not_found")
	}

	contact.FileID = nil
	if err := h.db.Save(&contact).Error; err != nil {
		return response.InternalError(c, "err.failed_to_delete_avatar")
	}

	return response.NoContent(c)
}

func buildContactName(c *models.Contact) string {
	var parts []string
	if c.FirstName != nil && *c.FirstName != "" {
		parts = append(parts, *c.FirstName)
	}
	if c.LastName != nil && *c.LastName != "" {
		parts = append(parts, *c.LastName)
	}
	if len(parts) == 0 && c.Nickname != nil && *c.Nickname != "" {
		parts = append(parts, *c.Nickname)
	}
	return strings.Join(parts, " ")
}
