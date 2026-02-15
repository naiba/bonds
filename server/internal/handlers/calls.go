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

type CallHandler struct {
	callService *services.CallService
}

func NewCallHandler(callService *services.CallService) *CallHandler {
	return &CallHandler{callService: callService}
}

// List godoc
//
//	@Summary		List calls
//	@Description	Return paginated calls for a contact
//	@Tags			calls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.CallResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/calls [get]
func (h *CallHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	calls, meta, err := h.callService.List(contactID, vaultID, page, perPage)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_calls")
	}
	return response.Paginated(c, calls, meta)
}

// Create godoc
//
//	@Summary		Create a call
//	@Description	Log a new call for a contact
//	@Tags			calls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreateCallRequest	true	"Call details"
//	@Success		201			{object}	response.APIResponse{data=dto.CallResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/calls [post]
func (h *CallHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateCallRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	call, err := h.callService.Create(contactID, vaultID, userID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_call")
	}
	return response.Created(c, call)
}

// Update godoc
//
//	@Summary		Update a call
//	@Description	Update an existing call record
//	@Tags			calls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Call ID"
//	@Param			request		body		dto.UpdateCallRequest	true	"Call details"
//	@Success		200			{object}	response.APIResponse{data=dto.CallResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/calls/{id} [put]
func (h *CallHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_id", nil)
	}

	var req dto.UpdateCallRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	call, err := h.callService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrCallNotFound) {
			return response.NotFound(c, "err.call_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_call")
	}
	return response.OK(c, call)
}

// Delete godoc
//
//	@Summary		Delete a call
//	@Description	Permanently delete a call record
//	@Tags			calls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			id			path	integer	true	"Call ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/calls/{id} [delete]
func (h *CallHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_call_id", nil)
	}

	if err := h.callService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrCallNotFound) {
			return response.NotFound(c, "err.call_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_call")
	}
	return response.NoContent(c)
}
