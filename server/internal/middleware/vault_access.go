package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

const (
	PermissionManager = 100
	PermissionEditor  = 200
	PermissionViewer  = 300
)

type VaultAccessMiddleware struct {
	db *gorm.DB
}

func NewVaultAccessMiddleware(db *gorm.DB) *VaultAccessMiddleware {
	return &VaultAccessMiddleware{db: db}
}

type userVaultRow struct {
	VaultID    string `gorm:"column:vault_id"`
	UserID     string `gorm:"column:user_id"`
	ContactID  string `gorm:"column:contact_id"`
	Permission int    `gorm:"column:permission"`
}

func (userVaultRow) TableName() string { return "user_vault" }

func (m *VaultAccessMiddleware) RequireAccess(minPermission int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := GetUserID(c)
			vaultID := c.Param("vault_id")
			if vaultID == "" {
				vaultID = c.Param("id")
			}
			if vaultID == "" {
				return response.BadRequest(c, "err.vault_id_required", nil)
			}

			var row userVaultRow
			err := m.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&row).Error
			if err != nil {
				return response.Forbidden(c, "err.no_vault_access")
			}

			if row.Permission > minPermission {
				return response.Forbidden(c, "err.insufficient_permissions")
			}

			c.Set("vault_id", vaultID)
			c.Set("vault_permission", row.Permission)
			return next(c)
		}
	}
}

func GetVaultID(c echo.Context) string {
	id, _ := c.Get("vault_id").(string)
	if id == "" {
		return c.Param("vault_id")
	}
	return id
}

func GetVaultPermission(c echo.Context) int {
	perm, _ := c.Get("vault_permission").(int)
	return perm
}
