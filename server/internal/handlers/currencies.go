package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.CurrencyResponse

type CurrencyHandler struct {
	currencyService *services.CurrencyService
}

func NewCurrencyHandler(svc *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{currencyService: svc}
}

// List godoc
//
//	@Summary		List currencies
//	@Description	Return all available currencies
//	@Tags			currencies
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse{data=[]dto.CurrencyResponse}
//	@Failure		500	{object}	response.APIResponse
//	@Router			/currencies [get]
func (h *CurrencyHandler) List(c echo.Context) error {
	currencies, err := h.currencyService.List()
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_currencies")
	}
	return response.OK(c, currencies)
}

// Toggle godoc
//
//	@Summary		Toggle currency
//	@Description	Toggle a currency's active status for the account
//	@Tags			personalize
//	@Produce		json
//	@Security		BearerAuth
//	@Param			currencyId	path	integer	true	"Currency ID"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/settings/personalize/currencies/{currencyId}/toggle [put]
func (h *CurrencyHandler) Toggle(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	currencyID, err := strconv.ParseUint(c.Param("currencyId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_currency_id", nil)
	}
	if err := h.currencyService.Toggle(accountID, uint(currencyID)); err != nil {
		if errors.Is(err, services.ErrCurrencyNotFound) {
			return response.NotFound(c, "err.currency_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_currency")
	}
	return response.OK(c, nil)
}

// EnableAll godoc
//
//	@Summary		Enable all currencies
//	@Description	Enable all currencies for the account
//	@Tags			personalize
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/currencies/enable-all [post]
func (h *CurrencyHandler) EnableAll(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	if err := h.currencyService.EnableAll(accountID); err != nil {
		return response.InternalError(c, "err.failed_to_enable_currencies")
	}
	return response.Created(c, nil)
}

// DisableAll godoc
//
//	@Summary		Disable all currencies
//	@Description	Disable all currencies for the account
//	@Tags			personalize
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/settings/personalize/currencies/disable-all [delete]
func (h *CurrencyHandler) DisableAll(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	if err := h.currencyService.DisableAll(accountID); err != nil {
		return response.InternalError(c, "err.failed_to_disable_currencies")
	}
	return response.OK(c, nil)
}
