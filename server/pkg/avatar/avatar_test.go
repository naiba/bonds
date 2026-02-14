package avatar

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateInitialsValidPNG(t *testing.T) {
	data := GenerateInitials("John Doe", 128)

	if len(data) == 0 {
		t.Fatal("Expected non-empty PNG data")
	}

	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Generated data is not valid PNG: %v", err)
	}
}

func TestGenerateInitialsSingleName(t *testing.T) {
	data := GenerateInitials("Alice", 64)

	if len(data) == 0 {
		t.Fatal("Expected non-empty PNG data for single name")
	}

	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Generated data is not valid PNG: %v", err)
	}
}

func TestGenerateInitialsEmptyName(t *testing.T) {
	data := GenerateInitials("", 128)

	if len(data) == 0 {
		t.Fatal("Expected non-empty PNG data for empty name")
	}

	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Generated data is not valid PNG: %v", err)
	}
}

func TestGenerateInitialsDefaultSize(t *testing.T) {
	data := GenerateInitials("Test User", 0)

	if len(data) == 0 {
		t.Fatal("Expected non-empty PNG data with default size")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Generated data is not valid PNG: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 128 {
		t.Errorf("Expected 128x128, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestExtractInitials(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"John Doe", "JD"},
		{"Alice", "A"},
		{"", "?"},
		{"  ", "?"},
		{"Mary Jane Watson", "MW"},
	}

	for _, tt := range tests {
		got := extractInitials(tt.name)
		if got != tt.expected {
			t.Errorf("extractInitials(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestColorFromNameDeterministic(t *testing.T) {
	c1 := colorFromName("John")
	c2 := colorFromName("John")

	if c1 != c2 {
		t.Error("Expected same color for same name")
	}
}

func TestColorFromNameDifferentNames(t *testing.T) {
	c1 := colorFromName("Alice")
	c2 := colorFromName("Bob")

	if c1 == c2 {
		t.Log("Different names produced same color (possible but unlikely for just two names)")
	}
}
