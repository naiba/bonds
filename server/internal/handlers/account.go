package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

type AccountHandler struct {
	db *gorm.DB
}

func NewAccountHandler(db *gorm.DB) *AccountHandler {
	return &AccountHandler{db: db}
}

// GetAccount godoc
//
//	@Summary		Get current account
//	@Description	Return the current user's account details
//	@Tags			account
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/account [get]
func (h *AccountHandler) GetAccount(c echo.Context) error {
	accountID := middleware.GetAccountID(c)
	if accountID == "" {
		return response.Unauthorized(c, "err.invalid_account")
	}

	var account models.Account
	if err := h.db.First(&account, "id = ?", accountID).Error; err != nil {
		return response.NotFound(c, "err.account_not_found")
	}

	return response.OK(c, account)
}
