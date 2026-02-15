package handlers

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultFileResponse

var allowedMimeTypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"image/webp":         true,
	"application/pdf":    true,
	"text/plain":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

const maxUploadSize = 10 * 1024 * 1024

type VaultFileHandler struct {
	vaultFileService *services.VaultFileService
}

func NewVaultFileHandler(vaultFileService *services.VaultFileService) *VaultFileHandler {
	return &VaultFileHandler{vaultFileService: vaultFileService}
}

// List godoc
//
//	@Summary		List vault files
//	@Description	Return all files for a vault
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files [get]
func (h *VaultFileHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	files, err := h.vaultFileService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.OK(c, files)
}

// Delete godoc
//
//	@Summary		Delete a vault file
//	@Description	Delete a file by ID
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"File ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files/{id} [delete]
func (h *VaultFileHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_file_id", nil)
	}
	if err := h.vaultFileService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.file_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_file")
	}
	return response.NoContent(c)
}

// Upload godoc
//
//	@Summary		Upload a file
//	@Description	Upload a file to a vault
//	@Tags			files
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			file		formData	file	true	"File to upload"
//	@Param			contact_id	formData	string	false	"Contact ID"
//	@Param			file_type	formData	string	false	"File type (document/photo)"
//	@Success		201			{object}	response.APIResponse{data=dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files [post]
func (h *VaultFileHandler) Upload(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.FormValue("contact_id")
	fileType := c.FormValue("file_type")
	if fileType == "" {
		fileType = "document"
	}

	return h.handleUpload(c, vaultID, contactID, fileType)
}

// UploadContactFile godoc
//
//	@Summary		Upload a contact file
//	@Description	Upload a photo or document for a contact
//	@Tags			files
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			file		formData	file	true	"File to upload"
//	@Success		201			{object}	response.APIResponse{data=dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/photos [post]
func (h *VaultFileHandler) UploadContactFile(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")

	fileType := "document"
	if strings.HasSuffix(c.Path(), "photos") {
		fileType = "photo"
	}

	return h.handleUpload(c, vaultID, contactID, fileType)
}

func (h *VaultFileHandler) handleUpload(c echo.Context, vaultID, contactID, fileType string) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "err.file_required", nil)
	}

	if fileHeader.Size > maxUploadSize {
		return response.BadRequest(c, "err.file_too_large", map[string]string{
			"max_size": fmt.Sprintf("%d MB", maxUploadSize/(1024*1024)),
		})
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[mimeType] {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	result, err := h.vaultFileService.Upload(vaultID, contactID, fileType, fileHeader.Filename, mimeType, fileHeader.Size, src)
	if err != nil {
		return response.InternalError(c, "err.failed_to_upload_file")
	}

	return response.Created(c, result)
}

// Serve godoc
//
//	@Summary		Download a file
//	@Description	Download a file as attachment
//	@Tags			files
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"File ID"
//	@Success		200			{file}		file
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/files/{id}/download [get]
func (h *VaultFileHandler) Serve(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_file_id", nil)
	}

	file, err := h.vaultFileService.Get(uint(id), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.file_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_file")
	}

	filePath := filepath.Join(h.vaultFileService.UploadDir(), file.UUID)
	return c.Attachment(filePath, file.Name)
}
