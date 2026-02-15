package handlers

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

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

func (h *VaultFileHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	files, err := h.vaultFileService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_files")
	}
	return response.OK(c, files)
}

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

func (h *VaultFileHandler) Upload(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.FormValue("contact_id")
	fileType := c.FormValue("file_type")
	if fileType == "" {
		fileType = "document"
	}

	return h.handleUpload(c, vaultID, contactID, fileType)
}

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
