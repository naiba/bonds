package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type SearchHandler struct {
	searchService *services.SearchService
}

func NewSearchHandler(searchService *services.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

func (h *SearchHandler) Search(c echo.Context) error {
	vaultID := c.Param("vault_id")
	query := c.QueryParam("q")
	if query == "" {
		return response.BadRequest(c, "err.search_query_required", nil)
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	result, err := h.searchService.Search(vaultID, query, page, perPage)
	if err != nil {
		return response.InternalError(c, "err.search_failed")
	}
	return response.OK(c, result)
}
