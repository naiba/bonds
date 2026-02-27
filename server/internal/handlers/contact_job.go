package handlers

import (
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type ContactJobHandler struct {
	contactJobService *services.ContactJobService
}

func NewContactJobHandler(contactJobService *services.ContactJobService) *ContactJobHandler {
	return &ContactJobHandler{contactJobService: contactJobService}
}

// List godoc
//
//	@Summary		List contact jobs
//	@Description	Return all jobs for a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse{data=[]dto.ContactJobResponse}
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobs [get]
func (h *ContactJobHandler) List(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	jobs, err := h.contactJobService.List(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_list_jobs")
	}
	return response.OK(c, jobs)
}

// Create godoc
//
//	@Summary		Create contact job
//	@Description	Add a new job for a contact
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			request		body		dto.CreateContactJobRequest	true	"Job details"
//	@Success		201			{object}	response.APIResponse{data=dto.ContactJobResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobs [post]
func (h *ContactJobHandler) Create(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.CreateContactJobRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	job, err := h.contactJobService.Create(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		return response.InternalError(c, "err.failed_to_create_job")
	}
	return response.Created(c, job)
}

// Update godoc
//
//	@Summary		Update contact job
//	@Description	Update a specific job for a contact
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string						true	"Vault ID"
//	@Param			contact_id	path		string						true	"Contact ID"
//	@Param			job_id		path		integer						true	"Job ID"
//	@Param			request		body		dto.UpdateContactJobRequest	true	"Job details"
//	@Success		200			{object}	response.APIResponse{data=dto.ContactJobResponse}
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		422			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobs/{job_id} [put]
func (h *ContactJobHandler) UpdateJob(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	jobID, err := strconv.ParseUint(c.Param("job_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_job_id", nil)
	}

	var req dto.UpdateContactJobRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := validateRequest(req); err != nil {
		return response.ValidationError(c, map[string]string{"validation": err.Error()})
	}

	job, err := h.contactJobService.Update(contactID, vaultID, uint(jobID), req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactJobNotFound) {
			return response.NotFound(c, "err.job_not_found")
		}
		if errors.Is(err, services.ErrCompanyNotFound) {
			return response.NotFound(c, "err.company_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_job")
	}
	return response.OK(c, job)
}

// DeleteJob godoc
//
//	@Summary		Delete contact job
//	@Description	Delete a specific job for a contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path	string	true	"Vault ID"
//	@Param			contact_id	path	string	true	"Contact ID"
//	@Param			job_id		path	integer	true	"Job ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobs/{job_id} [delete]
func (h *ContactJobHandler) DeleteJob(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")
	jobID, err := strconv.ParseUint(c.Param("job_id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "err.invalid_job_id", nil)
	}

	if err := h.contactJobService.Delete(contactID, vaultID, uint(jobID)); err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		if errors.Is(err, services.ErrContactJobNotFound) {
			return response.NotFound(c, "err.job_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_job")
	}
	return response.NoContent(c)
}

// LegacyUpdate godoc
//
//	@Summary		Update contact job information (legacy)
//	@Description	Update the job information of a contact (backward compatible)
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string					true	"Vault ID"
//	@Param			contact_id	path		string					true	"Contact ID"
//	@Param			request		body		dto.UpdateJobInfoRequest	true	"Job details"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobInformation [put]
func (h *ContactJobHandler) LegacyUpdate(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	var req dto.UpdateJobInfoRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}

	contact, err := h.contactJobService.LegacyUpdate(contactID, vaultID, req)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_update_job_info")
	}
	return response.OK(c, contact)
}

// LegacyDelete godoc
//
//	@Summary		Delete contact job information (legacy)
//	@Description	Clear the job information of a contact (backward compatible)
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vault_id	path		string	true	"Vault ID"
//	@Param			contact_id	path		string	true	"Contact ID"
//	@Success		200			{object}	response.APIResponse
//	@Failure		401			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/vaults/{vault_id}/contacts/{contact_id}/jobInformation [delete]
func (h *ContactJobHandler) LegacyDelete(c echo.Context) error {
	contactID := c.Param("contact_id")
	vaultID := c.Param("vault_id")

	contact, err := h.contactJobService.LegacyDelete(contactID, vaultID)
	if err != nil {
		if errors.Is(err, services.ErrContactNotFound) {
			return response.NotFound(c, "err.contact_not_found")
		}
		return response.InternalError(c, "err.failed_to_delete_job_info")
	}
	return response.OK(c, contact)
}
