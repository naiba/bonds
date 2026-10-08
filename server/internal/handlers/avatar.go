package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/utils"
	"github.com/naiba/bonds/pkg/avatar"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

var _ dto.VaultFileResponse

type AvatarHandler struct {
	db               *gorm.DB
	vaultFileService *services.VaultFileService
}

func NewAvatarHandler(db *gorm.DB, vaultFileService *services.VaultFileService) *AvatarHandler {
	return &AvatarHandler{db: db, vaultFileService: vaultFileService}
}

// GetAvatar godoc
//
//	@Summary		Get contact avatar
//	@Description	Return contact avatar image or generated initials
//	@Tags			contacts
//	@Produce		png
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{file}		file
//	@Failure		404			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/avatar [get]
func (h *AvatarHandler) GetAvatar(c *echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var contact models.Contact
	if err := h.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		return response.NotFound(c, "err.contact_not_found")
	}

	if contact.FileID != nil {
		if _, filePath, err := h.vaultFileService.ResolvePath(*contact.FileID, vaultID); err == nil {
			return serveLocalFile(c, filePath)
		}
	}

	nameOrder, err := services.GetUserNameOrder(h.db, userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_contact")
	}
	name := utils.FormatContactName(nameOrder, &contact, "")
	pngData := avatar.GenerateInitials(name, 128)

	return c.Blob(http.StatusOK, "image/png", pngData)
}

// UpdateAvatar godoc
//
//	@Summary		Update contact avatar
//	@Description	Upload a new avatar image for a contact
//	@Tags			contacts
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			file		formData	file	true	"Avatar image file"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/avatar [put]
func (h *AvatarHandler) UpdateAvatar(c *echo.Context) error {
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

	authorID := middleware.GetUserID(c)
	file, err := h.vaultFileService.UploadContactAvatar(contactID, vaultID, authorID, fileHeader.Filename, mimeType, fileHeader.Size, src)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_upload_file")
	}

	return response.OK(c, file)
}

// DeleteAvatar godoc
//
//	@Summary		Delete contact avatar
//	@Description	Remove the avatar from a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Success		204			"No Content"
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/avatar [delete]
func (h *AvatarHandler) DeleteAvatar(c *echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	// Use the same guarded profile write as other avatar entry points; Save
	// would upsert a source soft-deleted after the initial contact read.
	if _, err := services.NewContactAvatarService(h.db).DeleteAvatar(contactID, vaultID, middleware.GetUserID(c)); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_avatar")
	}

	return response.NoContent(c)
}
