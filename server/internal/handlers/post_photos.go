package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultFileResponse

type PostPhotoHandler struct {
	vaultFileService *services.VaultFileService
}

func NewPostPhotoHandler(vaultFileService *services.VaultFileService) *PostPhotoHandler {
	return &PostPhotoHandler{vaultFileService: vaultFileService}
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
