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
