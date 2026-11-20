package avatar

import (
	"bytes"
	"image/color"
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

// 两个具体名字可能恰好撞色，所以断言的是分布：一批不同的名字必须落在多种颜色上，
// 而且每个颜色都来自调色板。只打日志不断言的写法在取色逻辑退化成常量时也会通过。
func TestColorFromNameSpreadsNamesAcrossPalette(t *testing.T) {
	names := []string{"Alice", "Bob", "Carol", "Dave", "Erin", "Frank", "Grace", "Heidi", "Ivan", "Judy", "Mallory", "Niaj"}
	inPalette := make(map[color.RGBA]bool, len(palette))
	for _, c := range palette {
		inPalette[c] = true
	}

	used := make(map[color.RGBA]bool)
	for _, name := range names {
		c := colorFromName(name)
		if !inPalette[c] {
			t.Fatalf("colorFromName(%q) = %v, which is not a palette color", name, c)
		}
		used[c] = true
	}

	if len(used) < 3 {
		t.Fatalf("%d names mapped onto only %d color(s)", len(names), len(used))
	}
}
