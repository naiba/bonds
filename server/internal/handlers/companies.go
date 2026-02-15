package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.CompanyResponse

type CompanyHandler struct {
	companyService *services.CompanyService
}

func NewCompanyHandler(companyService *services.CompanyService) *CompanyHandler {
	return &CompanyHandler{companyService: companyService}
}

// List godoc
//
//	@Summary		List companies
//	@Description	Return all companies for a vault
//	@Tags			companies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.CompanyResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies [get]
func (h *CompanyHandler) List(c echo.Context) error {
	vaultID := c.Param("vault_id")
	companies, err := h.companyService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_companies")
	}
	return response.OK(c, companies)
}

// ListForContact godoc
//
//	@Summary		List companies for a contact
//	@Description	Return all companies associated with a contact
//	@Tags			companies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.CompanyResponse}
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/companies/list [get]
func (h *CompanyHandler) ListForContact(c echo.Context) error {
	vaultID := c.Param("vault_id")
	companies, err := h.companyService.List(vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_companies")
	}
	return response.OK(c, companies)
}

// Get godoc
//
//	@Summary		Get a company
//	@Description	Return a single company by ID
//	@Tags			companies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			id			path		integer	true	"Company ID"
//	@Success		200			{object}	response.APIResponse{data=dto.CompanyResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies/{id} [get]
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
