package markdown

import (
	"strings"
	"testing"
)

func TestRenderMarkdownFeaturesAndCustomReferences(t *testing.T) {
	input := "# Day\n\n**bold**\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n@[Alice](contact:550e8400-e29b-41d4-a716-446655440000)\n\n![photo](bonds-file:12) [report](bonds-file:13)"
	got := Render(input, FormatMarkdown)
	for _, expected := range []string{
		"<h1>Day</h1>", "<strong>bold</strong>", "<table>",
		`data-bonds-contact="550e8400-e29b-41d4-a716-446655440000"`,
		`data-bonds-file="12"`, `data-bonds-kind="image"`,
		`data-bonds-file="13"`, `data-bonds-kind="file"`,
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("rendered HTML missing %q:\n%s", expected, got)
		}
	}
}

func TestRenderSanitizesHTMLAndDangerousProtocols(t *testing.T) {
	input := `<script>alert(1)</script> [bad](javascript:alert(2)) ![bad](data:text/html;base64,xxx) <img src=x onerror=alert(3)>`
	got := Render(input, FormatMarkdown)
	for _, forbidden := range []string{"<script", `href="javascript:`, `src="data:`, "<img src=x"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("rendered HTML contains %q: %s", forbidden, got)
		}
	}
}

func TestRenderPlainDoesNotReinterpretMarkdown(t *testing.T) {
	got := Render("# not a heading\n**not bold**", FormatPlain)
	if strings.Contains(got, "<h1>") || strings.Contains(got, "<strong>") {
		t.Fatalf("plain content was reinterpreted: %s", got)
	}
	if !strings.Contains(got, "# not a heading") || !strings.Contains(got, "**not bold**") {
		t.Fatalf("plain content changed: %s", got)
	}
}

func TestExtractFileIDs(t *testing.T) {
	got := ExtractFileIDs("![one](bonds-file:12 \"preview\") [two][report] [again](bonds-file:12) `[code](bonds-file:99)` [remote](https://example.com)\n\n[report]: bonds-file:13")
	if len(got) != 2 || got[0] != 12 || got[1] != 13 {
		t.Fatalf("ExtractFileIDs = %v, want [12 13]", got)
	}
}
