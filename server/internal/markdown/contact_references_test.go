package markdown

import (
	"strings"
	"testing"
)

func TestContactReferencesFollowMarkdownRendering(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	link := "[Alice](contact:" + id + ")"
	for _, tc := range []struct {
		name, body string
		count      int
	}{
		{"bare", link, 1}, {"legacy", "@" + link, 1},
		{"inline_code", "`" + link + "`", 0}, {"legacy_code", "`@" + link + "`", 0},
		{"double_backticks", "`` ` " + link + " ``", 0},
		{"fenced_code", "```text\n" + link + "\n```", 0},
		{"tilde_fence", "~~~\n" + link + "\n~~~", 0},
		{"indented_code", "    " + link, 0},
		{"restored_before_indented", "@" + link + "\n\n    sample", 1},
		{"restored_before_unclosed_fence", "@" + link + "\n\n```text\nunclosed example", 1},
		{"escaped", `\` + link, 0},
		{"image", "!" + link, 0},
		{"image_link_label", "![example " + link + "](https://example.test/photo.png)", 0},
		{"html", "<div>\n" + link + "\n</div>", 0},
		{"mixed", link + " `" + link + "` \\" + link + " @" + link, 2},
		{"title", "[Alice](contact:" + id + ` "name")`, 1},
		{"reference", "[Alice][friend]\n\n[friend]: contact:" + id, 1},
		{"id_in_reference_label", "[contact:" + id + "]\n\n[contact:" + id + "]: contact:" + id, 1},
		{"unused_reference", "[friend]: contact:" + id, 0},
		{"autolink", "<contact:" + id + ">", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refs := ContactReferences(tc.body, FormatMarkdown)
			rendered := Render(tc.body, FormatMarkdown)
			if len(refs) != tc.count || strings.Count(rendered, `data-bonds-contact="`+id+`"`) != tc.count {
				t.Fatalf("refs=%+v rendered=%s", refs, rendered)
			}
			for _, ref := range refs {
				if ref.ID != id || tc.body[ref.Start:ref.End] != id {
					t.Fatalf("invalid source range: %+v", ref)
				}
			}
			if tc.name == "legacy_code" && !strings.Contains(rendered, "@[Alice]") {
				t.Fatalf("literal @ changed: %s", rendered)
			}
		})
	}
	// Plain text intentionally keeps its existing inline marker semantics, even
	// where Markdown would treat the same bytes as a code example.
	if refs := ContactReferences("`"+link+"`", FormatPlain); len(refs) != 1 {
		t.Fatalf("plain markers changed: %+v", refs)
	}
}

func TestEncodedContactReferencesFollowRendering(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	for _, destination := range []string{
		"contact:" + id, `contact\:` + id, "contact&#58;" + id,
		"contact&colon;" + id, "&#99;ontact:" + id,
		"contact:" + strings.ReplaceAll(id, "-", `\-`),
		"contact:&#53;" + id[1:], "contact:" + strings.ReplaceAll(id, "-", "&#x2d;"),
		"contact&#92;:" + id,
	} {
		for name, body := range map[string]string{
			"inline":                "[Alice](" + destination + ")",
			"nested_autolink_label": "[outer <" + destination + ">](https://example.test)",
			"footnote":              "Text[^friend]\n\n[^friend]: [Alice](" + destination + ")",
			"heart_label":           "[<3](" + destination + ")",
			"heart_reference_label": "[<3][friend]\n\n[friend]: " + destination,
			"preceding_less_than":   "<text>[Alice](" + destination + ")",
			"legacy":                "@[Alice](" + destination + ")",
			"angle":                 "[Alice](<" + destination + ">)",
			"reference":             "[Alice][friend]\n\n[friend]: " + destination,
			"autolink":              "<" + destination + ">",
			"code":                  "`[Alice](" + destination + ")`",
			"escaped":               `\[Alice](` + destination + ")",
			"fence":                 "```\n[Alice](" + destination + ")\n```",
			"image":                 "![Alice](" + destination + ")",
			"image_label":           "![sample [Alice](" + destination + ")](https://example.test/photo.png)",
			"unused":                "[friend]: " + destination,
		} {
			t.Run(destination+"/"+name, func(t *testing.T) {
				refs := ContactReferences(body, FormatMarkdown)
				rendered := Render(body, FormatMarkdown)
				count := strings.Count(rendered, `data-bonds-contact="`+id+`"`)
				if len(refs) != count {
					t.Fatalf("refs=%+v rendered=%s", refs, rendered)
				}
				for _, ref := range refs {
					if ref.ID != id {
						t.Fatalf("decoded ID=%q", ref.ID)
					}
					changed := body[:ref.Start] + "660e8400-e29b-41d4-a716-446655440000" + body[ref.End:]
					if !strings.Contains(Render(changed, FormatMarkdown), `data-bonds-contact="660e8400-e29b-41d4-a716-446655440000"`) {
						t.Fatalf("wrong source offsets %+v: %s", ref, changed)
					}
				}
			})
		}
	}
}
