package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultFileResponse

type PostPhotoHandler struct {
	vaultFileService   *services.VaultFileService
	storageInfoService *services.StorageInfoService
	settingsService    *services.SystemSettingService
}

func NewPostPhotoHandler(vaultFileService *services.VaultFileService, storageInfoService *services.StorageInfoService, settingsService *services.SystemSettingService) *PostPhotoHandler {
	return &PostPhotoHandler{vaultFileService: vaultFileService, storageInfoService: storageInfoService, settingsService: settingsService}
}

// List godoc
//
//	@Summary		List post photos
//	@Description	Return all photos for a post
//	@Tags			post-photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Param			id			path		integer	true	"Post ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/photos [get]
func (h *PostPhotoHandler) List(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	files, err := h.vaultFileService.ListPostPhotos(uint(postID), vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_post_photos")
	}
	return response.OK(c, files)
}

// Upload godoc
//
//	@Summary		Upload a post photo
//	@Description	Upload a photo to a post
//	@Tags			post-photos
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			journal_id	path		integer	true	"Journal ID"
//	@Param			id			path		integer	true	"Post ID"
//	@Param			file		formData	file	true	"Photo file"
//	@Success		201			{object}	response.APIResponse{data=dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/photos [post]
func (h *PostPhotoHandler) Upload(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}

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
	accountID := middleware.GetVaultAccountID(c)
	storageInfo, sErr := h.storageInfoService.Get(accountID)
	if sErr == nil && storageInfo.LimitBytes > 0 {
		if storageInfo.UsedBytes+fileHeader.Size > storageInfo.LimitBytes {
			return response.BadRequest(c, "err.storage_quota_exceeded", nil)
		}
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(mimeType, "image/") {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	// multipart 里的 Content-Type 由客户端任填，而帖子照片会按存下来的类型内联返回：
	// 只看 "image/" 前缀的话 image/svg+xml 能带着脚本存进来，形成存储型 XSS。
	// 只拦“浏览器会当文档执行”的内容，不能改成固定白名单：HEIC、AVIF、BMP 等格式
	// 一直可以作为帖子照片上传，收紧成白名单会让这些老用法失效。
	scriptable, err := postPhotoIsScriptable(src, mimeType)
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	if scriptable {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}

	result, err := h.vaultFileService.UploadPostPhoto(uint(postID), vaultID, fileHeader.Filename, mimeType, fileHeader.Size, src)
	if err != nil {
		return response.InternalError(c, "err.failed_to_upload_post_photo")
	}
	return response.Created(c, result)
}

// Delete godoc
//
//	@Summary		Delete a post photo
//	@Description	Delete a photo from a post
//	@Tags			post-photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			journal_id	path	integer	true	"Journal ID"
//	@Param			id			path	integer	true	"Post ID"
//	@Param			photoId		path	integer	true	"Photo ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/journals/{journal_id}/posts/{id}/photos/{photoId} [delete]
func (h *PostPhotoHandler) Delete(c *echo.Context) error {
	vaultID := c.Param("vault_id")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	photoID, err := strconv.ParseUint(c.Param("photoId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_photo_id", nil)
	}
	if err := h.vaultFileService.DeletePostPhoto(uint(photoID), uint(postID), vaultID); err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.photo_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_post_photo")
	}
	return response.NoContent(c)
}

// postPhotoIsScriptable 判断上传内容是否可能被浏览器当作可执行文档：
// 声明为 SVG，或者字节嗅探出来是文本类（SVG / HTML / XML 都属于这一类）。
// 真正的位图无论哪种格式都不会嗅探成 text/*。
func postPhotoIsScriptable(src interface {
	io.Reader
	io.Seeker
}, declaredMimeType string) (bool, error) {
	var sniffBuffer [512]byte
	n, err := src.Read(sniffBuffer[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(declaredMimeType, ";")[0]))
	if mediaType == "image/svg+xml" {
		return true, nil
	}
	return strings.HasPrefix(http.DetectContentType(sniffBuffer[:n]), "text/"), nil
}
