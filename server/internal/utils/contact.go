package utils

import (
	"strings"

	"github.com/naiba/bonds/internal/models"
)

func BuildContactName(c *models.Contact) string {
	var parts []string
	if c.FirstName != nil && *c.FirstName != "" {
		parts = append(parts, *c.FirstName)
	}
	if c.LastName != nil && *c.LastName != "" {
		parts = append(parts, *c.LastName)
	}
	if len(parts) == 0 && c.Nickname != nil && *c.Nickname != "" {
		parts = append(parts, *c.Nickname)
	}
	return strings.Join(parts, " ")
}
