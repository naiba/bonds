package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

var _ dto.CompanyResponse

type CompanyHandler struct {
	companyService    *services.CompanyService
	contactJobService *services.ContactJobService
}

func NewCompanyHandler(companyService *services.CompanyService, contactJobService *services.ContactJobService) *CompanyHandler {
	return &CompanyHandler{companyService: companyService, contactJobService: contactJobService}
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
	contactID := c.Param("contact_id")
	companies, err := h.companyService.ListForContact(contactID, vaultID)
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

// Create godoc
//
//	@Summary		Create a company
//	@Description	Create a new company in a vault
//	@Tags			companies
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			request		body		dto.CreateCompanyRequest		true	"Company details"
//	@Success		201			{object}	response.APIResponse{data=dto.CompanyResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies [post]
func (h *CompanyHandler) Create(c echo.Context) error {
	vaultID := c.Param("vault_id")

	var req dto.CreateCompanyRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	company, err := h.companyService.Create(vaultID, req)
	if err != nil {
		return response.InternalError(c, "err.failed_to_create_company")
	}
	return response.Created(c, company)
}

// Update godoc
//
//	@Summary		Update a company
//	@Description	Update an existing company
//	@Tags			companies
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			id			path		integer						true	"Company ID"
//	@Param			request		body		dto.UpdateCompanyRequest		true	"Company details"
//	@Success		200			{object}	response.APIResponse{data=dto.CompanyResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies/{id} [put]
func (h *CompanyHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_company_id", nil)
	}

	var req dto.UpdateCompanyRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	company, err := h.companyService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_company")
	}
	return response.OK(c, company)
}

// Delete godoc
//
//	@Summary		Delete a company
//	@Description	Permanently delete a company and unlink associated contacts
//	@Tags			companies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Company ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies/{id} [delete]
func (h *CompanyHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_company_id", nil)
	}

	if err := h.companyService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_company")
	}
	return response.NoContent(c)
}

// AddEmployee godoc
//
//	@Summary		Add employee to company
//	@Description	Add a contact as an employee of a company
//	@Tags			companies
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			id			path		integer					true	"Company ID"
//	@Param			request		body		dto.AddEmployeeRequest	true	"Employee details"
//	@Success		201			{object}	response.APIResponse{data=dto.ContactJobResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies/{id}/employees [post]
func (h *CompanyHandler) AddEmployee(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_company_id", nil)
	}

	var req dto.AddEmployeeRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	job, err := h.contactJobService.AddEmployee(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_add_employee")
	}
	return response.Created(c, job)
}

// RemoveEmployee godoc
//
//	@Summary		Remove employee from company
//	@Description	Remove a contact from a company's employee list
//	@Tags			companies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			id			path	integer	true	"Company ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/companies/{id}/employees/{contact_id} [delete]
func (h *CompanyHandler) RemoveEmployee(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_company_id", nil)
	}
	contactID := c.Param("contact_id")

	if err := h.contactJobService.RemoveEmployee(uint(id), vaultID, contactID); err != nil {
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		if errors.Is(err, services.ErrContactJobNotFound) {
			return response.NotFound(c, "err.employee_not_found")
		}
		return response.InternalError(c, "err.failed_to_remove_employee")
	}
	return response.NoContent(c)
}
