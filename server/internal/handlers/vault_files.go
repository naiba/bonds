package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultFileResponse

var allowedMimeTypes = map[string]bool{
	"image/jpeg":         true,
	"image/png":          true,
	"image/gif":          true,
	"image/webp":         true,
	"video/mp4":          true,
	"video/webm":         true,
	"video/ogg":          true,
	"video/quicktime":    true,
	"application/pdf":    true,
	"text/plain":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

// maxUploadSize default is 10MB, overridden by storage.max_size_mb system setting (MB) in handleUpload()

type VaultFileHandler struct {
	vaultFileService   *services.VaultFileService
	storageInfoService *services.StorageInfoService
	settingsService    *services.SystemSettingService
}

func NewVaultFileHandler(vaultFileService *services.VaultFileService, storageInfoService *services.StorageInfoService, settingsService *services.SystemSettingService) *VaultFileHandler {
	return &VaultFileHandler{vaultFileService: vaultFileService, storageInfoService: storageInfoService, settingsService: settingsService}
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
		if errors.Is(err, services.ErrFileInUse) {
			return response.BadRequest(c, "err.file_referenced_by_quick_fact", nil)
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
//	@Param			file_type	formData	string	false	"File type (document/photo/video)"
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
//	@Description	Upload media or document for a contact
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
		fileType = "media"
	}

	return h.handleUpload(c, vaultID, contactID, fileType)
}

func contactMediaFileType(mimeType string) (string, bool) {
	if strings.HasPrefix(mimeType, "image/") {
		return "photo", true
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video", true
	}
	return "", false
}

func uploadFileTypeMatchesMime(fileType, mimeType string) bool {
	switch fileType {
	case "media":
		_, ok := contactMediaFileType(mimeType)
		return ok
	case "photo", "avatar":
		return strings.HasPrefix(mimeType, "image/")
	case "video":
		return strings.HasPrefix(mimeType, "video/")
	default:
		return !strings.HasPrefix(mimeType, "video/")
	}
}

func uploadMediaMimeMatches(mimeType string, sniffedData []byte) bool {
	detectedMimeType := http.DetectContentType(sniffedData)
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4", "video/webm":
		return detectedMimeType == mimeType
	case "video/ogg":
		return detectedMimeType == "application/ogg" || detectedMimeType == "video/ogg"
	case "video/quicktime":
		return len(sniffedData) >= 12 && bytes.Equal(sniffedData[4:8], []byte("ftyp")) && bytes.Equal(sniffedData[8:12], []byte("qt  "))
	default:
		return false
	}
}

func verifyUploadedMediaContent(src interface {
	io.Reader
	io.Seeker
}, mimeType string) (bool, error) {
	var sniffBuffer [512]byte
	n, err := src.Read(sniffBuffer[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	return uploadMediaMimeMatches(mimeType, sniffBuffer[:n]), nil
}

func (h *VaultFileHandler) handleUpload(c echo.Context, vaultID, contactID, fileType string) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "err.file_required", nil)
	}

	var maxUploadSize int64 = 10 * 1024 * 1024
	if h.settingsService != nil {
		// storage.max_size_mb 存储的是 MB，需要转换为字节进行比较
		maxSizeMB := h.settingsService.GetInt64("storage.max_size_mb", 0)
		if maxSizeMB > 0 {
			maxUploadSize = maxSizeMB * 1024 * 1024
		}
	}

	if fileHeader.Size > maxUploadSize {
		return response.BadRequest(c, "err.file_too_large", map[string]string{
			"max_size": fmt.Sprintf("%d MB", maxUploadSize/(1024*1024)),
		})
	}

	// 检查账户存储配额。limit_bytes=0 表示无限制。
	accountID := middleware.GetAccountID(c)
	storageInfo, err := h.storageInfoService.Get(accountID)
	if err == nil && storageInfo.LimitBytes > 0 {
		if storageInfo.UsedBytes+fileHeader.Size > storageInfo.LimitBytes {
			return response.BadRequest(c, "err.storage_quota_exceeded", nil)
		}
	}

	src, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[mimeType] {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}
	if !uploadFileTypeMatchesMime(fileType, mimeType) {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}
	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") {
		// Multipart Content-Type is user controlled; validate inline-previewable media by bytes.
		validMedia, err := verifyUploadedMediaContent(src, mimeType)
		if err != nil {
			return response.InternalError(c, "err.failed_to_read_file")
		}
		if !validMedia {
			return response.BadRequest(c, "err.file_type_not_allowed", nil)
		}
	}
	if fileType == "media" {
		inferredFileType, ok := contactMediaFileType(mimeType)
		if !ok {
			return response.BadRequest(c, "err.file_type_not_allowed", nil)
		}
		fileType = inferredFileType
	}

	authorID := middleware.GetUserID(c)
	result, err := h.vaultFileService.Upload(vaultID, contactID, authorID, fileType, fileHeader.Filename, mimeType, fileHeader.Size, src)
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
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	if c.QueryParam("preview") == "true" && (strings.HasPrefix(file.MimeType, "image/") || strings.HasPrefix(file.MimeType, "video/")) {
		c.Response().Header().Set(echo.HeaderContentType, file.MimeType)
		return c.Inline(filePath, file.Name)
	}
	return c.Attachment(filePath, file.Name)
}
