package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactPhotoHandler struct {
	vaultFileService *services.VaultFileService
}

func NewContactPhotoHandler(vaultFileService *services.VaultFileService) *ContactPhotoHandler {
	return &ContactPhotoHandler{vaultFileService: vaultFileService}
}

func (h *ContactPhotoHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	files, err := h.vaultFileService.ListContactPhotos(contactID, vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contact_photos")
	}
	return response.OK(c, files)
}

func (h *ContactPhotoHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("photoId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_photo_id", nil)
	}
	file, err := h.vaultFileService.GetContactPhoto(uint(id), contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.photo_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_contact_photo")
	}
	return response.OK(c, file)
}

func (h *ContactPhotoHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("photoId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_photo_id", nil)
	}
	if err := h.vaultFileService.DeleteContactPhoto(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.photo_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_contact_photo")
	}
	return response.NoContent(c)
}

type ContactDocumentHandler struct {
	vaultFileService *services.VaultFileService
}

func NewContactDocumentHandler(vaultFileService *services.VaultFileService) *ContactDocumentHandler {
	return &ContactDocumentHandler{vaultFileService: vaultFileService}
}

func (h *ContactDocumentHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	files, err := h.vaultFileService.ListContactDocuments(contactID, vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contact_documents")
	}
	return response.OK(c, files)
}

func (h *ContactDocumentHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_document_id", nil)
	}
	if err := h.vaultFileService.DeleteContactDocument(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrFileNotFound) {
			return response.NotFound(c, "err.document_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_contact_document")
	}
	return response.NoContent(c)
}
