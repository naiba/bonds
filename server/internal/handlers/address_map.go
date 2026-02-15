package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

// GetMapImage godoc
//
//	@Summary		Get map image for an address
//	@Description	Redirect to a static map image URL for the given address
//	@Tags			addresses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Address ID"
//	@Param			width		path		integer	true	"Image width"
//	@Param			height		path		integer	true	"Image height"
//	@Success		307			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/addresses/{id}/image/{width}/{height} [get]
func (h *AddressHandler) GetMapImage(c echo.Context) error {
	vaultID := c.Param("vault_id")
	contactID := c.Param("contact_id")
	addressID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_address_id", nil)
	}
	width, err := strconv.Atoi(c.Param("width"))
	if err != nil || width <= 0 || width > 2048 {
		width = 300
	}
	height, err := strconv.Atoi(c.Param("height"))
	if err != nil || height <= 0 || height > 2048 {
		height = 300
	}
	mapURL, err := h.addressService.GetMapImageURL(uint(addressID), contactID, vaultID, width, height)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrAddressNotFound) {
			return response.NotFound(c, "err.address_not_found")
		}
		return response.BadRequest(c, "err.address_has_no_coordinates", nil)
	}
	return c.Redirect(http.StatusTemporaryRedirect, mapURL)
}
