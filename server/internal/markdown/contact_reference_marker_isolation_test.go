package markdown

import (
	"strings"
	"testing"
)

func TestEncodedParsingMarkerDoesNotMakeCodeLive(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	body := "`[Example](contact:" + id + ")`\n\n[Unrelated](bonds&#45;contact-reference-0-contact:" + id + ")"
	rendered := Render(body, FormatMarkdown)
	if strings.Contains(rendered, "data-bonds-contact=") {
		t.Fatalf("fixture is live: %s", rendered)
	}
	refs := ContactReferences(body, FormatMarkdown)
	if len(refs) != 0 {
		t.Fatalf("noninteractive code was discovered as a live reference: refs=%+v body=%q rendered=%s", refs, body, rendered)
	}
}

func TestParsingMarkersCannotAliasDecodedDestinations(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	literal := "`[Example](contact:" + id + ")`"
	for _, encodedPrefix := range []string{"bonds&#45;contact-reference-", `bonds\-contact-reference-`, "&#98;onds-contact-reference-", "bonds&#x2d;contact-reference-"} {
		for _, kind := range []string{"inline", "reference", "multiple_prefixes"} {
			t.Run(encodedPrefix+kind, func(t *testing.T) {
				spoof := "[Unrelated](" + encodedPrefix + "0-contact:" + id + ")"
				if kind == "reference" {
					spoof = "[Unrelated][address]\n\n[address]: " + encodedPrefix + "0-contact:" + id
				}
				if kind == "multiple_prefixes" {
					spoof += "\n[Another](" + encodedPrefix + "x0-contact:" + id + ")"
				}
				for _, live := range []bool{false, true} {
					body := literal + "\n\n" + spoof
					want := 0
					if live {
						body += "\n\n[Live](contact:" + id + ")"
						want = 1
					}
					rendered := Render(body, FormatMarkdown)
					if strings.Count(rendered, "data-bonds-contact=") != want {
						t.Fatalf("invalid fixture: %s", rendered)
					}
					refs := ContactReferences(body, FormatMarkdown)
					if len(refs) != want {
						t.Fatalf("literal aliased a marker: refs=%+v body=%q", refs, body)
					}
					if live && (refs[0].ID != id || refs[0].Start != strings.LastIndex(body, "contact:"+id)+len("contact:") || refs[0].End != len(body)-1) {
						t.Fatalf("incorrect source range: %+v", refs)
					}
				}
			})
		}
	}
}
