package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

func (h *QuickFactHandler) Toggle(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	newValue, err := h.quickFactService.ToggleShowQuickFacts(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_quick_facts")
	}
	return response.OK(c, map[string]bool{"show_quick_facts": newValue})
}
