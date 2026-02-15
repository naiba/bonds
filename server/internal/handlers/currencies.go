package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
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
