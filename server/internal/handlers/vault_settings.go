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

func (h *VaultSettingsHandler) ListUsers(c echo.Context) error {
	vaultID := c.Param("vault_id")
	users, err := h.usersService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_vault_users")
	}
	return response.OK(c, users)
}

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

func (h *VaultSettingsHandler) ListLabels(c echo.Context) error {
	vaultID := c.Param("vault_id")
	labels, err := h.labelService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_labels")
	}
	return response.OK(c, labels)
}

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

func (h *VaultSettingsHandler) ListTags(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tags, err := h.tagService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_tags")
	}
	return response.OK(c, tags)
}

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

func (h *VaultSettingsHandler) ListDateTypes(c echo.Context) error {
	vaultID := c.Param("vault_id")
	types, err := h.dateTypeService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_date_types")
	}
	return response.OK(c, types)
}

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

func (h *VaultSettingsHandler) ListMoodParams(c echo.Context) error {
	vaultID := c.Param("vault_id")
	params, err := h.moodService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_mood_params")
	}
	return response.OK(c, params)
}

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

func (h *VaultSettingsHandler) ListLifeEventCategories(c echo.Context) error {
	vaultID := c.Param("vault_id")
	cats, err := h.lifeEventSvc.ListCategories(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_life_event_categories")
	}
	return response.OK(c, cats)
}

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

func (h *VaultSettingsHandler) ListQuickFactTemplates(c echo.Context) error {
	vaultID := c.Param("vault_id")
	tpls, err := h.quickFactSvc.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_quick_fact_templates")
	}
	return response.OK(c, tpls)
}

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
