package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
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
//	@Description	Return all life metrics for a vault
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.LifeMetricResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics [get]
func (h *LifeMetricHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	metrics, err := h.svc.List(vaultID)
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

// AddContact godoc
//
//	@Summary		Add contact to life metric
//	@Description	Associate a contact with a life metric
//	@Tags			life-metrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			id			path		integer								true	"Life Metric ID"
//	@Param			request		body		dto.AddLifeMetricContactRequest		true	"Add contact request"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id}/contacts [post]
func (h *LifeMetricHandler) AddContact(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	var req dto.AddLifeMetricContactRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	if err := h.svc.AddContact(uint(id), vaultID, req.ContactID); err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_life_metric_contact")
	}
	return response.NoContent(c)
}

// RemoveContact godoc
//
//	@Summary		Remove contact from life metric
//	@Description	Remove association between a contact and a life metric
//	@Tags			life-metrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Life Metric ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/lifeMetrics/{id}/contacts/{contact_id} [delete]
func (h *LifeMetricHandler) RemoveContact(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_metric_id", nil)
	}
	contactID := c.Param("contact_id")
	if contactID == "" {
		return response.BadRequest(c, "err.invalid_contact_id", nil)
	}
	if err := h.svc.RemoveContact(uint(id), vaultID, contactID); err != nil {
		if errors.Is(err, services.ErrLifeMetricNotFound) {
			return response.NotFound(c, "err.life_metric_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_life_metric_contact")
	}
	return response.NoContent(c)
}
