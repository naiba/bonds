package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.StorageResponse

type StorageInfoHandler struct {
	storageInfoService *services.StorageInfoService
}

func NewStorageInfoHandler(svc *services.StorageInfoService) *StorageInfoHandler {
	return &StorageInfoHandler{storageInfoService: svc}
}

// Get godoc
//
//	@Summary		Get storage information
//	@Description	Return storage usage for the account
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.StorageResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/storage [get]
func (h *StorageInfoHandler) Get(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	info, err := h.storageInfoService.Get(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_storage_info")
	}
	return response.OK(c, info)
}
