package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

type AdminHandler struct {
	adminService   *services.AdminService
	settingService *services.SystemSettingService
	searchService  *services.SearchService
	db             *gorm.DB
	reloaders      []func()
}

func NewAdminHandler(adminService *services.AdminService, settingService *services.SystemSettingService, searchService *services.SearchService, db *gorm.DB) *AdminHandler {
	return &AdminHandler{adminService: adminService, settingService: settingService, searchService: searchService, db: db}
}

func (h *AdminHandler) RegisterReloader(fn func()) {
	h.reloaders = append(h.reloaders, fn)
}

// ListUsers godoc
//
//	@Summary		List all users
//	@Description	List all users with stats (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.AdminUserResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.adminService.ListUsers()
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_users")
	}
	return response.OK(c, users)
}

// ToggleUser godoc
//
//	@Summary		Enable or disable a user
//	@Description	Toggle a user's disabled status (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"User ID"
//	@Param			request	body		dto.AdminToggleUserRequest	true	"Toggle request"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/users/{id}/toggle [put]
func (h *AdminHandler) ToggleUser(c echo.Context) error {
	actorID := middleware.GetUserID(c)
	targetID := c.Param("id")

	var req dto.AdminToggleUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.Disabled == nil {
		return response.BadRequest(c, "err.disabled_field_required", nil)
	}

	err := h.adminService.ToggleUser(actorID, targetID, *req.Disabled)
	if err != nil {
		if errors.Is(err, services.ErrCannotDisableSelf) {
			return response.BadRequest(c, "err.cannot_disable_self", nil)
		}
		if errors.Is(err, services.ErrAdminUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_user")
	}

	return response.OK(c, map[string]string{"status": "ok"})
}

// SetAdmin godoc
//
//	@Summary		Set or unset instance admin
//	@Description	Promote or demote a user as instance administrator
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"User ID"
//	@Param			request	body		dto.AdminSetAdminRequest	true	"Admin request"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/users/{id}/admin [put]
func (h *AdminHandler) SetAdmin(c echo.Context) error {
	actorID := middleware.GetUserID(c)
	targetID := c.Param("id")

	var req dto.AdminSetAdminRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	err := h.adminService.SetAdmin(actorID, targetID, req.IsInstanceAdministrator)
	if err != nil {
		if errors.Is(err, services.ErrCannotDemoteSelf) {
			return response.BadRequest(c, "err.cannot_demote_self", nil)
		}
		if errors.Is(err, services.ErrAdminUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_set_admin")
	}

	return response.OK(c, map[string]string{"status": "ok"})
}

// SetStorageLimit godoc
//
//	@Summary		Set storage limit for a user's account
//	@Description	Set the storage quota (in MB) for a user's account. 0 means unlimited.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string								true	"User ID"
//	@Param			request	body		dto.AdminSetStorageLimitRequest	true	"Storage limit request"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/users/{id}/storage-limit [put]
func (h *AdminHandler) SetStorageLimit(c echo.Context) error {
	targetID := c.Param("id")

	var req dto.AdminSetStorageLimitRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if req.StorageLimitInMB < 0 {
		return response.BadRequest(c, "err.invalid_storage_limit", nil)
	}

	err := h.adminService.SetStorageLimit(targetID, req.StorageLimitInMB)
	if err != nil {
		if errors.Is(err, services.ErrAdminUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_set_storage_limit")
	}

	return response.OK(c, map[string]string{"status": "ok"})
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	Delete a user and all associated data (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"User ID"
//	@Success		204
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	actorID := middleware.GetUserID(c)
	targetID := c.Param("id")

	err := h.adminService.DeleteUser(actorID, targetID)
	if err != nil {
		if errors.Is(err, services.ErrCannotDeleteSelf) {
			return response.BadRequest(c, "err.cannot_delete_self", nil)
		}
		if errors.Is(err, services.ErrAdminUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_user")
	}

	return response.NoContent(c)
}

// GetSettings godoc
//
//	@Summary		Get system settings
//	@Description	Get all system settings (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.SystemSettingsResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/settings [get]
func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.settingService.GetAll()
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_settings")
	}
	return response.OK(c, dto.SystemSettingsResponse{Settings: settings})
}

// UpdateSettings godoc
//
//	@Summary		Update system settings
//	@Description	Bulk update system settings (instance admin only)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.UpdateSystemSettingsRequest	true	"Settings to update"
//	@Success		200		{object}	response.APIResponse{data=dto.SystemSettingsResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/admin/settings [put]
func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var req dto.UpdateSystemSettingsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	if err := h.settingService.BulkSet(req.Settings); err != nil {
		return response.InternalError(c, "err.failed_to_update_settings")
	}

	// Trigger hot-reload for all registered services
	for _, reload := range h.reloaders {
		reload()
	}

	settings, err := h.settingService.GetAll()
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_settings")
	}
	return response.OK(c, dto.SystemSettingsResponse{Settings: settings})
}

// RebuildSearchIndex godoc
//
//	@Summary		Rebuild search index
//	@Description	Rebuild the full-text search index by re-indexing all contacts and notes (instance admin only)
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=dto.RebuildSearchIndexResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		403	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/admin/search/rebuild [post]
func (h *AdminHandler) RebuildSearchIndex(c echo.Context) error {
	contactCount, noteCount, err := h.searchService.RebuildIndex(h.db)
	if err != nil {
		return response.InternalError(c, "err.failed_to_rebuild_search_index")
	}
	return response.OK(c, dto.RebuildSearchIndexResponse{
		ContactsIndexed: contactCount,
		NotesIndexed:    noteCount,
	})
}
