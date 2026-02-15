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

// Search godoc
//
//	@Summary		Search contacts and notes
//	@Description	Full-text search across contacts and notes in a vault
//	@Tags			search
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			q			query		string	true	"Search query"
//	@Param			page		query		integer	false	"Page number"
//	@Param			per_page	query		integer	false	"Items per page"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/search [get]
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
