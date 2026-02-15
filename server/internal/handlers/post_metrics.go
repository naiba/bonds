package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type PostMetricHandler struct {
	svc *services.PostMetricService
}

func NewPostMetricHandler(svc *services.PostMetricService) *PostMetricHandler {
	return &PostMetricHandler{svc: svc}
}

func (h *PostMetricHandler) Create(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	var req dto.CreatePostMetricRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	metric, err := h.svc.Create(uint(postID), uint(journalID), req)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrJournalMetricNotFound) {
			return response.NotFound(c, "err.journal_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_post_metric")
	}
	return response.Created(c, metric)
}

func (h *PostMetricHandler) Delete(c echo.Context) error {
	journalID, err := strconv.ParseUint(c.Param("journal_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_journal_id", nil)
	}
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_post_id", nil)
	}
	metricID, err := strconv.ParseUint(c.Param("metricId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_metric_id", nil)
	}
	if err := h.svc.Delete(uint(metricID), uint(postID), uint(journalID)); err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			return response.NotFound(c, "err.post_not_found")
		}
		if errors.Is(err, services.ErrPostMetricNotFound) {
			return response.NotFound(c, "err.post_metric_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_post_metric")
	}
	return response.NoContent(c)
}
