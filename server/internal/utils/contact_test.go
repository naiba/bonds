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
