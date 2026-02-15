package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ReminderHandler struct {
	reminderService *services.ReminderService
}

func NewReminderHandler(reminderService *services.ReminderService) *ReminderHandler {
	return &ReminderHandler{reminderService: reminderService}
}

// List godoc
//
//	@Summary		List reminders
//	@Description	Return all reminders for a contact
//	@Tags			reminders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ReminderResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/reminders [get]
func (h *ReminderHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	reminders, err := h.reminderService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_reminders")
	}
	return response.OK(c, reminders)
}

// Create godoc
//
//	@Summary		Create a reminder
//	@Description	Create a new reminder for a contact
//	@Tags			reminders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			request		body		dto.CreateReminderRequest	true	"Reminder details"
//	@Success		201			{object}	response.APIResponse{data=dto.ReminderResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/reminders [post]
func (h *ReminderHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateReminderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	reminder, err := h.reminderService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_reminder")
	}
	return response.Created(c, reminder)
}

// Update godoc
//
//	@Summary		Update a reminder
//	@Description	Update an existing reminder
//	@Tags			reminders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			id			path		integer						true	"Reminder ID"
//	@Param			request		body		dto.UpdateReminderRequest	true	"Reminder details"
//	@Success		200			{object}	response.APIResponse{data=dto.ReminderResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/reminders/{id} [put]
func (h *ReminderHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_reminder_id", nil)
	}

	var req dto.UpdateReminderRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	reminder, err := h.reminderService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrReminderNotFound) {
			return response.NotFound(c, "err.reminder_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_reminder")
	}
	return response.OK(c, reminder)
}

// Delete godoc
//
//	@Summary		Delete a reminder
//	@Description	Permanently delete a reminder
//	@Tags			reminders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Reminder ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/reminders/{id} [delete]
func (h *ReminderHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_reminder_id", nil)
	}

	if err := h.reminderService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrReminderNotFound) {
			return response.NotFound(c, "err.reminder_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_reminder")
	}
	return response.NoContent(c)
}
