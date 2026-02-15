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

func (h *LifeMetricHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	metrics, err := h.svc.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_life_metrics")
	}
	return response.OK(c, metrics)
}

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
