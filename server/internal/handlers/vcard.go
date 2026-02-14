package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type VCardHandler struct {
	vcardService *services.VCardService
}

func NewVCardHandler(vcardService *services.VCardService) *VCardHandler {
	return &VCardHandler{vcardService: vcardService}
}

func (h *VCardHandler) ExportContact(c echo.Context) error {
	contactID := c.Param("contact_id")

	data, err := h.vcardService.ExportContact(contactID)
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
