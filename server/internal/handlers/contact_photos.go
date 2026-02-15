package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VaultFileResponse

type ContactPhotoHandler struct {
	vaultFileService *services.VaultFileService
}

func NewContactPhotoHandler(vaultFileService *services.VaultFileService) *ContactPhotoHandler {
	return &ContactPhotoHandler{vaultFileService: vaultFileService}
}

// List godoc
//
//	@Summary		List contact photos
//	@Description	Return all photos for a contact
//	@Tags			contact-photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/photos [get]
func (h *ContactPhotoHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	files, err := h.vaultFileService.ListContactPhotos(contactID, vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contact_photos")
	}
	return response.OK(c, files)
}

// Get godoc
//
//	@Summary		Get a contact photo
//	@Description	Return a single contact photo by ID
//	@Tags			contact-photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			photoId		path		integer	true	"Photo ID"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultFileResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/photos/{photoId} [get]
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

// Delete godoc
//
//	@Summary		Delete a contact photo
//	@Description	Delete a contact photo by ID
//	@Tags			contact-photos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			photoId		path	integer	true	"Photo ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/photos/{photoId} [delete]
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

// List godoc
//
//	@Summary		List contact documents
//	@Description	Return all documents for a contact
//	@Tags			contact-documents
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultFileResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/documents [get]
func (h *ContactDocumentHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	files, err := h.vaultFileService.ListContactDocuments(contactID, vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_contact_documents")
	}
	return response.OK(c, files)
}

// Delete godoc
//
//	@Summary		Delete a contact document
//	@Description	Delete a contact document by ID
//	@Tags			contact-documents
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Document ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/documents/{id} [delete]
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
