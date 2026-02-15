package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type CurrencyHandler struct {
	currencyService *services.CurrencyService
}

func NewCurrencyHandler(svc *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{currencyService: svc}
}

func (h *CurrencyHandler) List(c echo.Context) error {
	currencies, err := h.currencyService.List()
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_currencies")
	}
	return response.OK(c, currencies)
}
