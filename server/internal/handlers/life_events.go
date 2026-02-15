package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type LifeEventHandler struct {
	lifeEventService *services.LifeEventService
}

func NewLifeEventHandler(lifeEventService *services.LifeEventService) *LifeEventHandler {
	return &LifeEventHandler{lifeEventService: lifeEventService}
}

func (h *LifeEventHandler) ListTimelineEvents(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	events, meta, err := h.lifeEventService.ListTimelineEvents(contactID, vaultID, page, perPage)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_timeline_events")
	}
	return response.Paginated(c, events, meta)
}

func (h *LifeEventHandler) CreateTimelineEvent(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	var req dto.CreateTimelineEventRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	event, err := h.lifeEventService.CreateTimelineEvent(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_timeline_event")
	}
	return response.Created(c, event)
}

func (h *LifeEventHandler) AddLifeEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	var req dto.CreateLifeEventRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	event, err := h.lifeEventService.AddLifeEvent(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrTimelineEventNotFound) {
			return response.NotFound(c, "err.timeline_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_life_event")
	}
	return response.Created(c, event)
}

func (h *LifeEventHandler) UpdateLifeEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	lifeEventID, err := strconv.ParseUint(c.Param("lifeEventId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_event_id", nil)
	}
	var req dto.UpdateLifeEventRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	event, err := h.lifeEventService.UpdateLifeEvent(uint(id), uint(lifeEventID), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLifeEventNotFound) {
			return response.NotFound(c, "err.life_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_life_event")
	}
	return response.OK(c, event)
}

func (h *LifeEventHandler) DeleteTimelineEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	if err := h.lifeEventService.DeleteTimelineEvent(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrTimelineEventNotFound) {
			return response.NotFound(c, "err.timeline_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_timeline_event")
	}
	return response.NoContent(c)
}

func (h *LifeEventHandler) ToggleTimelineEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	if err := h.lifeEventService.ToggleTimelineEvent(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrTimelineEventNotFound) {
			return response.NotFound(c, "err.timeline_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_timeline_event")
	}
	return response.NoContent(c)
}

func (h *LifeEventHandler) ToggleLifeEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	lifeEventID, err := strconv.ParseUint(c.Param("lifeEventId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_event_id", nil)
	}
	if err := h.lifeEventService.ToggleLifeEvent(uint(id), uint(lifeEventID), vaultID); err != nil {
		if errors.Is(err, services.ErrTimelineEventNotFound) {
			return response.NotFound(c, "err.timeline_event_not_found")
		}
		if errors.Is(err, services.ErrLifeEventNotFound) {
			return response.NotFound(c, "err.life_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_life_event")
	}
	return response.NoContent(c)
}

func (h *LifeEventHandler) DeleteLifeEvent(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_timeline_event_id", nil)
	}
	lifeEventID, err := strconv.ParseUint(c.Param("lifeEventId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_life_event_id", nil)
	}
	if err := h.lifeEventService.DeleteLifeEvent(uint(id), uint(lifeEventID), vaultID); err != nil {
		if errors.Is(err, services.ErrLifeEventNotFound) {
			return response.NotFound(c, "err.life_event_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_life_event")
	}
	return response.NoContent(c)
}
