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

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	channels, err := h.notificationService.List(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_notification_channels")
	}
	return response.OK(c, channels)
}

func (h *NotificationHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var req dto.CreateNotificationChannelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	channel, err := h.notificationService.Create(userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_notification_channel")
	}
	return response.Created(c, channel)
}

func (h *NotificationHandler) Toggle(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	channel, err := h.notificationService.Toggle(uint(id), userID)
	if err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_notification_channel")
	}
	return response.OK(c, channel)
}

func (h *NotificationHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	if err := h.notificationService.Delete(uint(id), userID); err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_notification_channel")
	}
	return response.NoContent(c)
}
