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

// ListTimelineEvents godoc
//
//	@Summary		List timeline events for a contact
//	@Description	Return paginated timeline events belonging to a contact
//	@Tags			life-events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.TimelineEventResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents [get]
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

// CreateTimelineEvent godoc
//
//	@Summary		Create a timeline event
//	@Description	Create a new timeline event for a contact
//	@Tags			life-events
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			contact_id	path		string								true	"Contact ID"
//	@Param			request		body		dto.CreateTimelineEventRequest		true	"Timeline event details"
//	@Success		201			{object}	response.APIResponse{data=dto.TimelineEventResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents [post]
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

// AddLifeEvent godoc
//
//	@Summary		Add a life event to a timeline event
//	@Description	Create a new life event under a timeline event
//	@Tags			life-events
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			contact_id	path		string							true	"Contact ID"
//	@Param			id			path		integer							true	"Timeline Event ID"
//	@Param			request		body		dto.CreateLifeEventRequest		true	"Life event details"
//	@Success		201			{object}	response.APIResponse{data=dto.LifeEventResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id}/lifeEvents [post]
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

// UpdateLifeEvent godoc
//
//	@Summary		Update a life event
//	@Description	Update an existing life event
//	@Tags			life-events
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			contact_id	path		string							true	"Contact ID"
//	@Param			id			path		integer							true	"Timeline Event ID"
//	@Param			lifeEventId	path		integer							true	"Life Event ID"
//	@Param			request		body		dto.UpdateLifeEventRequest		true	"Life event details"
//	@Success		200			{object}	response.APIResponse{data=dto.LifeEventResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id}/lifeEvents/{lifeEventId} [put]
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

// DeleteTimelineEvent godoc
//
//	@Summary		Delete a timeline event
//	@Description	Delete a timeline event and all its life events
//	@Tags			life-events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Timeline Event ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id} [delete]
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

// ToggleTimelineEvent godoc
//
//	@Summary		Toggle timeline event collapsed state
//	@Description	Toggle whether a timeline event is collapsed or expanded
//	@Tags			life-events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Timeline Event ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id}/toggle [put]
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

// ToggleLifeEvent godoc
//
//	@Summary		Toggle life event collapsed state
//	@Description	Toggle whether a life event is collapsed or expanded
//	@Tags			life-events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Timeline Event ID"
//	@Param			lifeEventId	path		integer	true	"Life Event ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id}/lifeEvents/{lifeEventId}/toggle [put]
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

// DeleteLifeEvent godoc
//
//	@Summary		Delete a life event
//	@Description	Delete a life event from a timeline event
//	@Tags			life-events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Timeline Event ID"
//	@Param			lifeEventId	path		integer	true	"Life Event ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/timelineEvents/{id}/lifeEvents/{lifeEventId} [delete]
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
