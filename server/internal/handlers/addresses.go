package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type AddressHandler struct {
	addressService *services.AddressService
}

func NewAddressHandler(addressService *services.AddressService) *AddressHandler {
	return &AddressHandler{addressService: addressService}
}

func (h *AddressHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	addresses, err := h.addressService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_addresses")
	}
	return response.OK(c, addresses)
}

func (h *AddressHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateAddressRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	address, err := h.addressService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_address")
	}
	return response.Created(c, address)
}

func (h *AddressHandler) Update(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_address_id", nil)
	}

	var req dto.UpdateAddressRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	address, err := h.addressService.Update(uint(id), contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrAddressNotFound) {
			return response.NotFound(c, "err.address_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_address")
	}
	return response.OK(c, address)
}

func (h *AddressHandler) Delete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_address_id", nil)
	}

	if err := h.addressService.Delete(uint(id), contactID, vaultID); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrAddressNotFound) {
			return response.NotFound(c, "err.address_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_address")
	}
	return response.NoContent(c)
}
