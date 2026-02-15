package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.VCardImportResponse

type VCardHandler struct {
	vcardService *services.VCardService
}

func NewVCardHandler(vcardService *services.VCardService) *VCardHandler {
	return &VCardHandler{vcardService: vcardService}
}

// ExportContact godoc
//
//	@Summary		Export contact as vCard
//	@Description	Export a single contact as vCard 4.0 format
//	@Tags			vcard
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{file}		file
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/vcard [get]
func (h *VCardHandler) ExportContact(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")

	data, err := h.vcardService.ExportContact(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_export_vcard")
	}

	c.Response().Header().Set("Content-Type", "text/vcard")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=contact.vcf")
	return c.Blob(http.StatusOK, "text/vcard", data)
}

// ExportVault godoc
//
//	@Summary		Export all contacts as vCard
//	@Description	Export all contacts in a vault as vCard 4.0 format
//	@Tags			vcard
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{file}		file
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/export [get]
func (h *VCardHandler) ExportVault(c echo.Context) error {
	vaultID := c.Param("vault_id")

	data, err := h.vcardService.ExportVault(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_export_vcard")
	}

	c.Response().Header().Set("Content-Type", "text/vcard")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=contacts.vcf")
	return c.Blob(http.StatusOK, "text/vcard", data)
}

// ImportVCard godoc
//
//	@Summary		Import contacts from vCard
//	@Description	Import contacts from a .vcf file
//	@Tags			vcard
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			file		formData	file	true	"vCard file (.vcf)"
//	@Success		201			{object}	response.APIResponse{data=dto.VCardImportResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/import [post]
func (h *VCardHandler) ImportVCard(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "err.file_required", nil)
	}

	src, err := file.Open()
	if err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}
	defer src.Close()

	result, err := h.vcardService.ImportVCard(vaultID, userID, src)
	if err != nil {
		if errors.Is(err, services.ErrVCardInvalidData) {
			return response.BadRequest(c, "err.invalid_vcard_data", nil)
		}
		return response.InternalError(c, "err.failed_to_import_vcard")
	}

	return response.Created(c, result)
}
