package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type VaultSettingsHandler struct {
	settingsService *services.VaultSettingsService
	usersService    *services.VaultUsersService
	labelService    *services.VaultLabelService
	tagService      *services.VaultTagService
	dateTypeService *services.VaultImportantDateTypeService
	moodService     *services.VaultMoodParamService
	lifeEventSvc    *services.VaultLifeEventService
	quickFactSvc    *services.VaultQuickFactTemplateService
}

func NewVaultSettingsHandler(
	settingsService *services.VaultSettingsService,
	usersService *services.VaultUsersService,
	labelService *services.VaultLabelService,
	tagService *services.VaultTagService,
	dateTypeService *services.VaultImportantDateTypeService,
	moodService *services.VaultMoodParamService,
	lifeEventSvc *services.VaultLifeEventService,
	quickFactSvc *services.VaultQuickFactTemplateService,
) *VaultSettingsHandler {
	return &VaultSettingsHandler{
		settingsService: settingsService,
		usersService:    usersService,
		labelService:    labelService,
		tagService:      tagService,
		dateTypeService: dateTypeService,
		moodService:     moodService,
		lifeEventSvc:    lifeEventSvc,
		quickFactSvc:    quickFactSvc,
	}
}

// Get godoc
//
//	@Summary		Get vault settings
//	@Description	Return settings for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultSettingsResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings [get]
func (h *VaultSettingsHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	settings, err := h.settingsService.Get(vaultID)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_vault_settings")
	}
	return response.OK(c, settings)
}

// Update godoc
//
//	@Summary		Update vault settings
//	@Description	Update name and description for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			request		body		dto.UpdateVaultSettingsRequest	true	"Settings"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultSettingsResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings [put]
func (h *VaultSettingsHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.UpdateVaultSettingsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	settings, err := h.settingsService.Update(vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrVaultNotFound) {
			return response.NotFound(c, "err.vault_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_vault_settings")
	}
	return response.OK(c, settings)
}

// UpdateVisibility godoc
//
//	@Summary		Update tab visibility
//	@Description	Update tab visibility settings for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			request		body		dto.UpdateTabVisibilityRequest	true	"Visibility settings"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultSettingsResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/visibility [put]
func (h *VaultSettingsHandler) UpdateVisibility(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.UpdateTabVisibilityRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	settings, err := h.settingsService.UpdateVisibility(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_update_visibility")
	}
	return response.OK(c, settings)
}

// UpdateTemplate godoc
//
//	@Summary		Update default template
//	@Description	Update the default template for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			request		body		dto.UpdateDefaultTemplateRequest	true	"Template ID"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultSettingsResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/template [put]
func (h *VaultSettingsHandler) UpdateTemplate(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.UpdateDefaultTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	settings, err := h.settingsService.UpdateDefaultTemplate(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_update_template")
	}
	return response.OK(c, settings)
}

// ListUsers godoc
//
//	@Summary		List vault users
//	@Description	Return all users with access to a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.VaultUserResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/users [get]
func (h *VaultSettingsHandler) ListUsers(c echo.Context) error {
	vaultID := c.Param("vault_id")
	users, err := h.usersService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_users")
	}
	return response.OK(c, users)
}

// AddUser godoc
//
//	@Summary		Add a user to a vault
//	@Description	Add a user to a vault by email
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.AddVaultUserRequest	true	"User details"
//	@Success		201			{object}	response.APIResponse{data=dto.VaultUserResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/users [post]
func (h *VaultSettingsHandler) AddUser(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.AddVaultUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	user, err := h.usersService.Add(vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserEmailNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		if errors.Is(err, services.ErrUserAlreadyInVault) {
			return response.BadRequest(c, "err.user_already_in_vault", nil)
		}
		return response.InternalError(c, "err.failed_to_add_vault_user")
	}
	return response.Created(c, user)
}

// UpdateUserPermission godoc
//
//	@Summary		Update vault user permission
//	@Description	Update a user's permission level in a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			id			path		integer								true	"Vault User ID"
//	@Param			request		body		dto.UpdateVaultUserPermRequest	true	"Permission"
//	@Success		200			{object}	response.APIResponse{data=dto.VaultUserResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/users/{id} [put]
func (h *VaultSettingsHandler) UpdateUserPermission(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_user_id", nil)
	}
	var req dto.UpdateVaultUserPermRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	user, err := h.usersService.UpdatePermission(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrVaultUserNotFound) {
			return response.NotFound(c, "err.vault_user_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_vault_user")
	}
	return response.OK(c, user)
}

// RemoveUser godoc
//
//	@Summary		Remove a user from a vault
//	@Description	Remove a user's access to a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Vault User ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/users/{id} [delete]
func (h *VaultSettingsHandler) RemoveUser(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_user_id", nil)
	}
	if err := h.usersService.Remove(uint(id), vaultID, userID); err != nil {
		if errors.Is(err, services.ErrVaultUserNotFound) {
			return response.NotFound(c, "err.vault_user_not_found")
		}
		if errors.Is(err, services.ErrCannotRemoveSelf) {
			return response.BadRequest(c, "err.cannot_remove_self", nil)
		}
		return response.InternalError(c, "err.failed_to_remove_vault_user")
	}
	return response.NoContent(c)
}

// ListLabels godoc
//
//	@Summary		List vault labels
//	@Description	Return all labels for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.LabelResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/labels [get]
func (h *VaultSettingsHandler) ListLabels(c echo.Context) error {
	vaultID := c.Param("vault_id")
	labels, err := h.labelService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_labels")
	}
	return response.OK(c, labels)
}

// CreateLabel godoc
//
//	@Summary		Create a vault label
//	@Description	Create a new label for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.CreateLabelRequest	true	"Label details"
//	@Success		201			{object}	response.APIResponse{data=dto.LabelResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/labels [post]
func (h *VaultSettingsHandler) CreateLabel(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateLabelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	label, err := h.labelService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_label")
	}
	return response.Created(c, label)
}

// UpdateLabel godoc
//
//	@Summary		Update a vault label
//	@Description	Update an existing vault label
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Label ID"
//	@Param			request		body		dto.UpdateLabelRequest	true	"Label details"
//	@Success		200			{object}	response.APIResponse{data=dto.LabelResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/labels/{id} [put]
func (h *VaultSettingsHandler) UpdateLabel(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_label_id", nil)
	}
	var req dto.UpdateLabelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	label, err := h.labelService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLabelNotFound) {
			return response.NotFound(c, "err.label_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_label")
	}
	return response.OK(c, label)
}

// DeleteLabel godoc
//
//	@Summary		Delete a vault label
//	@Description	Delete a label from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Label ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/labels/{id} [delete]
func (h *VaultSettingsHandler) DeleteLabel(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_label_id", nil)
	}
	if err := h.labelService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrLabelNotFound) {
			return response.NotFound(c, "err.label_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_label")
	}
	return response.NoContent(c)
}

// ListTags godoc
//
//	@Summary		List vault tags
//	@Description	Return all tags for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.TagResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/tags [get]
func (h *VaultSettingsHandler) ListTags(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tags, err := h.tagService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_tags")
	}
	return response.OK(c, tags)
}

// CreateTag godoc
//
//	@Summary		Create a vault tag
//	@Description	Create a new tag for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			request		body		dto.CreateTagRequest	true	"Tag details"
//	@Success		201			{object}	response.APIResponse{data=dto.TagResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/tags [post]
func (h *VaultSettingsHandler) CreateTag(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	tag, err := h.tagService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_tag")
	}
	return response.Created(c, tag)
}

// UpdateTag godoc
//
//	@Summary		Update a vault tag
//	@Description	Update an existing vault tag
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			id			path		integer					true	"Tag ID"
//	@Param			request		body		dto.UpdateTagRequest	true	"Tag details"
//	@Success		200			{object}	response.APIResponse{data=dto.TagResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/tags/{id} [put]
func (h *VaultSettingsHandler) UpdateTag(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_tag_id", nil)
	}
	var req dto.UpdateTagRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	tag, err := h.tagService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrTagNotFound) {
			return response.NotFound(c, "err.tag_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_tag")
	}
	return response.OK(c, tag)
}

// DeleteTag godoc
//
//	@Summary		Delete a vault tag
//	@Description	Delete a tag from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Tag ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/tags/{id} [delete]
func (h *VaultSettingsHandler) DeleteTag(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_tag_id", nil)
	}
	if err := h.tagService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrTagNotFound) {
			return response.NotFound(c, "err.tag_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_tag")
	}
	return response.NoContent(c)
}

// ListDateTypes godoc
//
//	@Summary		List important date types
//	@Description	Return all important date types for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ImportantDateTypeResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/dateTypes [get]
func (h *VaultSettingsHandler) ListDateTypes(c echo.Context) error {
	vaultID := c.Param("vault_id")
	types, err := h.dateTypeService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_date_types")
	}
	return response.OK(c, types)
}

// CreateDateType godoc
//
//	@Summary		Create an important date type
//	@Description	Create a new important date type for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			request		body		dto.CreateImportantDateTypeRequest	true	"Date type details"
//	@Success		201			{object}	response.APIResponse{data=dto.ImportantDateTypeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/dateTypes [post]
func (h *VaultSettingsHandler) CreateDateType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateImportantDateTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	dt, err := h.dateTypeService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_date_type")
	}
	return response.Created(c, dt)
}

// UpdateDateType godoc
//
//	@Summary		Update an important date type
//	@Description	Update an existing important date type
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			id			path		integer									true	"Date Type ID"
//	@Param			request		body		dto.UpdateImportantDateTypeRequest	true	"Date type details"
//	@Success		200			{object}	response.APIResponse{data=dto.ImportantDateTypeResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/dateTypes/{id} [put]
func (h *VaultSettingsHandler) UpdateDateType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_date_type_id", nil)
	}
	var req dto.UpdateImportantDateTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	dt, err := h.dateTypeService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrDateTypeNotFound) {
			return response.NotFound(c, "err.date_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_date_type")
	}
	return response.OK(c, dt)
}

// DeleteDateType godoc
//
//	@Summary		Delete an important date type
//	@Description	Delete an important date type from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Date Type ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/dateTypes/{id} [delete]
func (h *VaultSettingsHandler) DeleteDateType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_date_type_id", nil)
	}
	if err := h.dateTypeService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrDateTypeNotFound) {
			return response.NotFound(c, "err.date_type_not_found")
		}
		if errors.Is(err, services.ErrCannotDeleteDefault) {
			return response.BadRequest(c, "err.cannot_delete_default", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_date_type")
	}
	return response.NoContent(c)
}

// ListMoodParams godoc
//
//	@Summary		List mood tracking parameters
//	@Description	Return all mood tracking parameters for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.MoodTrackingParameterResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/moodParams [get]
func (h *VaultSettingsHandler) ListMoodParams(c echo.Context) error {
	vaultID := c.Param("vault_id")
	params, err := h.moodService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_mood_params")
	}
	return response.OK(c, params)
}

// CreateMoodParam godoc
//
//	@Summary		Create a mood tracking parameter
//	@Description	Create a new mood tracking parameter for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string										true	"Vault ID"
//	@Param			request		body		dto.CreateMoodTrackingParameterRequest	true	"Parameter details"
//	@Success		201			{object}	response.APIResponse{data=dto.MoodTrackingParameterResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/moodParams [post]
func (h *VaultSettingsHandler) CreateMoodParam(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateMoodTrackingParameterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	param, err := h.moodService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_mood_param")
	}
	return response.Created(c, param)
}

// UpdateMoodParam godoc
//
//	@Summary		Update a mood tracking parameter
//	@Description	Update an existing mood tracking parameter
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string										true	"Vault ID"
//	@Param			id			path		integer										true	"Parameter ID"
//	@Param			request		body		dto.UpdateMoodTrackingParameterRequest	true	"Parameter details"
//	@Success		200			{object}	response.APIResponse{data=dto.MoodTrackingParameterResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/moodParams/{id} [put]
func (h *VaultSettingsHandler) UpdateMoodParam(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_mood_param_id", nil)
	}
	var req dto.UpdateMoodTrackingParameterRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	param, err := h.moodService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrMoodParamNotFound) {
			return response.NotFound(c, "err.mood_param_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_mood_param")
	}
	return response.OK(c, param)
}

// UpdateMoodParamOrder godoc
//
//	@Summary		Update mood parameter position
//	@Description	Update the position of a mood tracking parameter
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Parameter ID"
//	@Param			request		body		dto.UpdatePositionRequest	true	"Position"
//	@Success		200			{object}	response.APIResponse{data=dto.MoodTrackingParameterResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/moodParams/{id}/position [post]
func (h *VaultSettingsHandler) UpdateMoodParamOrder(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_mood_param_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	param, err := h.moodService.UpdatePosition(uint(id), vaultID, req.Position)
	if err != nil {
		if errors.Is(err, services.ErrMoodParamNotFound) {
			return response.NotFound(c, "err.mood_param_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_mood_param_order")
	}
	return response.OK(c, param)
}

// DeleteMoodParam godoc
//
//	@Summary		Delete a mood tracking parameter
//	@Description	Delete a mood tracking parameter from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Parameter ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/moodParams/{id} [delete]
func (h *VaultSettingsHandler) DeleteMoodParam(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_mood_param_id", nil)
	}
	if err := h.moodService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrMoodParamNotFound) {
			return response.NotFound(c, "err.mood_param_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_mood_param")
	}
	return response.NoContent(c)
}

// ListLifeEventCategories godoc
//
//	@Summary		List life event categories
//	@Description	Return all life event categories for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.LifeEventCategoryResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories [get]
func (h *VaultSettingsHandler) ListLifeEventCategories(c echo.Context) error {
	vaultID := c.Param("vault_id")
	cats, err := h.lifeEventSvc.ListCategories(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_life_event_categories")
	}
	return response.OK(c, cats)
}

// CreateLifeEventCategory godoc
//
//	@Summary		Create a life event category
//	@Description	Create a new life event category for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			request		body		dto.CreateLifeEventCategoryRequest	true	"Category details"
//	@Success		201			{object}	response.APIResponse{data=dto.LifeEventCategoryResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories [post]
func (h *VaultSettingsHandler) CreateLifeEventCategory(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateLifeEventCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	cat, err := h.lifeEventSvc.CreateCategory(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_life_event_category")
	}
	return response.Created(c, cat)
}

// UpdateLifeEventCategory godoc
//
//	@Summary		Update a life event category
//	@Description	Update an existing life event category
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			id			path		integer									true	"Category ID"
//	@Param			request		body		dto.UpdateLifeEventCategoryRequest	true	"Category details"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeEventCategoryResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{id} [put]
func (h *VaultSettingsHandler) UpdateLifeEventCategory(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	var req dto.UpdateLifeEventCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	cat, err := h.lifeEventSvc.UpdateCategory(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_life_event_category")
	}
	return response.OK(c, cat)
}

// UpdateLifeEventCategoryOrder godoc
//
//	@Summary		Update life event category position
//	@Description	Update the position of a life event category
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Category ID"
//	@Param			request		body		dto.UpdatePositionRequest	true	"Position"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeEventCategoryResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{id}/position [post]
func (h *VaultSettingsHandler) UpdateLifeEventCategoryOrder(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	cat, err := h.lifeEventSvc.UpdateCategoryPosition(uint(id), vaultID, req.Position)
	if err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_category_order")
	}
	return response.OK(c, cat)
}

// DeleteLifeEventCategory godoc
//
//	@Summary		Delete a life event category
//	@Description	Delete a life event category from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Category ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{id} [delete]
func (h *VaultSettingsHandler) DeleteLifeEventCategory(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	if err := h.lifeEventSvc.DeleteCategory(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		if errors.Is(err, services.ErrCannotDeleteDefault) {
			return response.BadRequest(c, "err.cannot_delete_default", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_life_event_category")
	}
	return response.NoContent(c)
}

// CreateLifeEventType godoc
//
//	@Summary		Create a life event type
//	@Description	Create a new life event type under a category
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id		path		string							true	"Vault ID"
//	@Param			categoryId	path		integer							true	"Category ID"
//	@Param			request			body		dto.CreateLifeEventTypeRequest	true	"Type details"
//	@Success		201				{object}	response.APIResponse{data=dto.LifeEventTypeResponse}
//	@Failure		400				{object}	response.APIResponse
//	@Failure		401				{object}	response.APIResponse
//	@Failure		404				{object}	response.APIResponse
//	@Failure		422				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{categoryId}/types [post]
func (h *VaultSettingsHandler) CreateLifeEventType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	var req dto.CreateLifeEventTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	lt, err := h.lifeEventSvc.CreateType(uint(categoryID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_life_event_type")
	}
	return response.Created(c, lt)
}

// UpdateLifeEventType godoc
//
//	@Summary		Update a life event type
//	@Description	Update an existing life event type
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id		path		string							true	"Vault ID"
//	@Param			categoryId	path		integer							true	"Category ID"
//	@Param			typeId		path		integer							true	"Type ID"
//	@Param			request			body		dto.UpdateLifeEventTypeRequest	true	"Type details"
//	@Success		200				{object}	response.APIResponse{data=dto.LifeEventTypeResponse}
//	@Failure		400				{object}	response.APIResponse
//	@Failure		401				{object}	response.APIResponse
//	@Failure		404				{object}	response.APIResponse
//	@Failure		422				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{categoryId}/types/{typeId} [put]
func (h *VaultSettingsHandler) UpdateLifeEventType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	typeID, err := strconv.ParseUint(c.Param("typeId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_type_id", nil)
	}
	var req dto.UpdateLifeEventTypeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	lt, err := h.lifeEventSvc.UpdateType(uint(typeID), uint(categoryID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		if errors.Is(err, services.ErrLifeTypeNotFound) {
			return response.NotFound(c, "err.life_event_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_life_event_type")
	}
	return response.OK(c, lt)
}

// UpdateLifeEventTypeOrder godoc
//
//	@Summary		Update life event type position
//	@Description	Update the position of a life event type
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id		path		string						true	"Vault ID"
//	@Param			categoryId	path		integer						true	"Category ID"
//	@Param			typeId		path		integer						true	"Type ID"
//	@Param			request			body		dto.UpdatePositionRequest	true	"Position"
//	@Success		200				{object}	response.APIResponse{data=dto.LifeEventTypeResponse}
//	@Failure		400				{object}	response.APIResponse
//	@Failure		401				{object}	response.APIResponse
//	@Failure		404				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{categoryId}/lifeEventTypes/{typeId}/position [post]
func (h *VaultSettingsHandler) UpdateLifeEventTypeOrder(c echo.Context) error {
	vaultID := c.Param("vault_id")
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	typeID, err := strconv.ParseUint(c.Param("typeId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_type_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	lt, err := h.lifeEventSvc.UpdateTypePosition(uint(typeID), uint(categoryID), vaultID, req.Position)
	if err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		if errors.Is(err, services.ErrLifeTypeNotFound) {
			return response.NotFound(c, "err.life_event_type_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_type_order")
	}
	return response.OK(c, lt)
}

// DeleteLifeEventType godoc
//
//	@Summary		Delete a life event type
//	@Description	Delete a life event type from a category
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id		path	string	true	"Vault ID"
//	@Param			categoryId	path	integer	true	"Category ID"
//	@Param			typeId		path	integer	true	"Type ID"
//	@Success		204				"No Content"
//	@Failure		400				{object}	response.APIResponse
//	@Failure		401				{object}	response.APIResponse
//	@Failure		404				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/lifeEventCategories/{categoryId}/types/{typeId} [delete]
func (h *VaultSettingsHandler) DeleteLifeEventType(c echo.Context) error {
	vaultID := c.Param("vault_id")
	categoryID, err := strconv.ParseUint(c.Param("categoryId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_category_id", nil)
	}
	typeID, err := strconv.ParseUint(c.Param("typeId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_type_id", nil)
	}
	if err := h.lifeEventSvc.DeleteType(uint(typeID), uint(categoryID), vaultID); err != nil {
		if errors.Is(err, services.ErrLifeCategoryNotFound) {
			return response.NotFound(c, "err.life_event_category_not_found")
		}
		if errors.Is(err, services.ErrLifeTypeNotFound) {
			return response.NotFound(c, "err.life_event_type_not_found")
		}
		if errors.Is(err, services.ErrCannotDeleteDefault) {
			return response.BadRequest(c, "err.cannot_delete_default", nil)
		}
		return response.InternalError(c, "err.failed_to_delete_life_event_type")
	}
	return response.NoContent(c)
}

// ListQuickFactTemplates godoc
//
//	@Summary		List quick fact templates
//	@Description	Return all quick fact templates for a vault
//	@Tags			vault-settings
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.QuickFactTemplateResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/quickFactTemplates [get]
func (h *VaultSettingsHandler) ListQuickFactTemplates(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tpls, err := h.quickFactSvc.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_quick_fact_templates")
	}
	return response.OK(c, tpls)
}

// CreateQuickFactTemplate godoc
//
//	@Summary		Create a quick fact template
//	@Description	Create a new quick fact template for a vault
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			request		body		dto.CreateQuickFactTemplateRequest	true	"Template details"
//	@Success		201			{object}	response.APIResponse{data=dto.QuickFactTemplateResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/quickFactTemplates [post]
func (h *VaultSettingsHandler) CreateQuickFactTemplate(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateQuickFactTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	tpl, err := h.quickFactSvc.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_quick_fact_template")
	}
	return response.Created(c, tpl)
}

// UpdateQuickFactTemplate godoc
//
//	@Summary		Update a quick fact template
//	@Description	Update an existing quick fact template
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string									true	"Vault ID"
//	@Param			id			path		integer									true	"Template ID"
//	@Param			request		body		dto.UpdateQuickFactTemplateRequest	true	"Template details"
//	@Success		200			{object}	response.APIResponse{data=dto.QuickFactTemplateResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/quickFactTemplates/{id} [put]
func (h *VaultSettingsHandler) UpdateQuickFactTemplate(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	var req dto.UpdateQuickFactTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	tpl, err := h.quickFactSvc.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrQuickFactTplNotFound) {
			return response.NotFound(c, "err.quick_fact_template_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_quick_fact_template")
	}
	return response.OK(c, tpl)
}

// UpdateQuickFactTemplateOrder godoc
//
//	@Summary		Update quick fact template position
//	@Description	Update the position of a quick fact template
//	@Tags			vault-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Template ID"
//	@Param			request		body		dto.UpdatePositionRequest	true	"Position"
//	@Success		200			{object}	response.APIResponse{data=dto.QuickFactTemplateResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/quickFactTemplates/{id}/position [post]
func (h *VaultSettingsHandler) UpdateQuickFactTemplateOrder(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	var req dto.UpdatePositionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	tpl, err := h.quickFactSvc.UpdatePosition(uint(id), vaultID, req.Position)
	if err != nil {
		if errors.Is(err, services.ErrQuickFactTplNotFound) {
			return response.NotFound(c, "err.quick_fact_template_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_template_order")
	}
	return response.OK(c, tpl)
}

// DeleteQuickFactTemplate godoc
//
//	@Summary		Delete a quick fact template
//	@Description	Delete a quick fact template from a vault
//	@Tags			vault-settings
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Template ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/settings/quickFactTemplates/{id} [delete]
func (h *VaultSettingsHandler) DeleteQuickFactTemplate(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_template_id", nil)
	}
	if err := h.quickFactSvc.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrQuickFactTplNotFound) {
			return response.NotFound(c, "err.quick_fact_template_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_quick_fact_template")
	}
	return response.NoContent(c)
}
