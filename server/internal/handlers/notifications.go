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

// List godoc
//
//	@Summary		List notification channels
//	@Description	Return all notification channels for the current user
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.NotificationChannelResponse}
//	@Failure		401	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/notifications [get]
func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	channels, err := h.notificationService.List(userID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_notification_channels")
	}
	return response.OK(c, channels)
}

// Create godoc
//
//	@Summary		Create a notification channel
//	@Description	Create a new notification channel for the current user
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CreateNotificationChannelRequest	true	"Channel details"
//	@Success		201		{object}	response.APIResponse{data=dto.NotificationChannelResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/notifications [post]
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

// Update godoc
//
//	@Summary		Update a notification channel
//	@Description	Update label, content, or preferred time of a notification channel
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer										true	"Channel ID"
//	@Param			request	body		dto.UpdateNotificationChannelRequest		true	"Updated channel details"
//	@Success		200		{object}	response.APIResponse{data=dto.NotificationChannelResponse}
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		422		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/notifications/{id} [put]
func (h *NotificationHandler) Update(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	var req dto.UpdateNotificationChannelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	channel, err := h.notificationService.Update(uint(id), userID, req)
	if err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_notification_channel")
	}
	return response.OK(c, channel)
}

// Toggle godoc
//
//	@Summary		Toggle notification channel active status
//	@Description	Toggle a notification channel on or off
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Channel ID"
//	@Success		200	{object}	response.APIResponse{data=dto.NotificationChannelResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/notifications/{id}/toggle [put]
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

// Delete godoc
//
//	@Summary		Delete a notification channel
//	@Description	Delete a notification channel by ID
//	@Tags			notifications
//	@Security		BearerAuth
//	@Param			id	path	integer	true	"Channel ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/notifications/{id} [delete]
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

// Verify godoc
//
//	@Summary		Verify a notification channel
//	@Description	Verify a notification channel using a verification token
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		integer	true	"Channel ID"
//	@Param			token	path		string	true	"Verification token"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Failure		404		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/settings/notifications/{id}/verify/{token} [get]
func (h *NotificationHandler) Verify(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	token := c.Param("token")
	if err := h.notificationService.Verify(uint(id), userID, token); err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			return response.BadRequest(c, "err.invalid_verification_token", nil)
		}
		return response.InternalError(c, "err.failed_to_verify_notification")
	}
	return response.OK(c, map[string]string{"status": "verified"})
}

// SendTest godoc
//
//	@Summary		Send a test notification
//	@Description	Send a test notification through a channel
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Channel ID"
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/notifications/{id}/test [post]
func (h *NotificationHandler) SendTest(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	if err := h.notificationService.SendTest(uint(id), userID); err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_send_test")
	}
	return response.OK(c, map[string]string{"status": "sent"})
}

// ListLogs godoc
//
//	@Summary		List notification logs
//	@Description	Return notification logs for a specific channel
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		integer	true	"Channel ID"
//	@Success		200	{object}	response.APIResponse{data=[]dto.NotificationLogResponse}
//	@Failure		400	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/notifications/{id}/logs [get]
func (h *NotificationHandler) ListLogs(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_channel_id", nil)
	}
	logs, err := h.notificationService.ListLogs(uint(id), userID)
	if err != nil {
		if errors.Is(err, services.ErrNotificationChannelNotFound) {
			return response.NotFound(c, "err.notification_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_logs")
	}
	return response.OK(c, logs)
}
