package utils

import (
	"testing"

	"github.com/naiba/bonds/internal/models"
)

func strPtr(s string) *string { return &s }

func TestBuildContactName(t *testing.T) {
	tests := []struct {
		name     string
		contact  *models.Contact
		expected string
	}{
		{
			name:     "first and last name",
			contact:  &models.Contact{FirstName: strPtr("John"), LastName: strPtr("Doe")},
			expected: "John Doe",
		},
		{
			name:     "first name only",
			contact:  &models.Contact{FirstName: strPtr("Alice")},
			expected: "Alice",
		},
		{
			name:     "last name only",
			contact:  &models.Contact{LastName: strPtr("Smith")},
			expected: "Smith",
		},
		{
			name:     "nickname fallback when no first or last",
			contact:  &models.Contact{Nickname: strPtr("Bobby")},
			expected: "Bobby",
		},
		{
			name:     "nickname ignored when first name present",
			contact:  &models.Contact{FirstName: strPtr("John"), Nickname: strPtr("Johnny")},
			expected: "John",
		},
		{
			name:     "nickname ignored when last name present",
			contact:  &models.Contact{LastName: strPtr("Doe"), Nickname: strPtr("Johnny")},
			expected: "Doe",
		},
		{
			name:     "nickname ignored when first and last present",
			contact:  &models.Contact{FirstName: strPtr("John"), LastName: strPtr("Doe"), Nickname: strPtr("JD")},
			expected: "John Doe",
		},
		{
			name:     "all nil pointers",
			contact:  &models.Contact{},
			expected: "",
		},
		{
			name:     "empty strings treated as absent",
			contact:  &models.Contact{FirstName: strPtr(""), LastName: strPtr(""), Nickname: strPtr("")},
			expected: "",
		},
		{
			name:     "empty first and last with valid nickname",
			contact:  &models.Contact{FirstName: strPtr(""), LastName: strPtr(""), Nickname: strPtr("Buddy")},
			expected: "Buddy",
		},
		{
			name:     "nil first with valid last and nickname",
			contact:  &models.Contact{LastName: strPtr("Doe"), Nickname: strPtr("Johnny")},
			expected: "Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildContactName(tt.contact)
			if got != tt.expected {
				t.Errorf("BuildContactName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatContactName(t *testing.T) {
	contact := &models.Contact{
		FirstName:  strPtr("James"),
		MiddleName: strPtr("Herbert"),
		LastName:   strPtr("Bond"),
		Nickname:   strPtr("007"),
		MaidenName: strPtr("Muller"),
		Prefix:     strPtr("Dr."),
		Suffix:     strPtr("III"),
	}

	tests := []struct {
		name     string
		template string
		contact  *models.Contact
		fallback string
		expected string
	}{
		{"first last", "%first_name% %last_name%", contact, "Unknown", "Dr. James Bond III"},
		{"last first", "%last_name%, %first_name%", contact, "Unknown", "Dr. Bond, James III"},
		{"conditional present", "%first_name% %last_name% {nickname? (%nickname%)}", contact, "Unknown", "Dr. James Bond (007) III"},
		{"conditional absent", "%first_name% %last_name% {nickname? (%nickname%)}", &models.Contact{FirstName: strPtr("Jane"), LastName: strPtr("Doe")}, "Unknown", "Jane Doe"},
		{"empty parentheses removed", "%first_name% %last_name% (%nickname%)", &models.Contact{FirstName: strPtr("Jane"), LastName: strPtr("Doe")}, "Unknown", "Jane Doe"},
		{"maiden name", "%first_name% (%maiden_name%) %last_name%", contact, "Unknown", "Dr. James (Muller) Bond III"},
		{"fallback", "%first_name%", &models.Contact{}, "Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatContactName(tt.template, tt.contact, tt.fallback)
			if got != tt.expected {
				t.Errorf("FormatContactName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
