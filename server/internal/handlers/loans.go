package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type LoanHandler struct {
	loanService *services.LoanService
}

func NewLoanHandler(loanService *services.LoanService) *LoanHandler {
	return &LoanHandler{loanService: loanService}
}

// List godoc
//
//	@Summary		List loans for a contact
//	@Description	Return all loans belonging to a contact
//	@Tags			loans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.LoanResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/loans [get]
func (h *LoanHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	loans, err := h.loanService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_loans")
	}
	return response.OK(c, loans)
}

// Create godoc
//
//	@Summary		Create a loan
//	@Description	Create a new loan for a contact
//	@Tags			loans
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.CreateLoanRequest	true	"Loan details"
//	@Success		201			{object}	response.APIResponse{data=dto.LoanResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/loans [post]
func (h *LoanHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateLoanRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	loan, err := h.loanService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_loan")
	}
	return response.Created(c, loan)
}

// Update godoc
//
//	@Summary		Update a loan
//	@Description	Update an existing loan
//	@Tags			loans
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			id			path		integer					true	"Loan ID"
//	@Param			request		body		dto.UpdateLoanRequest	true	"Loan details"
//	@Success		200			{object}	response.APIResponse{data=dto.LoanResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/loans/{id} [put]
func (h *LoanHandler) Update(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_loan_id", nil)
	}

	var req dto.UpdateLoanRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	loan, err := h.loanService.Update(uint(id), vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			return response.NotFound(c, "err.loan_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_loan")
	}
	return response.OK(c, loan)
}

// ToggleSettled godoc
//
//	@Summary		Toggle loan settled status
//	@Description	Toggle whether a loan is settled or not
//	@Tags			loans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Loan ID"
//	@Success		200			{object}	response.APIResponse{data=dto.LoanResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/loans/{id}/toggle [put]
func (h *LoanHandler) ToggleSettled(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_loan_id", nil)
	}

	loan, err := h.loanService.ToggleSettled(uint(id), vaultID)
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			return response.NotFound(c, "err.loan_not_found")
		}
		return response.InternalError(c, "err.failed_to_toggle_loan_settled")
	}
	return response.OK(c, loan)
}

// Delete godoc
//
//	@Summary		Delete a loan
//	@Description	Delete a loan
//	@Tags			loans
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Param			id			path		integer	true	"Loan ID"
//	@Success		204			{object}	nil
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/loans/{id} [delete]
func (h *LoanHandler) Delete(c echo.Context) error {
	vaultID := c.Param("vault_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_loan_id", nil)
	}

	if err := h.loanService.Delete(uint(id), vaultID); err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			return response.NotFound(c, "err.loan_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_loan")
	}
	return response.NoContent(c)
}
