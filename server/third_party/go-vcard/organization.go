package vcard

import "strings"

type organizationValue struct {
	value      string
	components []string
}

// Organization returns the organization name followed by its optional units.
// Component boundaries must be decoded before text escapes: \\; and \; have
// different meanings even though generic parseValue collapses them identically.
func (f *Field) Organization() []string {
	if f == nil {
		return nil
	}
	if f.organization != nil && f.organization.value == f.Value {
		return append([]string(nil), f.organization.components...)
	}
	return parseOrganization(f.Value)
}

// NewOrganization constructs an ORG field from unescaped components.
func NewOrganization(components []string) *Field {
	value := strings.Join(components, ";")
	return &Field{Value: value, organization: &organizationValue{value: value, components: append([]string(nil), components...)}}
}

func parseOrganization(value string) []string {
	var result []string
	var component strings.Builder
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ';':
			result = append(result, component.String())
			component.Reset()
		case '\\':
			if i+1 < len(value) {
				switch value[i+1] {
				case '\\', ';', ',':
					i++
					component.WriteByte(value[i])
					continue
				case 'n', 'N':
					i++
					component.WriteByte('\n')
					continue
				}
			}
			component.WriteByte(value[i])
		default:
			component.WriteByte(value[i])
		}
	}
	return append(result, component.String())
}

func formatOrganization(components []string) string {
	escaped := make([]string, len(components))
	replacer := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\r\n", "\\n", "\n", "\\n", "\r", "\\n")
	for i, component := range components {
		escaped[i] = replacer.Replace(component)
	}
	return strings.Join(escaped, ";")
}
