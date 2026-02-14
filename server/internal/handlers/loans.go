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

func (h *LoanHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	loans, err := h.loanService.List(contactID, vaultID)
	if err != nil {
		return response.InternalError(c, "err.failed_to_list_loans")
	}
	return response.OK(c, loans)
}

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
		return response.InternalError(c, "err.failed_to_create_loan")
	}
	return response.Created(c, loan)
}

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
