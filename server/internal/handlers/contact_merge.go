package handlers

import (
	"errors"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

// Merge godoc
// @Summary Merge contacts
// @Description Merge 2 to 50 contacts in one vault, preserving the target identity with explicit conflict choices and no profile snapshots. Sources are soft-deleted. Requires Editor permission.
// @Tags contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vault_id path string true "Vault ID"
// @Param request body dto.MergeContactsRequest true "Target and ordered sources"
// @Success 200 {object} response.APIResponse{data=dto.ContactResponse}
// @Failure 400,401,403,404,409,422,500 {object} response.APIResponse
// @Router /vaults/{vault_id}/contacts/merge [post]
func (h *ContactHandler) Merge(c *echo.Context) error {
	var req dto.MergeContactsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	contact, err := h.contactService.MergeContacts(c.Param("vault_id"), middleware.GetUserID(c), req)
	switch {
	case errors.Is(err, services.ErrContactNotFound):
		return response.NotFound(c, "err.contact_not_found")
	case errors.Is(err, services.ErrVaultForbidden), errors.Is(err, services.ErrInsufficientPerm):
		return response.Forbidden(c, "err.vault_forbidden")
	case errors.Is(err, services.ErrContactCannotBeDeleted):
		return response.Conflict(c, "err.contact_cannot_be_deleted")
	case errors.Is(err, services.ErrContactMergeBlocked):
		return response.Conflict(c, "err.contact_merge_blocked")
	case errors.Is(err, services.ErrContactMergeReviewChanged):
		return response.Conflict(c, "err.contact_merge_review_changed")
	case errors.Is(err, services.ErrContactMergeSelection):
		return response.BadRequest(c, "err.contact_merge_selection", nil)
	case err != nil:
		return response.InternalError(c, "err.failed_to_merge_contacts")
	default:
		return response.OK(c, contact)
	}
}

// PreviewMerge godoc
// @Summary Review contact merge conflicts and effects
// @Tags contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vault_id path string true "Vault ID"
// @Param request body dto.MergeContactsRequest true "Target and ordered sources"
// @Success 200 {object} response.APIResponse{data=dto.ContactMergePreview}
// @Failure 400,401,403,404,500 {object} response.APIResponse
// @Router /vaults/{vault_id}/contacts/merge/preview [post]
func (h *ContactHandler) PreviewMerge(c *echo.Context) error {
	var req dto.MergeContactsRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}
	preview, err := h.contactService.PreviewContactMerge(c.Param("vault_id"), middleware.GetUserID(c), req)
	switch {
	case errors.Is(err, services.ErrContactNotFound):
		return response.NotFound(c, "err.contact_not_found")
	case errors.Is(err, services.ErrVaultForbidden), errors.Is(err, services.ErrInsufficientPerm):
		return response.Forbidden(c, "err.vault_forbidden")
	case errors.Is(err, services.ErrContactMergeSelection):
		return response.BadRequest(c, "err.contact_merge_selection", nil)
	case err != nil:
		return response.InternalError(c, "err.failed_to_merge_contacts")
	default:
		return response.OK(c, preview)
	}
}
