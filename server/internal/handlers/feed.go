package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.FeedItemResponse

type FeedHandler struct {
	feedService *services.FeedService
}

func NewFeedHandler(feedService *services.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

// GetContactFeed godoc
//
//	@Summary		Get contact feed
//	@Description	Return paginated feed items for a contact
//	@Tags			feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.FeedItemResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/feed [get]
func (h *FeedHandler) GetContactFeed(c echo.Context) error {
	contactID := c.Param("contact_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	items, meta, err := h.feedService.ListContactFeed(contactID, page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_feed")
	}
	return response.Paginated(c, items, meta)
}

// Get godoc
//
//	@Summary		Get vault feed
//	@Description	Return paginated feed items for a vault
//	@Tags			feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse{data=[]dto.FeedItemResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/feed [get]
func (h *FeedHandler) Get(c echo.Context) error {
	vaultID := c.Param("vault_id")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	items, meta, err := h.feedService.GetFeed(vaultID, page, perPage)
	if err != nil {
		return response.InternalError(c, "err.failed_to_get_feed")
	}
	return response.Paginated(c, items, meta)
}
