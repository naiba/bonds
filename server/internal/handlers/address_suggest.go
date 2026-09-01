package handlers

import (
	"errors"

	"github.com/labstack/echo/v5"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.AddressSuggestionsResponse

// Suggest godoc
//
//	@Summary		Suggest addresses
//	@Description	Look up candidate addresses for a partial query, for address autocomplete
//	@Tags			addresses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			q			query		string	true	"Partial address"
//	@Success		200			{object}	response.APIResponse{data=dto.AddressSuggestionsResponse}
//	@Failure		429			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/addresses/suggest [get]
func (h *AddressHandler) Suggest(c *echo.Context) error {
	suggestions, enabled, err := h.addressService.Suggest(c.QueryParam("q"), 0)
	if err != nil {
		if errors.Is(err, services.ErrGeocoderBusy) {
			// The provider's rate limit, not a failure — the caller should slow
			// down and try again rather than treat lookup as broken.
			return response.TooManyRequests(c, "err.address_lookup_busy")
		}
		return response.InternalError(c, "err.failed_to_suggest_addresses")
	}
	return response.OK(c, dto.AddressSuggestionsResponse{
		Enabled:     enabled,
		Suggestions: suggestions,
		Attribution: h.addressService.Attribution(),
	})
}
