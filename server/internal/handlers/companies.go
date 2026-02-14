package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type CompanyHandler struct {
	companyService *services.CompanyService
}

func NewCompanyHandler(companyService *services.CompanyService) *CompanyHandler {
	return &CompanyHandler{companyService: companyService}
}

func (h *CompanyHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	companies, err := h.companyService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_companies")
	}
	return response.OK(c, companies)
}

func (h *CompanyHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_company_id", nil)
	}
	company, err := h.companyService.Get(uint(id))
	if err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		return response.InternalError(c, "err.failed_to_get_company")
	}
	return response.OK(c, company)
}
