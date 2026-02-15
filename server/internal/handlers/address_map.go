package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

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
