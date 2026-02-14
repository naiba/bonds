package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type FeedHandler struct {
	feedService *services.FeedService
}

func NewFeedHandler(feedService *services.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

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
