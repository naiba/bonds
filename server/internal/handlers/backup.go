package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.BackupResponse
var _ dto.BackupConfigResponse

type BackupHandler struct {
	backupService *services.BackupService
}

func NewBackupHandler(svc *services.BackupService) *BackupHandler {
	return &BackupHandler{backupService: svc}
}

// List godoc
//
//	@Summary		List backups
//	@Description	List all available backup files
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.BackupResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/backups [get]
func (h *BackupHandler) List(c echo.Context) error {
	backups, err := h.backupService.List()
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_backups")
	}
	return response.OK(c, backups)
}

// Create godoc
//
//	@Summary		Create backup
//	@Description	Create a new database backup
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	response.APIResponse{data=dto.BackupResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/backups [post]
func (h *BackupHandler) Create(c echo.Context) error {
	backup, err := h.backupService.Create()
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_backup")
	}
	return response.Created(c, backup)
}

// GetConfig godoc
//
//	@Summary		Get backup config
//	@Description	Get current backup configuration
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.BackupConfigResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Router			/settings/backups/config [get]
func (h *BackupHandler) GetConfig(c echo.Context) error {
	cfg := h.backupService.GetConfig()
	return response.OK(c, cfg)
}

// Download godoc
//
//	@Summary		Download backup
//	@Description	Download a backup file
//	@Tags			settings
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Param			filename	path	string	true	"Backup filename"
//	@Success		200
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/settings/backups/{filename}/download [get]
func (h *BackupHandler) Download(c echo.Context) error {
	filename := c.Param("filename")
	fullPath, err := h.backupService.GetFilePath(filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return response.NotFound(c, "err.backup_not_found")
		}
		if errors.Is(err, services.ErrBackupInvalidFilename) {
			return response.BadRequest(c, "err.invalid_backup_filename", nil)
		}
		return response.InternalError(c, "err.failed_to_download_backup")
	}
	return c.File(fullPath)
}

// Delete godoc
//
//	@Summary		Delete backup
//	@Description	Delete a backup file
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			filename	path	string	true	"Backup filename"
//	@Success		204
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/settings/backups/{filename} [delete]
func (h *BackupHandler) Delete(c echo.Context) error {
	filename := c.Param("filename")
	err := h.backupService.Delete(filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return response.NotFound(c, "err.backup_not_found")
		}
		if errors.Is(err, services.ErrBackupInvalidFilename) {
			return response.BadRequest(c, "err.invalid_backup_filename", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_backup")
	}
	return response.NoContent(c)
}

// Restore godoc
//
//	@Summary		Restore backup
//	@Description	Restore from a backup file
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			filename	path	string	true	"Backup filename"
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/backups/{filename}/restore [post]
func (h *BackupHandler) Restore(c echo.Context) error {
	filename := c.Param("filename")
	err := h.backupService.Restore(filename)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			return response.NotFound(c, "err.backup_not_found")
		}
		if errors.Is(err, services.ErrBackupInvalidFilename) {
			return response.BadRequest(c, "err.invalid_backup_filename", nil)
		}
		return response.InternalError(c, "err.failed_to_restore_backup")
	}
	return response.OK(c, map[string]string{"status": "restored"})
}
