package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactSortHandler struct {
	contactSortService *services.ContactSortService
}

func NewContactSortHandler(contactSortService *services.ContactSortService) *ContactSortHandler {
	return &ContactSortHandler{contactSortService: contactSortService}
}

// UpdateSort godoc
//
//	@Summary		Update contact sort order
//	@Description	Update the sort order for contacts in a vault
//	@Tags			contacts
//	@Accept			json
//	@Security		BearerAuth
//	@Param			vault_id	path	string							true	"Vault ID"
//	@Param			request		body	dto.UpdateContactSortRequest	true	"Sort details"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/sort [put]
func (h *ContactSortHandler) UpdateSort(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req dto.UpdateContactSortRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	if err := h.contactSortService.UpdateSort(userID, req); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			return response.NotFound(c, "err.user_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_sort_order")
	}
	return response.NoContent(c)
}
