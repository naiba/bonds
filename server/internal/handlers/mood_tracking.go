package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type MoodTrackingHandler struct {
	moodTrackingService *services.MoodTrackingService
}

func NewMoodTrackingHandler(moodTrackingService *services.MoodTrackingService) *MoodTrackingHandler {
	return &MoodTrackingHandler{moodTrackingService: moodTrackingService}
}

func (h *MoodTrackingHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	var req dto.CreateMoodTrackingEventRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	event, err := h.moodTrackingService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_mood_tracking_event")
	}
	return response.Created(c, event)
}

func (h *MoodTrackingHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	events, err := h.moodTrackingService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_mood_tracking_events")
	}
	return response.OK(c, events)
}
