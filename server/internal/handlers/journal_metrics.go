package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type JournalMetricHandler struct {
	svc *services.JournalMetricService
}

func NewJournalMetricHandler(svc *services.JournalMetricService) *JournalMetricHandler {
	return &JournalMetricHandler{svc: svc}
}

func (h *JournalMetricHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	metrics, err := h.svc.List(uint(journalID), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_journal_metrics")
	}
	return response.OK(c, metrics)
}

func (h *JournalMetricHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	var req dto.CreateJournalMetricRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	metric, err := h.svc.Create(uint(journalID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_journal_metric")
	}
	return response.Created(c, metric)
}

func (h *JournalMetricHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_metric_id", nil)
	}
	if err := h.svc.Delete(uint(id), uint(journalID), vaultID); err != nil {
		if errors.Is(err, services.ErrJournalNotFound) {
			return response.NotFound(c, "err.journal_not_found")
		}
		if errors.Is(err, services.ErrJournalMetricNotFound) {
			return response.NotFound(c, "err.journal_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_journal_metric")
	}
	return response.NoContent(c)
}
