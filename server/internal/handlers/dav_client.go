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

var _ dto.DavSubscriptionResponse   // type anchor for swag
var _ dto.TestDavConnectionResponse // type anchor for swag
var _ dto.TriggerSyncResponse       // type anchor for swag
var _ dto.DavSyncLogResponse        // type anchor for swag

type DavClientHandler struct {
	clientService *services.DavClientService
	syncService   *services.DavSyncService
}

func NewDavClientHandler(clientService *services.DavClientService, syncService *services.DavSyncService) *DavClientHandler {
	return &DavClientHandler{
		clientService: clientService,
		syncService:   syncService,
	}
}

// Create godoc
//
//	@Summary		Create a DAV subscription
//	@Description	Create a new CardDAV subscription for the vault
//	@Tags			dav-subscriptions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			request		body		dto.CreateDavSubscriptionRequest	true	"Subscription details"
//	@Success		201			{object}	response.APIResponse{data=dto.DavSubscriptionResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions [post]
func (h *DavClientHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")
	userID := middleware.GetUserID(c)

	var req dto.CreateDavSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	sub, err := h.clientService.Create(vaultID, userID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_subscription")
	}
	return response.Created(c, sub)
}

// List godoc
//
//	@Summary		List DAV subscriptions
//	@Description	Return all CardDAV subscriptions for a vault
//	@Tags			dav-subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.DavSubscriptionResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions [get]
func (h *DavClientHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")

	subs, err := h.clientService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_subscriptions")
	}
	return response.OK(c, subs)
}

// Get godoc
//
//	@Summary		Get a DAV subscription
//	@Description	Return a single CardDAV subscription
//	@Tags			dav-subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			sub_id		path		string	true	"Subscription ID"
//	@Success		200			{object}	response.APIResponse{data=dto.DavSubscriptionResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/{sub_id} [get]
func (h *DavClientHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	subID := c.Param("sub_id")

	sub, err := h.clientService.Get(subID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "err.subscription_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_subscription")
	}
	return response.OK(c, sub)
}

// Update godoc
//
//	@Summary		Update a DAV subscription
//	@Description	Update an existing CardDAV subscription
//	@Tags			dav-subscriptions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string								true	"Vault ID"
//	@Param			sub_id		path		string								true	"Subscription ID"
//	@Param			request		body		dto.UpdateDavSubscriptionRequest	true	"Updated details"
//	@Success		200			{object}	response.APIResponse{data=dto.DavSubscriptionResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/{sub_id} [put]
func (h *DavClientHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	subID := c.Param("sub_id")

	var req dto.UpdateDavSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	sub, err := h.clientService.Update(subID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "err.subscription_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_subscription")
	}
	return response.OK(c, sub)
}

// Delete godoc
//
//	@Summary		Delete a DAV subscription
//	@Description	Permanently delete a CardDAV subscription
//	@Tags			dav-subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			sub_id		path	string	true	"Subscription ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/{sub_id} [delete]
func (h *DavClientHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	subID := c.Param("sub_id")

	if err := h.clientService.Delete(subID, vaultID); err != nil {
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "err.subscription_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_subscription")
	}
	return response.NoContent(c)
}

// TestConnection godoc
//
//	@Summary		Test DAV connection
//	@Description	Test connectivity to a CardDAV server
//	@Tags			dav-subscriptions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string							true	"Vault ID"
//	@Param			request		body		dto.TestDavConnectionRequest	true	"Connection details"
//	@Success		200			{object}	response.APIResponse{data=dto.TestDavConnectionResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/test [post]
func (h *DavClientHandler) TestConnection(c echo.Context) error {
	var req dto.TestDavConnectionRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	result, err := h.syncService.TestConnection(req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_test_connection")
	}
	return response.OK(c, result)
}

// TriggerSync godoc
//
//	@Summary		Trigger DAV sync
//	@Description	Manually trigger a sync for a CardDAV subscription
//	@Tags			dav-subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			sub_id		path		string	true	"Subscription ID"
//	@Success		200			{object}	response.APIResponse{data=dto.TriggerSyncResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/{sub_id}/sync [post]
func (h *DavClientHandler) TriggerSync(c echo.Context) error {
	vaultID := c.Param("vault_id")
	subID := c.Param("sub_id")

	result, err := h.syncService.SyncSubscription(c.Request().Context(), subID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "err.subscription_not_found")
		}
		return response.InternalError(c, "err.failed_to_sync")
	}
	return response.OK(c, result)
}

// GetSyncLogs godoc
//
//	@Summary		Get DAV sync logs
//	@Description	Return paginated sync logs for a subscription
//	@Tags			dav-subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			sub_id		path		string	true	"Subscription ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.DavSyncLogResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/dav/subscriptions/{sub_id}/logs [get]
func (h *DavClientHandler) GetSyncLogs(c echo.Context) error {
	vaultID := c.Param("vault_id")
	subID := c.Param("sub_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	logs, meta, err := h.syncService.GetSyncLogs(subID, vaultID, page, perPage)
	if err != nil {
		if errors.Is(err, services.ErrSubscriptionNotFound) {
			return response.NotFound(c, "err.subscription_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_sync_logs")
	}
	return response.Paginated(c, logs, meta)
}
