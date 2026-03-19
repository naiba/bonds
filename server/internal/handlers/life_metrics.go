package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type LifeMetricHandler struct {
	svc *services.LifeMetricService
}

func NewLifeMetricHandler(svc *services.LifeMetricService) *LifeMetricHandler {
	return &LifeMetricHandler{svc: svc}
}

// List godoc
//
//	@Summary		List life metrics
//	@Description	Return all life metrics for a vault with stats for the current user
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.LifeMetricResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics [get]
func (h *LifeMetricHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)
	metrics, err := h.svc.List(vaultID, userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_life_metrics")
	}
	return response.OK(c, metrics)
}

// Create godoc
//
//	@Summary		Create a life metric
//	@Description	Create a new life metric in a vault
//	@Tags			life-metrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			request		body		dto.CreateLifeMetricRequest		true	"Create life metric request"
//	@Success		201			{object}	response.APIResponse{data=dto.LifeMetricResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics [post]
func (h *LifeMetricHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	var req dto.CreateLifeMetricRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	metric, err := h.svc.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_life_metric")
	}
	return response.Created(c, metric)
}

// Update godoc
//
//	@Summary		Update a life metric
//	@Description	Update a life metric by ID
//	@Tags			life-metrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			id			path		integer							true	"Life Metric ID"
//	@Param			request		body		dto.UpdateLifeMetricRequest		true	"Update life metric request"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeMetricResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id} [put]
func (h *LifeMetricHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	var req dto.UpdateLifeMetricRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	metric, err := h.svc.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_life_metric")
	}
	return response.OK(c, metric)
}

// Delete godoc
//
//	@Summary		Delete a life metric
//	@Description	Delete a life metric by ID
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Life Metric ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id} [delete]
func (h *LifeMetricHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	if err := h.svc.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_life_metric")
	}
	return response.NoContent(c)
}

// Increment godoc
//
//	@Summary		Increment a life metric
//	@Description	Record a "+1" event for the current user on a life metric
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Life Metric ID"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeMetricResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id}/increment [post]
func (h *LifeMetricHandler) Increment(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	userID := middleware.GetUserID(c)
	metric, err := h.svc.Increment(uint(id), vaultID, userID)
	if err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_increment_life_metric")
	}
	return response.OK(c, metric)
}

// GetDetail godoc
//
//	@Summary		Get life metric detail with monthly breakdown
//	@Description	Return a life metric with 12-month event breakdown for a given year
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Life Metric ID"
//	@Param			year		query		integer	false	"Year (defaults to current year)"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeMetricDetailResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id}/detail [get]
func (h *LifeMetricHandler) GetDetail(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	yearStr := c.QueryParam("year")
	year := time.Now().Year()
	if yearStr != "" {
		parsed, err := strconv.Atoi(yearStr)
		if err != nil {
			return response.BadRequest(c, "err.invalid_year", nil)
		}
		year = parsed
	}
	userID := middleware.GetUserID(c)
	detail, err := h.svc.GetDetail(uint(id), vaultID, userID, year)
	if err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_life_metric_detail")
	}
	return response.OK(c, detail)
}
