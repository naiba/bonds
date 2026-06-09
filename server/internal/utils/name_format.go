package utils

import (
	"regexp"
	"strings"

	"github.com/naiba/bonds/internal/models"
)

var (
	nameConditionalBlockRegex = regexp.MustCompile(`\{([a-z_]+)\?\s*(.*?)\}`)
	nameVariableRegex         = regexp.MustCompile(`%([a-z_]+)%`)
	emptyParenthesesRegex     = regexp.MustCompile(`\(\s*\)`)
	whitespaceRegex           = regexp.MustCompile(`\s+`)
)

// FormatContactName formats a contact using the same name_order template rules
// as the frontend. The fallback is returned only when every resolved field is empty.
func FormatContactName(nameOrder string, contact *models.Contact, fallback string) string {
	if contact == nil {
		return fallback
	}

	fields := map[string]string{
		"first_name":  ptrToString(contact.FirstName),
		"last_name":   ptrToString(contact.LastName),
		"middle_name": ptrToString(contact.MiddleName),
		"nickname":    ptrToString(contact.Nickname),
		"maiden_name": ptrToString(contact.MaidenName),
	}

	result := nameConditionalBlockRegex.ReplaceAllStringFunc(nameOrder, func(match string) string {
		parts := nameConditionalBlockRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return ""
		}
		if strings.TrimSpace(fields[parts[1]]) == "" {
			return ""
		}
		return parts[2]
	})

	result = nameVariableRegex.ReplaceAllStringFunc(result, func(match string) string {
		parts := nameVariableRegex.FindStringSubmatch(match)
		if len(parts) != 2 {
			return ""
		}
		return fields[parts[1]]
	})

	result = emptyParenthesesRegex.ReplaceAllString(result, "")

	if prefix := strings.TrimSpace(ptrToString(contact.Prefix)); prefix != "" {
		result = prefix + " " + result
	}
	if suffix := strings.TrimSpace(ptrToString(contact.Suffix)); suffix != "" {
		result += " " + suffix
	}

	result = strings.TrimSpace(whitespaceRegex.ReplaceAllString(result, " "))
	if result == "" {
		return fallback
	}
	return result
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
