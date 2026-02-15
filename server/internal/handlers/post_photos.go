package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PostPhotoHandler struct {
	vaultFileService *services.VaultFileService
}

func NewPostPhotoHandler(vaultFileService *services.VaultFileService) *PostPhotoHandler {
	return &PostPhotoHandler{vaultFileService: vaultFileService}
}

func (h *PostPhotoHandler) List(c echo.Context) error {
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

func (h *PostPhotoHandler) Upload(c echo.Context) error {
	vaultID := c.Param("vault_id")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}

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
	if !strings.HasPrefix(mimeType, "image/") {
		return response.BadRequest(c, "err.file_type_not_allowed", nil)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	result, err := h.vaultFileService.UploadPostPhoto(uint(postID), vaultID, fileHeader.Filename, mimeType, fileHeader.Size, src)
	if err != nil {
		return response.InternalError(c, "err.failed_to_upload_post_photo")
	}
	return response.Created(c, result)
}

func (h *PostPhotoHandler) Delete(c echo.Context) error {
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
