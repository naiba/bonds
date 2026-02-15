package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type StorageInfoHandler struct {
	storageInfoService *services.StorageInfoService
}

func NewStorageInfoHandler(svc *services.StorageInfoService) *StorageInfoHandler {
	return &StorageInfoHandler{storageInfoService: svc}
}

func (h *StorageInfoHandler) Get(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	info, err := h.storageInfoService.Get(accountID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_storage_info")
	}
	return response.OK(c, info)
}
