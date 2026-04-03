package handlers

import (
	"bytes"
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

// dto anchor — swag 需要解析此类型，但 handler 代码不直接引用 dto 包中的类型名
var _ dto.MonicaImportResponse

type MonicaImportHandler struct {
	monicaImportService *services.MonicaImportService
}

func NewMonicaImportHandler(svc *services.MonicaImportService) *MonicaImportHandler {
	return &MonicaImportHandler{monicaImportService: svc}
}

// Import godoc
//
//	@Summary		Import Monica 4.x JSON data
//	@Description	Import contacts and related data from a Monica 4.x JSON export file
//	@Tags			Vault Settings
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			file		formData	file	true	"Monica JSON export file"
//	@Success		200			{object}	response.APIResponse{data=dto.MonicaImportResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		403			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/import/monica [post]
func (h *MonicaImportHandler) Import(c echo.Context) error {
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

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(src); err != nil {
		return response.InternalError(c, "err.failed_to_read_file")
	}

	result, err := h.monicaImportService.Import(vaultID, userID, buf.Bytes())
	if err != nil {
		if errors.Is(err, services.ErrMonicaInvalidJSON) || errors.Is(err, services.ErrMonicaInvalidVersion) {
			return response.BadRequest(c, err.Error(), nil)
		}
		return response.InternalError(c, "err.failed_to_import_monica")
	}

	return response.OK(c, result)
}
